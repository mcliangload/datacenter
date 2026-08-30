package service

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"datacenter/internal/errno"
	"datacenter/internal/model"
	"datacenter/internal/store"
)

// ScrapeService 刮削任务服务（任务的创建/查询；执行由独立刮削子系统 cmd/scraper 完成）
type ScrapeService struct {
	tasks *store.TaskStore
	items *store.ItemStore
	cols  *store.CollectionStore
	perm  *PermissionChecker
	audit *store.AuditStore
}

// NewScrapeService 构造刮削任务服务
func NewScrapeService(tasks *store.TaskStore, items *store.ItemStore,
	cols *store.CollectionStore, users *store.UserStore, audit *store.AuditStore) *ScrapeService {
	return &ScrapeService{
		tasks: tasks,
		items: items,
		cols:  cols,
		perm:  NewPermissionChecker(cols, users, audit),
		audit: audit,
	}
}

// Trigger 手动触发刮削（操作工/集合管理员）：基于最近一次任务快照创建重试任务
func (s *ScrapeService) Trigger(ctx context.Context, userID string, itemID primitive.ObjectID) (*model.ScrapeTask, *errno.Error) {
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
	if c.ScrapeScript == nil {
		return nil, errno.ErrNoScrapeScript
	}

	last, err := s.tasks.LatestByItem(ctx, itemID)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	task := &model.ScrapeTask{
		CollectionID: item.CollectionID,
		ItemID:       itemID,
		ScriptPath:   c.ScrapeScript.Path,
		DataPath:     item.Path,
		Status:       model.TaskStatusPending,
		TriggerBy:    userID,
		CreatedAt:    time.Now(),
	}
	if last != nil {
		task.RetryOf = &last.ID
	}
	if err := s.tasks.Create(ctx, task); err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if err := s.items.UpdateFields(ctx, itemID, bson.M{"scrape_status": model.ItemScrapePending}); err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	s.audit.Log(ctx, uid, "item.scrape", "手动触发刮削 "+item.Path, &item.CollectionID, &itemID)
	return task, nil
}

// GetTask 任务详情（任务所属集合的操作工/集合管理员）
func (s *ScrapeService) GetTask(ctx context.Context, userID string, taskID primitive.ObjectID) (*model.ScrapeTask, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	task, err := s.tasks.FindByID(ctx, taskID)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if task == nil {
		return nil, errno.ErrTaskNotFound
	}
	if _, err := s.perm.RequireRole(ctx, task.CollectionID, uid, model.MemberRoleOperator); err != nil {
		return nil, err
	}
	return task, nil
}

// ListByItem 某数据项的刮削历史
func (s *ScrapeService) ListByItem(ctx context.Context, userID string, itemID primitive.ObjectID, page, pageSize int) ([]*model.ScrapeTask, int64, *errno.Error) {
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
	if _, err := s.perm.RequireRole(ctx, item.CollectionID, uid, model.MemberRoleOperator); err != nil {
		return nil, 0, err
	}
	tasks, total, err := s.tasks.ListByItem(ctx, itemID, int64(page), int64(pageSize))
	if err != nil {
		return nil, 0, errno.ErrInternal.WithCause(err)
	}
	return tasks, total, nil
}

// ListTasks 全局刮削任务列表（刮削管理页）：
// admin 可见全部任务，普通用户仅可见自己参与集合的任务；可按状态过滤。
func (s *ScrapeService) ListTasks(ctx context.Context, userID string, isAdmin bool, status string, page, pageSize int) ([]*model.ScrapeTask, int64, *errno.Error) {
	filter := bson.M{}
	if status != "" {
		switch status {
		case model.TaskStatusPending, model.TaskStatusRunning, model.TaskStatusSuccess, model.TaskStatusFailed:
			filter["status"] = status
		default:
			return nil, 0, errno.ErrParam
		}
	}
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
	tasks, total, err := s.tasks.List(ctx, filter, int64(page), int64(pageSize))
	if err != nil {
		return nil, 0, errno.ErrInternal.WithCause(err)
	}
	return tasks, total, nil
}
