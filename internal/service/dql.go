package service

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"datacenter/internal/dql"
	"datacenter/internal/errno"
	"datacenter/internal/model"
	"datacenter/internal/store"
)

// DQLService 跨集合 DQL 查询服务
type DQLService struct {
	cols      *store.CollectionStore
	items     *store.ItemStore
	relations *store.RelationStore
}

// NewDQLService 构造 DQL 查询服务
func NewDQLService(cols *store.CollectionStore, items *store.ItemStore, relations *store.RelationStore) *DQLService {
	return &DQLService{cols: cols, items: items, relations: relations}
}

// Query 执行 DQL 查询：
//   - 支持 AND/OR（AND 优先）、括号、=、!=、>、>=、<、<=、IN、EXISTS、LIKE；
//   - collection 关键字限定集合（=、!=、IN），不写则查询用户有权访问的全部集合；
//   - ORDER BY field ASC|DESC 排序（int/float/date 标签，系统优化 1.2）；
//   - 普通用户仅能查询自己参与的集合（admin 全部）。
func (s *DQLService) Query(ctx context.Context, userID string, isAdmin bool, dqlStr string, page, pageSize int) ([]*model.DataItem, int64, *errno.Error) {
	// 提取 ORDER BY（系统优化 1.2）
	cleanDQL, orderBy, err := dql.ExtractOrderBy(dqlStr)
	if err != nil {
		return nil, 0, dqlErr(err)
	}
	node, err := dql.Parse(cleanDQL)
	if err != nil {
		return nil, 0, dqlErr(err)
	}

	targetIDs, negIDs, schemas, e := s.resolveScope(ctx, userID, isAdmin, node)
	if e != nil {
		return nil, 0, e
	}
	if len(targetIDs) == 0 {
		return []*model.DataItem{}, 0, nil
	}
	filter, e := s.buildFilter(ctx, node, targetIDs, negIDs, schemas)
	if e != nil {
		return nil, 0, e
	}

	// ORDER BY 排序（系统优化 1.2）：字段必须为全部目标集合中的 int/float/date 标签
	sort := bson.D{{Key: "created_at", Value: -1}}
	if orderBy != nil {
		if !orderFieldNumericOrDate(schemas, orderBy.Field) {
			return nil, 0, dqlErr(fmt.Errorf("排序字段 %s 不存在或仅支持 int/float/date 类型", orderBy.Field))
		}
		dir := 1
		if orderBy.Desc {
			dir = -1
		}
		sort = bson.D{{Key: "tags." + orderBy.Field, Value: dir}}
	}

	items, total, err := s.items.ListWithSort(ctx, filter, sort, int64(page), int64(pageSize))
	if err != nil {
		return nil, 0, errno.ErrInternal.WithCause(err)
	}
	return items, total, nil
}

// resolveScope 解析用户可访问集合范围与 collection 限定，返回目标集合 id（正向）、
// 排除集合 id、各集合标签定义（Query / Aggregate 共用）。
func (s *DQLService) resolveScope(ctx context.Context, userID string, isAdmin bool, node dql.Node) ([]primitive.ObjectID, []primitive.ObjectID, map[string][]model.TagDefinition, *errno.Error) {
	refs, err := dql.ExtractCollections(node)
	if err != nil {
		return nil, nil, nil, dqlErr(err)
	}
	colFilter := bson.M{}
	var uid primitive.ObjectID
	if !isAdmin {
		uid, err = primitive.ObjectIDFromHex(userID)
		if err != nil {
			return nil, nil, nil, errno.ErrParam.WithCause(err)
		}
		colFilter["members.user_id"] = uid
	}
	allCols, _, err := s.cols.List(ctx, colFilter, 1, 100000)
	if err != nil {
		return nil, nil, nil, errno.ErrInternal.WithCause(err)
	}
	byName := make(map[string]*model.BusinessCollection, len(allCols))
	for _, c := range allCols {
		byName[c.Name] = c
	}
	var posIDs, negIDs []primitive.ObjectID
	for _, r := range refs {
		for _, name := range r.Names {
			c, ok := byName[name]
			if !ok {
				return nil, nil, nil, errno.ErrCollectionNotFound.WithCause(fmt.Errorf("集合不存在: %s", name))
			}
			if !isAdmin && !c.IsMember(uid) {
				return nil, nil, nil, errno.ErrNotMember.WithCause(fmt.Errorf("无权限访问集合: %s", name))
			}
			if r.Op == "!=" {
				negIDs = append(negIDs, c.ID)
			} else {
				posIDs = append(posIDs, c.ID)
			}
		}
	}
	targetIDs := posIDs
	if len(refs) == 0 || len(posIDs) == 0 {
		for _, c := range allCols {
			targetIDs = append(targetIDs, c.ID)
		}
	}
	schemas := make(map[string][]model.TagDefinition, len(allCols))
	for _, c := range allCols {
		schemas[c.ID.Hex()] = c.TagSchema
	}
	return targetIDs, negIDs, schemas, nil
}

// buildFilter 构建完整过滤条件（标签条件 + collection 范围 + 关联关系限定）
func (s *DQLService) buildFilter(ctx context.Context, node dql.Node, targetIDs, negIDs []primitive.ObjectID, schemas map[string][]model.TagDefinition) (bson.M, *errno.Error) {
	filter, err := dql.BuildFilter(node, schemas)
	if err != nil {
		return nil, dqlErr(err)
	}
	// 纯 collection/parent/ancestor 条件（无标签条件）时 BuildFilter 返回 nil，需初始化为空过滤
	if filter == nil {
		filter = bson.M{}
	}
	colCond := bson.M{"$in": toIfaceSlice(targetIDs)}
	if len(negIDs) > 0 {
		colCond["$nin"] = toIfaceSlice(negIDs)
	}
	filter["collection_id"] = colCond
	// 关联关系限定（v2：parent 直接子 / ancestor 子树）
	if err := s.applyRelationRefs(ctx, node, filter); err != nil {
		return nil, err
	}
	return filter, nil
}

// AggregateReq 聚合请求（系统优化 1.2）
type AggregateReq struct {
	DQL     string `json:"dql"`
	GroupBy string `json:"group_by" binding:"required"`
	Limit   int    `json:"limit"`
}

// AggregateResult 聚合结果项
type AggregateResult struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// Aggregate 按标签字段分组统计（DQL 范围 + GROUP BY，返回分布直方图，最多 limit 组）
func (s *DQLService) Aggregate(ctx context.Context, userID string, isAdmin bool, req AggregateReq) ([]*AggregateResult, *errno.Error) {
	limit := req.Limit
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	node, err := dql.Parse(req.DQL)
	if err != nil {
		return nil, dqlErr(err)
	}
	targetIDs, negIDs, schemas, e := s.resolveScope(ctx, userID, isAdmin, node)
	if e != nil {
		return nil, e
	}
	if len(targetIDs) == 0 {
		return []*AggregateResult{}, nil
	}
	// group_by 校验：全部目标集合中存在且为 string/enum
	if !groupFieldStringLike(schemas, req.GroupBy) {
		return nil, dqlErr(fmt.Errorf("分组字段 %s 不存在或仅支持 string/enum 类型", req.GroupBy))
	}
	filter, e := s.buildFilter(ctx, node, targetIDs, negIDs, schemas)
	if e != nil {
		return nil, e
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$tags." + req.GroupBy},
			{Key: "n", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "n", Value: -1}}}},
		{{Key: "$limit", Value: limit}},
	}
	cursor, err := s.items.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	defer cursor.Close(ctx)
	out := make([]*AggregateResult, 0, limit)
	for cursor.Next(ctx) {
		var row struct {
			ID interface{} `bson:"_id"`
			N  int64       `bson:"n"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, errno.ErrInternal.WithCause(err)
		}
		out = append(out, &AggregateResult{Value: fmt.Sprintf("%v", row.ID), Count: row.N})
	}
	if err := cursor.Err(); err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	return out, nil
}

// groupFieldStringLike 校验分组字段在全部目标集合中存在且为 string/enum
func groupFieldStringLike(schemas map[string][]model.TagDefinition, field string) bool {
	found := false
	for _, schema := range schemas {
		for _, t := range schema {
			if t.Name != field {
				continue
			}
			found = true
			switch t.Type {
			case model.TagTypeString, model.TagTypeEnum:
			default:
				return false
			}
		}
	}
	return found
}

// applyRelationRefs 解析 parent/ancestor 条件，将命中数据项 id 合入过滤条件（交集语义：多个 ref 取并集）
func (s *DQLService) applyRelationRefs(ctx context.Context, node dql.Node, filter bson.M) *errno.Error {
	refs, err := dql.ExtractRelationRefs(node)
	if err != nil {
		return dqlErr(err)
	}
	if len(refs) == 0 {
		return nil
	}
	ids := map[string]primitive.ObjectID{}
	for _, r := range refs {
		for _, v := range r.Values {
			oid, err := primitive.ObjectIDFromHex(v)
			if err != nil {
				return dqlErr(fmt.Errorf("%s 需要合法数据项 id，得到 %q", r.Field, v))
			}
			switch r.Field {
			case "parent":
				// 直接子节点：parent_child 入边 = oid
				edges, _, err := s.relations.List(ctx, bson.M{"type": model.RelationParentChild, "to_item_id": oid}, 1, 1000)
				if err != nil {
					return errno.ErrInternal.WithCause(err)
				}
				for _, e := range edges {
					ids[e.FromItemID.Hex()] = e.FromItemID
				}
			case "ancestor":
				// 子树内：物化路径 ancestors 包含 oid
				items, err := s.items.FindByAncestor(ctx, oid)
				if err != nil {
					return errno.ErrInternal.WithCause(err)
				}
				for _, it := range items {
					ids[it.ID.Hex()] = it.ID
				}
			}
		}
	}
	if len(ids) == 0 {
		// 无命中：返回空结果
		filter["_id"] = bson.M{"$in": []interface{}{}}
		return nil
	}
	filter["_id"] = bson.M{"$in": toIfaceSliceOf(ids)}
	return nil
}

func toIfaceSliceOf(ids map[string]primitive.ObjectID) []interface{} {
	out := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		out = append(out, id)
	}
	return out
}

// orderFieldNumericOrDate 校验排序字段在全部目标集合中存在且类型均为 int/float/date
func orderFieldNumericOrDate(schemas map[string][]model.TagDefinition, field string) bool {
	found := false
	for _, schema := range schemas {
		for _, t := range schema {
			if t.Name != field {
				continue
			}
			found = true
			switch t.Type {
			case model.TagTypeInt, model.TagTypeFloat, model.TagTypeDate:
			default:
				return false
			}
		}
	}
	return found
}

func dqlErr(err error) *errno.Error {
	return errno.New(errno.ErrDQLInvalid.Code, errno.ErrDQLInvalid.HTTPStatus, "DQL 语句不合法: "+err.Error())
}

func toIfaceSlice(ids []primitive.ObjectID) []interface{} {
	out := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		out = append(out, id)
	}
	return out
}
