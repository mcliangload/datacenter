package service

import (
	"context"
	"regexp"

	"go.mongodb.org/mongo-driver/bson"

	"datacenter/internal/errno"
	"datacenter/internal/store"
)

// AuditLogView 审计日志视图（含操作者用户名）
type AuditLogView struct {
	ID           string `json:"id"`
	ActorID      string `json:"actor_id"`
	ActorName    string `json:"actor_name"`
	Action       string `json:"action"`
	CollectionID string `json:"collection_id,omitempty"`
	ItemID       string `json:"item_id,omitempty"`
	Detail       string `json:"detail"`
	CreatedAt    string `json:"created_at"`
}

// AuditService 审计日志查询（admin 专属；系统优化 3.1）
type AuditService struct {
	audit *store.AuditStore
	users *store.UserStore
}

// NewAuditService 构造审计日志服务
func NewAuditService(audit *store.AuditStore, users *store.UserStore) *AuditService {
	return &AuditService{audit: audit, users: users}
}

// List 分页查询审计日志（按时间倒序）：action 支持精确/前缀（如 auth. 看全部安全事件），
// username 按操作者用户名过滤。
func (s *AuditService) List(ctx context.Context, action, username string, page, pageSize int) ([]*AuditLogView, int64, *errno.Error) {
	filter := bson.M{}
	if action != "" {
		filter["action"] = bson.M{"$regex": "^" + regexp.QuoteMeta(action)}
	}
	if username != "" {
		u, err := s.users.FindByUsername(ctx, username)
		if err != nil {
			return nil, 0, errno.ErrInternal.WithCause(err)
		}
		if u == nil {
			return []*AuditLogView{}, 0, nil
		}
		filter["actor_id"] = u.ID
	}
	logs, total, err := s.audit.List(ctx, filter, int64(page), int64(pageSize))
	if err != nil {
		return nil, 0, errno.ErrInternal.WithCause(err)
	}
	// 组装操作者用户名
	nameMap := map[string]string{}
	for _, l := range logs {
		if _, ok := nameMap[l.ActorID.Hex()]; ok {
			continue
		}
		u, err := s.users.FindByID(ctx, l.ActorID)
		if err == nil && u != nil {
			nameMap[l.ActorID.Hex()] = u.Username
		}
	}
	views := make([]*AuditLogView, 0, len(logs))
	for _, l := range logs {
		v := &AuditLogView{
			ID:        l.ID.Hex(),
			ActorID:   l.ActorID.Hex(),
			ActorName: nameMap[l.ActorID.Hex()],
			Action:    l.Action,
			Detail:    l.Detail,
			CreatedAt: l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if l.CollectionID != nil {
			v.CollectionID = l.CollectionID.Hex()
		}
		if l.ItemID != nil {
			v.ItemID = l.ItemID.Hex()
		}
		views = append(views, v)
	}
	return views, total, nil
}
