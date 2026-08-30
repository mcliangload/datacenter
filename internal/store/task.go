package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"datacenter/internal/model"
)

// TaskStore 刮削任务数据访问
type TaskStore struct {
	coll *mongo.Collection
}

// NewTaskStore 构造刮削任务存储
func NewTaskStore(db *mongo.Database) *TaskStore {
	return &TaskStore{coll: db.Collection(model.CollectionScrapeTasks)}
}

// Create 创建任务
func (s *TaskStore) Create(ctx context.Context, t *model.ScrapeTask) error {
	t.ID = primitive.NewObjectID()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	_, err := s.coll.InsertOne(ctx, t)
	return err
}

// FindByID 按 ID 查询，不存在时返回 (nil, nil)
func (s *TaskStore) FindByID(ctx context.Context, id primitive.ObjectID) (*model.ScrapeTask, error) {
	var t model.ScrapeTask
	err := s.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ClaimNext 领取下一个可执行任务（原子操作）：
//   - pending 任务直接领取；
//   - running 但 started_at 早于 reclaimSeconds 前的任务视为僵死，回收重新执行。
//
// 无任务时返回 (nil, nil)。
func (s *TaskStore) ClaimNext(ctx context.Context, reclaimSeconds int) (*model.ScrapeTask, error) {
	now := time.Now()
	stale := now.Add(-time.Duration(reclaimSeconds) * time.Second)
	filter := bson.M{"$or": []bson.M{
		{"status": model.TaskStatusPending},
		{"status": model.TaskStatusRunning, "started_at": bson.M{"$lt": stale}},
	}}
	update := bson.M{"$set": bson.M{"status": model.TaskStatusRunning, "started_at": now}}

	var t model.ScrapeTask
	err := s.coll.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Complete 完成任务（success/failed），记录退出码、错误信息与结果标签
func (s *TaskStore) Complete(ctx context.Context, id primitive.ObjectID, status string, exitCode *int, errMsg string, resultTags map[string]interface{}) error {
	fields := bson.M{"status": status, "finished_at": time.Now()}
	if exitCode != nil {
		fields["exit_code"] = *exitCode
	}
	if errMsg != "" {
		fields["error"] = errMsg
	}
	if resultTags != nil {
		fields["result_tags"] = resultTags
	}
	res, err := s.coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": fields})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// ListByItem 分页查询某数据项的刮削历史，按创建时间倒序
func (s *TaskStore) ListByItem(ctx context.Context, itemID primitive.ObjectID, page, pageSize int64) ([]*model.ScrapeTask, int64, error) {
	filter := bson.M{"item_id": itemID}
	total, err := s.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip((page - 1) * pageSize).
		SetLimit(pageSize).
		SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := s.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	tasks := make([]*model.ScrapeTask, 0, pageSize)
	for cursor.Next(ctx) {
		var t model.ScrapeTask
		if err := cursor.Decode(&t); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, &t)
	}
	return tasks, total, cursor.Err()
}

// LatestByItem 查询某数据项最近一次任务，不存在时返回 (nil, nil)
func (s *TaskStore) LatestByItem(ctx context.Context, itemID primitive.ObjectID) (*model.ScrapeTask, error) {
	var t model.ScrapeTask
	err := s.coll.FindOne(ctx, bson.M{"item_id": itemID},
		options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// DeleteByItem 级联删除某数据项的全部任务
func (s *TaskStore) DeleteByItem(ctx context.Context, itemID primitive.ObjectID) error {
	_, err := s.coll.DeleteMany(ctx, bson.M{"item_id": itemID})
	return err
}

// DeleteByItems 批量删除多个数据项的任务（级联删除用）
func (s *TaskStore) DeleteByItems(ctx context.Context, itemIDs []primitive.ObjectID) error {
	if len(itemIDs) == 0 {
		return nil
	}
	_, err := s.coll.DeleteMany(ctx, bson.M{"item_id": bson.M{"$in": itemIDs}})
	return err
}

// DeleteByCollection 级联删除某集合的全部任务
func (s *TaskStore) DeleteByCollection(ctx context.Context, collectionID primitive.ObjectID) error {
	_, err := s.coll.DeleteMany(ctx, bson.M{"collection_id": collectionID})
	return err
}

// List 分页查询任务（按创建时间倒序），支持任意过滤条件（如状态、集合范围）
func (s *TaskStore) List(ctx context.Context, filter bson.M, page, pageSize int64) ([]*model.ScrapeTask, int64, error) {
	total, err := s.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSkip((page - 1) * pageSize).
		SetLimit(pageSize).
		SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := s.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	tasks := make([]*model.ScrapeTask, 0, pageSize)
	for cursor.Next(ctx) {
		var t model.ScrapeTask
		if err := cursor.Decode(&t); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, &t)
	}
	return tasks, total, cursor.Err()
}

// Count 统计任务数（支持任意过滤条件）
func (s *TaskStore) Count(ctx context.Context, filter bson.M) (int64, error) {
	return s.coll.CountDocuments(ctx, filter)
}
