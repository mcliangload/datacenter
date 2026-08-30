package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"datacenter/internal/errno"
	"datacenter/internal/model"
	"datacenter/internal/store"
)

// ItemService 数据项服务（集合级权限逐集合判定）
type ItemService struct {
	items    *store.ItemStore
	cols     *store.CollectionStore
	tasks    *store.TaskStore
	perm     *PermissionChecker
	dataRoot string
	audit    *store.AuditStore
}

// NewItemService 构造数据项服务
func NewItemService(items *store.ItemStore, cols *store.CollectionStore,
	tasks *store.TaskStore, users *store.UserStore, dataRoot string, audit *store.AuditStore) *ItemService {
	return &ItemService{
		items:    items,
		cols:     cols,
		tasks:    tasks,
		perm:     NewPermissionChecker(cols, users, audit),
		dataRoot: filepath.Clean(dataRoot),
		audit:    audit,
	}
}

// CreateItemReq 添加数据项请求
type CreateItemReq struct {
	Path       string                 `json:"path"`
	Tags       map[string]interface{} `json:"tags"`
	AutoScrape *bool                  `json:"auto_scrape"` // 默认 true：刮削添加；false：直接添加
}

// BatchCreateReqItem 批量添加请求项
type BatchCreateReqItem struct {
	Path       string                 `json:"path"`
	Tags       map[string]interface{} `json:"tags"`
	AutoScrape *bool                  `json:"auto_scrape"` // 覆盖全局默认
}

// BatchCreateReq 批量添加请求（系统优化 1.1）：全局 auto_scrape 默认 false（批量导入通常是注册存量数据）
type BatchCreateReq struct {
	Items      []BatchCreateReqItem `json:"items" binding:"required"`
	AutoScrape bool                 `json:"auto_scrape"`
}

// Create 添加数据项（操作工/集合管理员）。
// 两种动作均校验路径存在性；autoScrape=true 时创建刮削任务并置 scrape_status=pending。
func (s *ItemService) Create(ctx context.Context, userID string, collectionID primitive.ObjectID, req CreateItemReq) (*model.DataItem, *model.ScrapeTask, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, nil, errno.ErrParam.WithCause(err)
	}
	c, e := s.perm.RequireRole(ctx, collectionID, uid, model.MemberRoleOperator)
	if e != nil {
		return nil, nil, e
	}
	if err := s.validatePath(req.Path); err != nil {
		return nil, nil, err
	}

	autoScrape := true
	if req.AutoScrape != nil {
		autoScrape = *req.AutoScrape
	}
	if autoScrape && c.ScrapeScript == nil {
		return nil, nil, errno.ErrNoScrapeScript
	}

	var tags map[string]interface{}
	if len(req.Tags) > 0 {
		tags, e = ValidateAndNormalizeTags(c.TagSchema, req.Tags, false)
		if e != nil {
			return nil, nil, e
		}
	}

	scrapeStatus := model.ItemScrapeNone
	if autoScrape {
		scrapeStatus = model.ItemScrapePending
	}
	item := &model.DataItem{
		CollectionID: collectionID,
		Path:         filepath.Clean(req.Path),
		Tags:         tags,
		ManualTags:   tags, // 直接添加的手动标签持久化，刮削时始终优先
		TagSource:    model.TagSourceManual,
		ScrapeStatus: scrapeStatus,
		CreatedBy:    uid,
	}
	if err := s.items.Create(ctx, item); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, nil, errno.ErrItemPathExists.WithCause(err)
		}
		return nil, nil, errno.ErrInternal.WithCause(err)
	}

	var task *model.ScrapeTask
	if autoScrape {
		task, e = s.enqueue(ctx, c, item, "auto")
		if e != nil {
			// 任务创建失败则回滚数据项，避免半状态
			_ = s.items.Delete(ctx, item.ID)
			return nil, nil, e
		}
	}
	s.audit.Log(ctx, uid, "item.create", "添加数据项 "+item.Path, &collectionID, &item.ID)
	return item, task, nil
}

// enqueue 创建刮削任务（脚本路径与数据路径快照）
func (s *ItemService) enqueue(ctx context.Context, c *model.BusinessCollection, item *model.DataItem, triggerBy string) (*model.ScrapeTask, *errno.Error) {
	if c.ScrapeScript == nil {
		return nil, errno.ErrNoScrapeScript
	}
	task := &model.ScrapeTask{
		CollectionID: c.ID,
		ItemID:       item.ID,
		ScriptPath:   c.ScrapeScript.Path,
		DataPath:     item.Path,
		Status:       model.TaskStatusPending,
		TriggerBy:    triggerBy,
		CreatedAt:    time.Now(),
	}
	if err := s.tasks.Create(ctx, task); err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	return task, nil
}

// BatchCreate 批量添加数据项（操作工/集合管理员；系统优化 1.1）。
// 单次 ≤500 条；权限一次判定，逐条校验（路径存在性/标签/脚本），
// 返回成功与失败明细（不因单条失败中断整批）。
func (s *ItemService) BatchCreate(ctx context.Context, userID string, collectionID primitive.ObjectID, req BatchCreateReq) (map[string]interface{}, *errno.Error) {
	if len(req.Items) == 0 {
		return nil, errno.ErrParam
	}
	if len(req.Items) > 500 {
		return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "单次最多批量添加 500 个数据项")
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	c, e := s.perm.RequireRole(ctx, collectionID, uid, model.MemberRoleOperator)
	if e != nil {
		return nil, e
	}
	success := make([]map[string]interface{}, 0)
	failed := make([]map[string]interface{}, 0)
	for i, it := range req.Items {
		path := strings.TrimSpace(it.Path)
		fail := func(msg string) {
			failed = append(failed, map[string]interface{}{"index": i, "path": it.Path, "error": msg})
		}
		if path == "" {
			fail("路径不能为空")
			continue
		}
		if err := s.validatePath(path); err != nil {
			fail(err.Message)
			continue
		}
		var tags map[string]interface{}
		if len(it.Tags) > 0 {
			tags, e = ValidateAndNormalizeTags(c.TagSchema, it.Tags, false)
			if e != nil {
				fail(e.Message)
				continue
			}
		}
		autoScrape := req.AutoScrape
		if it.AutoScrape != nil {
			autoScrape = *it.AutoScrape
		}
		if autoScrape && c.ScrapeScript == nil {
			fail(errno.ErrNoScrapeScript.Message)
			continue
		}
		scrapeStatus := model.ItemScrapeNone
		if autoScrape {
			scrapeStatus = model.ItemScrapePending
		}
		item := &model.DataItem{
			CollectionID: collectionID,
			Path:         filepath.Clean(path),
			Tags:         tags,
			ManualTags:   tags,
			TagSource:    model.TagSourceManual,
			ScrapeStatus: scrapeStatus,
			CreatedBy:    uid,
		}
		if err := s.items.Create(ctx, item); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				fail(errno.ErrItemPathExists.Message)
			} else {
				fail("服务内部错误")
			}
			continue
		}
		if autoScrape {
			if _, e := s.enqueue(ctx, c, item, "batch"); e != nil {
				_ = s.items.Delete(ctx, item.ID)
				fail(e.Message)
				continue
			}
		}
		success = append(success, map[string]interface{}{"index": i, "item_id": item.ID.Hex(), "path": item.Path})
	}
	if len(success) > 0 {
		s.audit.Log(ctx, uid, "item.batch_create",
			fmt.Sprintf("批量添加数据项 %d 条（成功 %d 失败 %d）", len(req.Items), len(success), len(failed)),
			&collectionID, nil)
	}
	return map[string]interface{}{"success": success, "failed": failed}, nil
}

// Get 数据项详情（所属集合操作工；admin 拥有全局只读浏览权）
func (s *ItemService) Get(ctx context.Context, userID string, itemID primitive.ObjectID, isAdmin bool) (*model.DataItem, *errno.Error) {
	item, err := s.items.FindByID(ctx, itemID)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if item == nil {
		return nil, errno.ErrItemNotFound
	}
	if isAdmin {
		return item, nil
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	if _, err := s.perm.RequireRole(ctx, item.CollectionID, uid, model.MemberRoleOperator); err != nil {
		return nil, err
	}
	return item, nil
}

// UpdateItemReq 修改数据项请求：标签值全量替换、路径可修改（均需校验）
type UpdateItemReq struct {
	Path *string                `json:"path"`
	Tags map[string]interface{} `json:"tags"`
}

// Update 修改数据项（所属集合操作工）
func (s *ItemService) Update(ctx context.Context, userID string, itemID primitive.ObjectID, req UpdateItemReq) (*model.DataItem, *errno.Error) {
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
	c, e := s.perm.RequireRole(ctx, item.CollectionID, uid, model.MemberRoleOperator)
	if e != nil {
		return nil, e
	}

	fields := bson.M{}
	if req.Path != nil {
		if err := s.validatePath(*req.Path); err != nil {
			return nil, err
		}
		fields["path"] = filepath.Clean(*req.Path)
	}
	if req.Tags != nil {
		newTags, e := ValidateAndNormalizeTags(c.TagSchema, req.Tags, false)
		if e != nil {
			return nil, e
		}
		// 手动标签始终优先：全量覆盖有效标签与手动标签，来源置回 manual
		fields["tags"] = newTags
		fields["manual_tags"] = newTags
		fields["tag_source"] = model.TagSourceManual
	}
	if len(fields) == 0 {
		return item, nil
	}
	if err := s.items.UpdateFields(ctx, itemID, fields); err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	s.audit.Log(ctx, uid, "item.update", "修改数据项 "+item.Path, &item.CollectionID, &itemID)
	updated, err := s.items.FindByID(ctx, itemID)
	if err != nil || updated == nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	return updated, nil
}

// Delete 删除数据项（仅元数据；级联删除其刮削任务）
func (s *ItemService) Delete(ctx context.Context, userID string, itemID primitive.ObjectID) *errno.Error {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errno.ErrParam.WithCause(err)
	}
	item, err := s.items.FindByID(ctx, itemID)
	if err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	if item == nil {
		return errno.ErrItemNotFound
	}
	if _, err := s.perm.RequireRole(ctx, item.CollectionID, uid, model.MemberRoleOperator); err != nil {
		return err
	}
	if err := s.tasks.DeleteByItem(ctx, itemID); err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	if err := s.items.Delete(ctx, itemID); err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	s.audit.Log(ctx, uid, "item.delete", "删除数据项 "+item.Path, &item.CollectionID, &itemID)
	return nil
}

// Search 在用户可访问集合内按路径包含匹配搜索数据项（添加关联/数据选择用）。
// 注意：此接口按路径搜索，不受标签类型限制（string 不支持模糊的约定仅针对标签查询）。
func (s *ItemService) Search(ctx context.Context, userID string, isAdmin bool, keyword string, page, pageSize int) ([]*model.DataItem, int64, *errno.Error) {
	filter := bson.M{}
	if !isAdmin {
		uid, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			return nil, 0, errno.ErrParam.WithCause(err)
		}
		cols, _, err := s.cols.List(ctx, bson.M{"members.user_id": uid}, 1, 100000)
		if err != nil {
			return nil, 0, errno.ErrInternal.WithCause(err)
		}
		ids := make([]primitive.ObjectID, 0, len(cols))
		for _, c := range cols {
			ids = append(ids, c.ID)
		}
		filter["collection_id"] = bson.M{"$in": ids}
	}
	if keyword != "" {
		filter["path"] = bson.M{"$regex": regexp.QuoteMeta(keyword), "$options": "i"}
	}
	items, total, err := s.items.List(ctx, filter, int64(page), int64(pageSize))
	if err != nil {
		return nil, 0, errno.ErrInternal.WithCause(err)
	}
	return items, total, nil
}

// List 按标签查询数据项（分页）。
// 支持操作符：等值(tag=value)、范围(tag.gt/gte/lt/lte)、存在(tag.exists=true)、in(tag.in=a,b)。
// string 不支持模糊/正则（Q6 已确认）。
func (s *ItemService) List(ctx context.Context, userID string, collectionID primitive.ObjectID, params url.Values, page, pageSize int) ([]*model.DataItem, int64, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, errno.ErrParam.WithCause(err)
	}
	c, e := s.perm.RequireRole(ctx, collectionID, uid, model.MemberRoleOperator)
	if e != nil {
		return nil, 0, e
	}

	filter, e := s.buildTagFilter(c.TagSchema, params)
	if e != nil {
		return nil, 0, e
	}
	filter["collection_id"] = collectionID

	items, total, err := s.items.List(ctx, filter, int64(page), int64(pageSize))
	if err != nil {
		return nil, 0, errno.ErrInternal.WithCause(err)
	}
	return items, total, nil
}

// buildTagFilter 将查询参数转换为标签过滤条件
func (s *ItemService) buildTagFilter(schema []model.TagDefinition, params url.Values) (bson.M, *errno.Error) {
	filter := bson.M{}
	schemaByName := make(map[string]model.TagDefinition, len(schema))
	for _, t := range schema {
		schemaByName[t.Name] = t
	}

	validOps := map[string]bool{"eq": true, "ne": true, "gt": true, "gte": true, "lt": true, "lte": true, "in": true, "exists": true}
	for key, values := range params {
		if key == "page" || key == "page_size" {
			continue
		}
		name, op := key, "eq"
		if i := strings.LastIndex(key, "."); i > 0 {
			name, op = key[:i], key[i+1:]
		}
		if !validOps[op] {
			return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "不支持的查询操作符: "+op)
		}
		t, ok := schemaByName[name]
		if !ok {
			return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "未知标签: "+name)
		}
		v := values[0]

		switch op {
		case "eq", "ne":
			fv, err := parseQueryValue(t, v)
			if err != nil {
				return nil, err
			}
			if op == "eq" {
				filter["tags."+name] = fv
			} else {
				filter["tags."+name] = bson.M{"$ne": fv}
			}
		case "gt", "gte", "lt", "lte":
			fv, err := parseQueryValue(t, v)
			if err != nil {
				return nil, err
			}
			filter["tags."+name] = bson.M{"$" + op: fv}
		case "in":
			parts := strings.Split(v, ",")
			parsed := make([]interface{}, 0, len(parts))
			for _, p := range parts {
				fv, err := parseQueryValue(t, strings.TrimSpace(p))
				if err != nil {
					return nil, err
				}
				parsed = append(parsed, fv)
			}
			filter["tags."+name] = bson.M{"$in": parsed}
		case "exists":
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "exists 参数应为 true/false")
			}
			filter["tags."+name] = bson.M{"$exists": b}
		}
	}
	return filter, nil
}

// parseQueryValue 按标签类型解析查询参数值
func parseQueryValue(t model.TagDefinition, s string) (interface{}, *errno.Error) {
	switch t.Type {
	case model.TagTypeString, model.TagTypeEnum:
		return s, nil
	case model.TagTypeInt:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "标签 "+t.Name+" 应为整数")
		}
		return n, nil
	case model.TagTypeFloat:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "标签 "+t.Name+" 应为数值")
		}
		return f, nil
	case model.TagTypeBool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "标签 "+t.Name+" 应为 true/false")
		}
		return b, nil
	case model.TagTypeDate:
		for _, layout := range []string{time.RFC3339, "2006-01-02"} {
			if tm, err := time.Parse(layout, s); err == nil {
				return tm, nil
			}
		}
		return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "标签 "+t.Name+" 应为日期/时间")
	case model.TagTypeArray, model.TagTypeObject:
		return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "标签 "+t.Name+" 不支持等值/范围查询")
	default:
		return nil, errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "不支持的标签类型: "+string(t.Type))
	}
}

// validatePath 校验数据路径：绝对路径、位于数据根目录内、且实际存在（文件/文件夹均可）
func (s *ItemService) validatePath(path string) *errno.Error {
	if strings.TrimSpace(path) == "" {
		return errno.ErrParam
	}
	if s.dataRoot == "" || s.dataRoot == "." {
		return errno.ErrInternal.WithCause(errors.New("未配置数据根目录 data.root_dir"))
	}
	if !filepath.IsAbs(path) {
		return errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "数据路径必须是绝对路径")
	}
	clean := filepath.Clean(path)
	if clean != s.dataRoot && !strings.HasPrefix(clean, s.dataRoot+string(filepath.Separator)) {
		return errno.ErrPathOutsideRoot
	}
	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return errno.ErrPathNotExist
		}
		return errno.ErrInternal.WithCause(err)
	}
	_ = info // 文件与文件夹均可（Q4：后端只判断存在性）
	return nil
}
