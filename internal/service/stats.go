package service

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"datacenter/internal/errno"
	"datacenter/internal/model"
	"datacenter/internal/store"
)

// StatsService 仪表盘统计服务
type StatsService struct {
	cols      *store.CollectionStore
	items     *store.ItemStore
	tasks     *store.TaskStore
	relations *store.RelationStore
}

// NewStatsService 构造统计服务
func NewStatsService(cols *store.CollectionStore, items *store.ItemStore, tasks *store.TaskStore, relations *store.RelationStore) *StatsService {
	return &StatsService{cols: cols, items: items, tasks: tasks, relations: relations}
}

// TaskCounts 任务状态计数
type TaskCounts struct {
	Pending int64 `json:"pending"`
	Running int64 `json:"running"`
	Success int64 `json:"success"`
	Failed  int64 `json:"failed"`
	Total   int64 `json:"total"`
}

// RelationCounts 关联关系计数（按类型）
type RelationCounts struct {
	ParentChild int64 `json:"parent_child"`
	Reference   int64 `json:"reference"`
	Call        int64 `json:"call"`
	Total       int64 `json:"total"`
}

// Overview 概览统计：admin 统计全局，普通用户仅统计自己参与的集合
type Overview struct {
	Collections int64          `json:"collections"`
	Items       int64          `json:"items"`
	Tasks       TaskCounts     `json:"tasks"`
	Relations   RelationCounts `json:"relations"`
}

// Overview 获取概览统计
func (s *StatsService) Overview(ctx context.Context, userID string, isAdmin bool) (*Overview, *errno.Error) {
	colFilter := bson.M{}
	if !isAdmin {
		uid, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			return nil, errno.ErrParam.WithCause(err)
		}
		colFilter["members.user_id"] = uid
	}
	colCount, err := s.cols.Count(ctx, colFilter)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}

	var itemFilter, taskFilter, relFilter bson.M
	if isAdmin {
		itemFilter = bson.M{}
		taskFilter = bson.M{}
		relFilter = bson.M{}
	} else {
		cols, _, err := s.cols.List(ctx, colFilter, 1, 100000)
		if err != nil {
			return nil, errno.ErrInternal.WithCause(err)
		}
		ids := make([]primitive.ObjectID, 0, len(cols))
		for _, c := range cols {
			ids = append(ids, c.ID)
		}
		itemFilter = bson.M{"collection_id": bson.M{"$in": ids}}
		taskFilter = bson.M{"collection_id": bson.M{"$in": ids}}
		relFilter = bson.M{"collection_id": bson.M{"$in": ids}}
	}

	itemCount, err := s.items.Count(ctx, itemFilter)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}

	tc := TaskCounts{}
	for _, st := range []struct {
		key string
		dst *int64
	}{
		{model.TaskStatusPending, &tc.Pending},
		{model.TaskStatusRunning, &tc.Running},
		{model.TaskStatusSuccess, &tc.Success},
		{model.TaskStatusFailed, &tc.Failed},
	} {
		f := bson.M{}
		for k, v := range taskFilter {
			f[k] = v
		}
		f["status"] = st.key
		n, err := s.tasks.Count(ctx, f)
		if err != nil {
			return nil, errno.ErrInternal.WithCause(err)
		}
		*st.dst = n
	}
	total, err := s.tasks.Count(ctx, taskFilter)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	tc.Total = total

	// 关联关系计数（按类型）
	rc := RelationCounts{}
	for _, st := range []struct {
		key string
		dst *int64
	}{
		{model.RelationParentChild, &rc.ParentChild},
		{model.RelationReference, &rc.Reference},
		{model.RelationCall, &rc.Call},
	} {
		f := bson.M{}
		for k, v := range relFilter {
			f[k] = v
		}
		f["type"] = st.key
		n, err := s.relations.Count(ctx, f)
		if err != nil {
			return nil, errno.ErrInternal.WithCause(err)
		}
		*st.dst = n
	}
	relTotal, err := s.relations.Count(ctx, relFilter)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	rc.Total = relTotal

	return &Overview{Collections: colCount, Items: itemCount, Tasks: tc, Relations: rc}, nil
}
