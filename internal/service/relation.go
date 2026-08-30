package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"datacenter/internal/errno"
	"datacenter/internal/model"
	"datacenter/internal/store"
)

// 树查询/级联删除的深度与每层数量上限（Q6 已确认，可配置）
const (
	defaultMaxDepth   = 10
	defaultPerLevel   = 500
	defaultBadgeLimit = 50
)

// RelationService 数据项关联关系服务：边 CRUD、树、徽标、删除预检与策略化连锁删除
type RelationService struct {
	items     *store.ItemStore
	cols      *store.CollectionStore
	tasks     *store.TaskStore
	relations *store.RelationStore
	perm      *PermissionChecker
	audit     *store.AuditStore
	maxDepth  int
	perLevel  int
}

// NewRelationService 构造关联关系服务
func NewRelationService(items *store.ItemStore, cols *store.CollectionStore,
	tasks *store.TaskStore, users *store.UserStore, relations *store.RelationStore, audit *store.AuditStore) *RelationService {
	return &RelationService{
		items:     items,
		cols:      cols,
		tasks:     tasks,
		relations: relations,
		perm:      NewPermissionChecker(cols, users, audit),
		audit:     audit,
		maxDepth:  defaultMaxDepth,
		perLevel:  defaultPerLevel,
	}
}

// ---------- 视图类型 ----------

// RelationView 关系视图（含对端摘要）
type RelationView struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Direction string                 `json:"direction"` // out | in
	Target    ItemBrief              `json:"target"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
	CreatedAt string                 `json:"created_at"`
}

// ItemBrief 数据项摘要（关联对端展示用）
type ItemBrief struct {
	ItemID         string                 `json:"item_id"`
	Path           string                 `json:"path"`
	CollectionID   string                 `json:"collection_id"`
	CollectionName string                 `json:"collection_name,omitempty"`
	SummaryTags    map[string]interface{} `json:"summary_tags,omitempty"`
}

// TreeItem 树节点
type TreeItem struct {
	Item     ItemBrief   `json:"item"`
	Meta     interface{} `json:"meta,omitempty"`
	Children []*TreeItem `json:"children,omitempty"`
}

// DeleteImpact 删除预检结果（策略化删除判定）
type DeleteImpact struct {
	ItemID              string       `json:"item_id"`
	Policy              DeletePolicy `json:"policy"`
	Children            []ItemBrief  `json:"children"` // 子节点（子树，不含自身）
	Incoming            []ItemBrief  `json:"incoming"` // 被引用/调用方
	ChildrenDeny        bool         `json:"children_deny"`
	IncomingDeny        bool         `json:"incoming_deny"`
	WillCascade         bool         `json:"will_cascade"`
	WillDetachChildren  bool         `json:"will_detach_children"`
	WillDetachIncoming  bool         `json:"will_detach_incoming"`
	AffectedItemCount   int          `json:"affected_item_count"` // 将被删除的数据项数（含自身与级联子树）
	DetachIncomingCount int          `json:"detach_incoming_count"`
}

// DeletePolicy 策略判定视图
type DeletePolicy struct {
	Children string `json:"children"`
	Incoming string `json:"incoming"`
}

// ---------- 建边 ----------

// CreateRelationReq 建边请求
type CreateRelationReq struct {
	Type     string                 `json:"type" binding:"required"`
	ToItemID string                 `json:"to_item_id" binding:"required"`
	Meta     map[string]interface{} `json:"meta"`
}

// Create 建边：权限、存在性、自环、类型、单父、父链环检测
func (s *RelationService) Create(ctx context.Context, userID string, fromID primitive.ObjectID, req CreateRelationReq) (*model.Relation, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	from, err := s.items.FindByID(ctx, fromID)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if from == nil {
		return nil, errno.ErrItemNotFound
	}
	if _, err := s.perm.RequireRole(ctx, from.CollectionID, uid, model.MemberRoleOperator); err != nil {
		return nil, err
	}

	toID, err := primitive.ObjectIDFromHex(req.ToItemID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	if fromID == toID {
		return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "不能与自身建立关系")
	}
	if req.Type != model.RelationParentChild && req.Type != model.RelationReference && req.Type != model.RelationCall {
		return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "不支持的关联类型: "+req.Type)
	}
	to, err := s.items.FindByID(ctx, toID)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if to == nil {
		return nil, errno.ErrItemNotFound.WithCause(errors.New("目标数据项不存在"))
	}
	// 目标可访问（跨集合关系 Q5）
	if err := s.ensureAccessible(ctx, uid, to.CollectionID); err != nil {
		return nil, err
	}

	// 单父 + 环检测（Q1）
	if req.Type == model.RelationParentChild {
		if err := s.validateParentEdge(ctx, fromID, toID); err != nil {
			return nil, err
		}
	}

	r := &model.Relation{
		CollectionID: from.CollectionID,
		FromItemID:   fromID,
		ToItemID:     toID,
		Type:         req.Type,
		Meta:         req.Meta,
		CreatedBy:    uid,
	}
	if err := s.relations.Create(ctx, r); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, errno.New(errno.ErrConflict.Code, errno.ErrConflict.HTTPStatus, "该关系已存在")
		}
		return nil, errno.ErrInternal.WithCause(err)
	}
	// 物化路径维护（P3 优化）
	if req.Type == model.RelationParentChild {
		if err := s.rebuildAncestors(ctx, toID); err != nil {
			return nil, err
		}
	}
	s.audit.Log(ctx, uid, "relation.create", fmt.Sprintf("%s: %s -> %s", req.Type, fromID.Hex(), toID.Hex()), &from.CollectionID, &fromID)
	return r, nil
}

// CreateBatch 批量建边（Q7）：逐条校验，返回成功/失败明细
func (s *RelationService) CreateBatch(ctx context.Context, userID string, fromID primitive.ObjectID, reqs []CreateRelationReq) (map[string]interface{}, *errno.Error) {
	if len(reqs) == 0 {
		return nil, errno.ErrParam
	}
	if len(reqs) > 200 {
		return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "单次最多批量建立 200 条关系")
	}
	success := make([]map[string]interface{}, 0, len(reqs))
	failed := make([]map[string]interface{}, 0)
	for i, req := range reqs {
		r, err := s.Create(ctx, userID, fromID, req)
		if err != nil {
			failed = append(failed, map[string]interface{}{
				"index": i, "to_item_id": req.ToItemID, "type": req.Type, "error": err.Message,
			})
			continue
		}
		success = append(success, map[string]interface{}{"index": i, "relation_id": r.ID.Hex(), "to_item_id": req.ToItemID})
	}
	return map[string]interface{}{"success": success, "failed": failed}, nil
}

// validateParentEdge 单父与环检测：目标已有父 → 拒绝；新父的祖先链中包含子节点 → 拒绝（会成环）
func (s *RelationService) validateParentEdge(ctx context.Context, newParentID, childID primitive.ObjectID) *errno.Error {
	existing, err := s.relations.ParentEdge(ctx, childID)
	if err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	if existing != nil {
		return errno.New(errno.ErrConflict.Code, errno.ErrConflict.HTTPStatus, "该数据项已有父节点，不能重复设置（单父约束）")
	}
	// 新父的祖先链（沿父链上溯）：若链中包含子节点，说明子节点是新父的祖先 → 互为父子成环
	ancestors, err := s.relations.AncestorsOf(ctx, newParentID, s.maxDepth)
	if err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	for _, anc := range ancestors {
		if anc == childID {
			return errno.New(errno.ErrConflict.Code, errno.ErrConflict.HTTPStatus, "不能将自身或后代设为父节点（会形成环）")
		}
	}
	return nil
}

// ensureAccessible 校验目标集合可访问（成员或 admin）
func (s *RelationService) ensureAccessible(ctx context.Context, userID, collectionID primitive.ObjectID) *errno.Error {
	role, err := s.perm.MemberRole(ctx, collectionID, userID)
	if err == nil {
		_ = role
		return nil
	}
	if err.Code == errno.ErrNotMember.Code {
		return errno.ErrNotMember.WithCause(errors.New("目标数据项所在集合不可访问"))
	}
	return err
}

// ---------- 查询 ----------

// List 出边/入边分页列表（含对端摘要）
func (s *RelationService) List(ctx context.Context, userID string, itemID primitive.ObjectID, direction, relType string, page, pageSize int, isAdmin bool) ([]*RelationView, int64, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, errno.ErrParam.WithCause(err)
	}
	item, err := s.items.FindByID(ctx, itemID)
	if err != nil {
		return nil, 0, errno.ErrInternal.WithCause(err)
	}
	if item == nil {
		return nil, 0, errno.ErrItemNotFound
	}
	if !isAdmin {
		if _, err := s.perm.RequireRole(ctx, item.CollectionID, uid, model.MemberRoleOperator); err != nil {
			return nil, 0, err
		}
	}
	if direction != "out" && direction != "in" {
		return nil, 0, errno.ErrParam
	}

	filter := bson.M{}
	if direction == "out" {
		filter["from_item_id"] = itemID
	} else {
		filter["to_item_id"] = itemID
	}
	if relType != "" {
		filter["type"] = relType
	}
	rels, total, err := s.relations.List(ctx, filter, int64(page), int64(pageSize))
	if err != nil {
		return nil, 0, errno.ErrInternal.WithCause(err)
	}
	views, err := s.buildViews(ctx, rels, direction)
	if err != nil {
		return nil, 0, errno.ErrInternal.WithCause(err)
	}
	return views, total, nil
}

// buildViews 组装对端摘要
func (s *RelationService) buildViews(ctx context.Context, rels []*model.Relation, direction string) ([]*RelationView, error) {
	views := make([]*RelationView, 0, len(rels))
	if len(rels) == 0 {
		return views, nil
	}
	// 收集对端 id + 集合名
	peerIDs := make([]primitive.ObjectID, 0, len(rels))
	colIDs := make([]primitive.ObjectID, 0, len(rels))
	for _, r := range rels {
		if direction == "out" {
			peerIDs = append(peerIDs, r.ToItemID)
		} else {
			peerIDs = append(peerIDs, r.FromItemID)
		}
		colIDs = append(colIDs, r.CollectionID)
	}
	peers, err := s.items.FindByIDs(ctx, peerIDs)
	if err != nil {
		return nil, err
	}
	colName := s.collectionNames(ctx, colIDs)
	peerMap := make(map[string]*model.DataItem, len(peers))
	for _, p := range peers {
		peerMap[p.ID.Hex()] = p
	}
	for _, r := range rels {
		var peerID primitive.ObjectID
		if direction == "out" {
			peerID = r.ToItemID
		} else {
			peerID = r.FromItemID
		}
		p := peerMap[peerID.Hex()]
		brief := ItemBrief{ItemID: peerID.Hex()}
		if p != nil {
			brief.Path = p.Path
			brief.CollectionID = p.CollectionID.Hex()
			brief.CollectionName = colName[p.CollectionID.Hex()]
			brief.SummaryTags = s.summaryTags(p)
		}
		views = append(views, &RelationView{
			ID:        r.ID.Hex(),
			Type:      r.Type,
			Direction: direction,
			Target:    brief,
			Meta:      r.Meta,
			CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return views, nil
}

// Tree 层级树（懒加载友好：direction=desc 返回子树嵌套结构，asc 返回祖先链）
func (s *RelationService) Tree(ctx context.Context, userID string, itemID primitive.ObjectID, direction string, depth int, relType string, isAdmin bool) (interface{}, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	item, err := s.items.FindByID(ctx, itemID)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if item == nil {
		return nil, errno.ErrItemNotFound
	}
	if !isAdmin {
		if _, err := s.perm.RequireRole(ctx, item.CollectionID, uid, model.MemberRoleOperator); err != nil {
			return nil, err
		}
	}
	if depth < 1 || depth > s.maxDepth {
		depth = s.maxDepth
	}
	if relType == "" {
		relType = model.RelationParentChild
	}

	if direction == "asc" {
		chain, err := s.relations.AncestorsOf(ctx, itemID, depth)
		if err != nil {
			return nil, errno.ErrInternal.WithCause(err)
		}
		items, err := s.items.FindByIDs(ctx, chain)
		if err != nil {
			return nil, errno.ErrInternal.WithCause(err)
		}
		colName := s.collectionNames(ctx, idsOfItems(items))
		tree := make([]*TreeItem, 0, len(items))
		for i := len(items) - 1; i >= 0; i-- { // 根 → 近
			p := items[i]
			tree = append(tree, &TreeItem{Item: s.itemBrief(p, colName)})
		}
		return tree, nil
	}

	// desc：BFS 构建嵌套子树
	root, err := s.buildTreeLevel(ctx, item, relType, depth)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	return root, nil
}

func (s *RelationService) buildTreeLevel(ctx context.Context, item *model.DataItem, relType string, depth int) (*TreeItem, error) {
	colName := s.collectionNames(ctx, []primitive.ObjectID{item.CollectionID})
	node := &TreeItem{Item: s.itemBrief(item, colName)}
	if depth <= 0 {
		return node, nil
	}
	children, _, err := s.relations.List(ctx, bson.M{"type": relType, "from_item_id": item.ID}, 1, int64(s.perLevel))
	if err != nil {
		return nil, err
	}
	peerIDs := make([]primitive.ObjectID, 0, len(children))
	for _, c := range children {
		peerIDs = append(peerIDs, c.ToItemID)
	}
	peers, err := s.items.FindByIDs(ctx, peerIDs)
	if err != nil {
		return nil, err
	}
	colIDs := make([]primitive.ObjectID, 0, len(peers))
	for _, p := range peers {
		colIDs = append(colIDs, p.CollectionID)
	}
	colNameMap := s.collectionNames(ctx, colIDs)
	for _, c := range children {
		var childItem *model.DataItem
		for _, p := range peers {
			if p.ID == c.ToItemID {
				childItem = p
				break
			}
		}
		if childItem == nil {
			continue
		}
		sub, err := s.buildTreeLevel(ctx, childItem, relType, depth-1)
		if err != nil {
			return nil, err
		}
		sub.Item = s.itemBrief(childItem, colNameMap)
		if len(c.Meta) > 0 {
			sub.Meta = c.Meta
		}
		node.Children = append(node.Children, sub)
	}
	return node, nil
}

// Badges 批量关联数徽标（数据查询结果行）
func (s *RelationService) Badges(ctx context.Context, userID string, itemIDs []primitive.ObjectID) (map[string]map[string]int64, *errno.Error) {
	if len(itemIDs) == 0 {
		return map[string]map[string]int64{}, nil
	}
	if len(itemIDs) > defaultBadgeLimit {
		return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, fmt.Sprintf("单次最多 %d 个数据项", defaultBadgeLimit))
	}
	out := make(map[string]map[string]int64, len(itemIDs))
	for _, id := range itemIDs {
		out[id.Hex()] = map[string]int64{"out": 0, "in": 0}
	}
	o, err := s.relations.Count(ctx, bson.M{"from_item_id": bson.M{"$in": itemIDs}})
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	i, err := s.relations.Count(ctx, bson.M{"to_item_id": bson.M{"$in": itemIDs}})
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	_ = o
	_ = i
	// 按 item 粒度统计（一次聚合）
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"$or": []bson.M{
			{"from_item_id": bson.M{"$in": itemIDs}},
			{"to_item_id": bson.M{"$in": itemIDs}},
		}}}},
		{{Key: "$project", Value: bson.M{
			"item": bson.M{"$cond": []interface{}{bson.M{"$in": []interface{}{"$from_item_id", itemIDs}}, "$from_item_id", "$to_item_id"}},
			"dir":  bson.M{"$cond": []interface{}{bson.M{"$in": []interface{}{"$from_item_id", itemIDs}}, "out", "in"}},
		}}},
		{{Key: "$group", Value: bson.M{"_id": bson.M{"item": "$item", "dir": "$dir"}, "n": bson.M{"$sum": 1}}}},
	}
	cursor, err := s.relations.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var row struct {
			ID struct {
				Item primitive.ObjectID `bson:"item"`
				Dir  string             `bson:"dir"`
			} `bson:"_id"`
			N int64 `bson:"n"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, errno.ErrInternal.WithCause(err)
		}
		if m, ok := out[row.ID.Item.Hex()]; ok {
			m[row.ID.Dir] = row.N
		}
	}
	if err := cursor.Err(); err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	return out, nil
}

// ---------- 改/删边 ----------

// UpdateMeta 修改边属性
func (s *RelationService) UpdateMeta(ctx context.Context, userID string, relationID primitive.ObjectID, meta map[string]interface{}) (*model.Relation, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	r, err := s.relations.FindByID(ctx, relationID)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if r == nil {
		return nil, errno.ErrNotFound.WithCause(errors.New("关系不存在"))
	}
	if _, err := s.perm.RequireRole(ctx, r.CollectionID, uid, model.MemberRoleOperator); err != nil {
		return nil, err
	}
	if err := s.relations.UpdateMeta(ctx, relationID, meta); err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	s.audit.Log(ctx, uid, "relation.update", "更新关系 "+relationID.Hex(), &r.CollectionID, &r.FromItemID)
	updated, err := s.relations.FindByID(ctx, relationID)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	return updated, nil
}

// Delete 删除单条边（任一端所在集合成员即可）
func (s *RelationService) Delete(ctx context.Context, userID string, relationID primitive.ObjectID) *errno.Error {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errno.ErrParam.WithCause(err)
	}
	r, err := s.relations.FindByID(ctx, relationID)
	if err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	if r == nil {
		return errno.ErrNotFound.WithCause(errors.New("关系不存在"))
	}
	// 任一端所在集合可操作
	if err := s.ensureEitherEndAccessible(ctx, uid, r); err != nil {
		return err
	}
	if err := s.relations.DeleteByID(ctx, relationID); err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	// 父边删除：子变孤儿，重建物化路径
	if r.Type == model.RelationParentChild {
		if err := s.rebuildAncestors(ctx, r.ToItemID); err != nil {
			return errno.ErrInternal.WithCause(err)
		}
	}
	s.audit.Log(ctx, uid, "relation.delete", "删除关系 "+relationID.Hex(), &r.CollectionID, &r.FromItemID)
	return nil
}

func (s *RelationService) ensureEitherEndAccessible(ctx context.Context, uid primitive.ObjectID, r *model.Relation) *errno.Error {
	if _, err := s.perm.RequireRole(ctx, r.CollectionID, uid, model.MemberRoleOperator); err == nil {
		return nil
	}
	to, err := s.items.FindByID(ctx, r.ToItemID)
	if err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	if to != nil {
		if _, err := s.perm.RequireRole(ctx, to.CollectionID, uid, model.MemberRoleOperator); err == nil {
			return nil
		}
	}
	return errno.ErrNoPermission
}

// ---------- 删除预检与策略化删除 ----------

// Impact 删除预检：返回关联影响清单与策略判定（不执行删除）
func (s *RelationService) Impact(ctx context.Context, userID string, itemID primitive.ObjectID, isAdmin bool) (*DeleteImpact, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	item, col, e := s.loadWithPolicy(ctx, itemID, uid, isAdmin)
	if e != nil {
		return nil, e
	}
	impact, _, _, e := s.collectImpact(ctx, item, col, false, false)
	if e != nil {
		return nil, e
	}
	return impact, nil
}

// DeleteItem 策略化删除数据项（含级联/解除/预检，dry_run 只返回不执行）
func (s *RelationService) DeleteItem(ctx context.Context, userID string, itemID primitive.ObjectID, cascade, force, dryRun bool) (*DeleteImpact, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	item, col, e := s.loadWithPolicy(ctx, itemID, uid, false) // 删除永远要求集合角色
	if e != nil {
		return nil, e
	}
	impact, childrenIDs, _, e := s.collectImpact(ctx, item, col, cascade, force)
	if e != nil {
		return nil, e
	}
	if dryRun {
		return impact, nil
	}
	// 策略拒绝
	if impact.ChildrenDeny && !cascade {
		return impact, errno.New(errno.ErrConflict.Code, errno.ErrConflict.HTTPStatus,
			fmt.Sprintf("该数据项有 %d 个子节点，集合策略拒绝直接删除；请先处理子节点或使用 cascade=1 级联删除", len(impact.Children)))
	}
	if impact.IncomingDeny && !force {
		return impact, errno.New(errno.ErrConflict.Code, errno.ErrConflict.HTTPStatus,
			fmt.Sprintf("该数据项被 %d 处引用/调用，集合策略拒绝直接删除；请先解除引用或使用 force=1 强制删除", len(impact.Incoming)))
	}

	// 执行：先删边再删数据项（引用完整性）
	if impact.WillCascade {
		if err := s.relations.DeleteByItems(ctx, childrenIDs); err != nil {
			return impact, errno.ErrInternal.WithCause(err)
		}
	} else {
		if err := s.relations.DeleteByItems(ctx, []primitive.ObjectID{itemID}); err != nil {
			return impact, errno.ErrInternal.WithCause(err)
		}
	}
	if impact.WillDetachChildren {
		// detach 策略：子节点保留，仅删除父子边（上面 DeleteByItems([itemID]) 已删除出边）
		// 子节点变孤儿：重建物化路径
		childEdges, err := s.relations.ChildrenOf(ctx, itemID)
		if err != nil {
			return impact, errno.ErrInternal.WithCause(err)
		}
		for _, ce := range childEdges {
			if err := s.rebuildAncestors(ctx, ce.ToItemID); err != nil {
				return impact, errno.ErrInternal.WithCause(err)
			}
		}
	}
	// 删除数据项与刮削任务
	toDelete := childrenIDs
	if !impact.WillCascade {
		toDelete = []primitive.ObjectID{itemID}
	}
	if err := s.tasks.DeleteByItems(ctx, toDelete); err != nil {
		return impact, errno.ErrInternal.WithCause(err)
	}
	if err := s.items.DeleteByIDs(ctx, toDelete); err != nil {
		return impact, errno.ErrInternal.WithCause(err)
	}
	s.audit.Log(ctx, uid, "item.delete_cascade",
		fmt.Sprintf("删除数据项 %s，影响 %d 个数据项（级联=%v force=%v），解除入边 %d 条",
			item.Path, len(toDelete), impact.WillCascade, force, len(impact.Incoming)),
		&item.CollectionID, &itemID)
	return impact, nil
}

// collectImpact 收集影响清单并做策略判定
func (s *RelationService) collectImpact(ctx context.Context, item *model.DataItem, col *model.BusinessCollection, cascade, force bool) (*DeleteImpact, []primitive.ObjectID, []*model.Relation, *errno.Error) {
	policy := col.EffectiveDeletePolicy()

	// 子节点（子树）
	descIDs, err := s.relations.DescendantIDs(ctx, item.ID, s.maxDepth)
	if err != nil {
		return nil, nil, nil, errno.ErrInternal.WithCause(err)
	}
	childIDs := descIDs[1:] // 不含自身
	children := []ItemBrief{}
	if len(childIDs) > 0 {
		peers, err := s.items.FindByIDs(ctx, childIDs)
		if err != nil {
			return nil, nil, nil, errno.ErrInternal.WithCause(err)
		}
		colName := s.collectionNames(ctx, idsOfItems(peers))
		for _, p := range peers {
			children = append(children, s.itemBrief(p, colName))
		}
	}

	// 入边（被引用/调用）
	incomingEdges, _, err := s.relations.List(ctx, bson.M{"to_item_id": item.ID}, 1, int64(s.perLevel))
	if err != nil {
		return nil, nil, nil, errno.ErrInternal.WithCause(err)
	}
	incoming := []ItemBrief{}
	fromIDs := make([]primitive.ObjectID, 0, len(incomingEdges))
	for _, e := range incomingEdges {
		fromIDs = append(fromIDs, e.FromItemID)
	}
	if len(fromIDs) > 0 {
		peers, err := s.items.FindByIDs(ctx, fromIDs)
		if err != nil {
			return nil, nil, nil, errno.ErrInternal.WithCause(err)
		}
		colName := s.collectionNames(ctx, idsOfItems(peers))
		for _, p := range peers {
			brief := s.itemBrief(p, colName)
			brief.SummaryTags = nil
			incoming = append(incoming, brief)
		}
	}

	impact := &DeleteImpact{
		ItemID:              item.ID.Hex(),
		Policy:              DeletePolicy{Children: policy.Children, Incoming: policy.Incoming},
		Children:            children,
		Incoming:            incoming,
		DetachIncomingCount: len(incomingEdges),
	}
	// 子节点策略
	if len(childIDs) > 0 {
		switch policy.Children {
		case model.PolicyCascade:
			impact.WillCascade = true
		case model.PolicyDetach:
			impact.WillDetachChildren = true
		default: // deny
			impact.ChildrenDeny = true
			if cascade {
				impact.WillCascade = true
			}
		}
	}
	// 入边策略
	if len(incomingEdges) > 0 {
		switch policy.Incoming {
		case model.PolicyDetach:
			impact.WillDetachIncoming = true
		default: // deny
			impact.IncomingDeny = true
			if force {
				impact.WillDetachIncoming = true
			}
		}
	}
	if impact.WillCascade {
		impact.AffectedItemCount = len(descIDs)
	} else {
		impact.AffectedItemCount = 1
	}
	return impact, descIDs, incomingEdges, nil
}

func (s *RelationService) loadWithPolicy(ctx context.Context, itemID primitive.ObjectID, uid primitive.ObjectID, isAdmin bool) (*model.DataItem, *model.BusinessCollection, *errno.Error) {
	item, err := s.items.FindByID(ctx, itemID)
	if err != nil {
		return nil, nil, errno.ErrInternal.WithCause(err)
	}
	if item == nil {
		return nil, nil, errno.ErrItemNotFound
	}
	if isAdmin {
		col, err := s.cols.FindByID(ctx, item.CollectionID)
		if err != nil {
			return nil, nil, errno.ErrInternal.WithCause(err)
		}
		return item, col, nil
	}
	col, e := s.perm.RequireRole(ctx, item.CollectionID, uid, model.MemberRoleOperator)
	if e != nil {
		return nil, nil, e
	}
	return item, col, nil
}

// ---------- 物化路径维护（P3 优化） ----------

// rebuildAncestors 重建某节点及其后代的物化路径（父链）
func (s *RelationService) rebuildAncestors(ctx context.Context, itemID primitive.ObjectID) *errno.Error {
	ancestors, err := s.relations.AncestorsOf(ctx, itemID, s.maxDepth)
	if err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	if err := s.items.UpdateFields(ctx, itemID, bson.M{"ancestors": ancestors}); err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	// 逐层更新后代
	type nodeAnc struct {
		id  primitive.ObjectID
		anc []primitive.ObjectID
	}
	level := []nodeAnc{{id: itemID, anc: ancestors}}
	for depth := 0; len(level) > 0 && depth < s.maxDepth; depth++ {
		levelIDs := make([]primitive.ObjectID, 0, len(level))
		ancByID := make(map[string][]primitive.ObjectID, len(level))
		for _, n := range level {
			levelIDs = append(levelIDs, n.id)
			ancByID[n.id.Hex()] = n.anc
		}
		edges, _, err := s.relations.List(ctx, bson.M{"type": model.RelationParentChild, "from_item_id": bson.M{"$in": levelIDs}}, 1, int64(s.perLevel))
		if err != nil {
			return errno.ErrInternal.WithCause(err)
		}
		var next []nodeAnc
		for _, e := range edges {
			parentAnc := ancByID[e.FromItemID.Hex()]
			newAnc := append(append([]primitive.ObjectID{}, parentAnc...), e.FromItemID)
			if err := s.items.UpdateFields(ctx, e.ToItemID, bson.M{"ancestors": newAnc}); err != nil {
				return errno.ErrInternal.WithCause(err)
			}
			next = append(next, nodeAnc{id: e.ToItemID, anc: newAnc})
		}
		level = next
	}
	return nil
}

// ---------- 工具 ----------

func (s *RelationService) collectionNames(ctx context.Context, colIDs []primitive.ObjectID) map[string]string {
	out := make(map[string]string, len(colIDs))
	if len(colIDs) == 0 {
		return out
	}
	unique := make([]primitive.ObjectID, 0, len(colIDs))
	seen := map[string]bool{}
	for _, id := range colIDs {
		if !seen[id.Hex()] {
			seen[id.Hex()] = true
			unique = append(unique, id)
		}
	}
	for _, id := range unique {
		c, err := s.cols.FindByID(ctx, id)
		if err == nil && c != nil {
			out[id.Hex()] = c.Name
		}
	}
	return out
}

func (s *RelationService) itemBrief(item *model.DataItem, colName map[string]string) ItemBrief {
	b := ItemBrief{
		ItemID:         item.ID.Hex(),
		Path:           item.Path,
		CollectionID:   item.CollectionID.Hex(),
		CollectionName: colName[item.CollectionID.Hex()],
	}
	b.SummaryTags = s.summaryTags(item)
	return b
}

func (s *RelationService) summaryTags(item *model.DataItem) map[string]interface{} {
	if item == nil || len(item.Tags) == 0 {
		return nil
	}
	// 摘要取前 3 个标签
	keys := make([]string, 0, len(item.Tags))
	for k := range item.Tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := map[string]interface{}{}
	for i := 0; i < len(keys) && i < 3; i++ {
		out[keys[i]] = item.Tags[keys[i]]
	}
	return out
}

func idsOfItems(items []*model.DataItem) []primitive.ObjectID {
	ids := make([]primitive.ObjectID, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	return ids
}
