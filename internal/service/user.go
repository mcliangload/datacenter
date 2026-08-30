package service

import (
	"context"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"datacenter/internal/errno"
	"datacenter/internal/model"
	"datacenter/internal/store"
)

// UserService 用户管理服务（公共权限：仅 admin 可调用，由中间件 RequireRole 保障）
type UserService struct {
	users *store.UserStore
	audit *store.AuditStore
}

// NewUserService 构造用户管理服务
func NewUserService(users *store.UserStore, audit *store.AuditStore) *UserService {
	return &UserService{users: users, audit: audit}
}

// CreateUserReq 创建用户请求
type CreateUserReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// Create 创建用户（admin 专属）
func (s *UserService) Create(ctx context.Context, actorID string, req CreateUserReq) (*model.User, *errno.Error) {
	username := strings.TrimSpace(req.Username)
	if username == "" || req.Password == "" {
		return nil, errno.ErrParam
	}
	// 安全增强 P0-3：密码强度校验（≥8 位 + 字母 + 数字）
	if e := validatePasswordStrength(req.Password); e != nil {
		return nil, e
	}
	role := req.Role
	if role == "" {
		role = model.RoleUser
	}
	if role != model.RoleAdmin && role != model.RoleUser {
		return nil, errno.ErrParam
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}

	u := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
		Status:       model.UserStatusActive,
	}
	if err := s.users.Create(ctx, u); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, errno.ErrUsernameExists.WithCause(err)
		}
		return nil, errno.ErrInternal.WithCause(err)
	}
	actorObjectID, _ := primitive.ObjectIDFromHex(actorID)
	s.audit.Log(ctx, actorObjectID, "user.create", "创建用户 "+u.Username+"（角色 "+role+"）", nil, nil)
	return u, nil
}

// List 分页查询用户，keyword 按用户名包含匹配
func (s *UserService) List(ctx context.Context, page, pageSize int, keyword string) ([]*model.User, int64, *errno.Error) {
	filter := bson.M{}
	if keyword != "" {
		filter["username"] = bson.M{"$regex": regexp.QuoteMeta(keyword)}
	}
	users, total, err := s.users.List(ctx, filter, int64(page), int64(pageSize))
	if err != nil {
		return nil, 0, errno.ErrInternal.WithCause(err)
	}
	return users, total, nil
}

// UpdateUserReq 更新用户请求（均为可选字段）
type UpdateUserReq struct {
	Password *string `json:"password"`
	Role     *string `json:"role"`
	Status   *string `json:"status"`
}

// Update 更新用户（admin 专属；不能修改自己，不能禁用/删除最后一个管理员）
func (s *UserService) Update(ctx context.Context, actorID string, id primitive.ObjectID, req UpdateUserReq) (*model.User, *errno.Error) {
	if actorID == id.Hex() {
		return nil, errno.ErrUserSelfOp
	}
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if u == nil {
		return nil, errno.ErrUserNotFound
	}

	fields := bson.M{}
	if req.Password != nil && *req.Password != "" {
		// 安全增强 P0-3：密码强度校验
		if e := validatePasswordStrength(*req.Password); e != nil {
			return nil, e
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, errno.ErrInternal.WithCause(err)
		}
		fields["password_hash"] = string(hash)
		// 安全增强 P1-7：改密后吊销该用户已签发的全部旧 token
		fields["password_version"] = u.PasswordVersion + 1
	}
	if req.Role != nil {
		if *req.Role != model.RoleAdmin && *req.Role != model.RoleUser {
			return nil, errno.ErrParam
		}
		fields["role"] = *req.Role
	}
	if req.Status != nil {
		if *req.Status != model.UserStatusActive && *req.Status != model.UserStatusDisabled {
			return nil, errno.ErrParam
		}
		if *req.Status == model.UserStatusDisabled && u.Role == model.RoleAdmin {
			if err := s.ensureNotLastAdmin(ctx); err != nil {
				return nil, err
			}
		}
		fields["status"] = *req.Status
	}
	if len(fields) == 0 {
		return u, nil
	}
	if err := s.users.UpdateFields(ctx, id, fields); err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	updated, err := s.users.FindByID(ctx, id)
	if err != nil || updated == nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	actorObjectID, _ := primitive.ObjectIDFromHex(actorID)
	s.audit.Log(ctx, actorObjectID, "user.update", "更新用户 "+u.Username, nil, nil)
	return updated, nil
}

// Delete 删除用户（admin 专属；不能删除自己，不能删除最后一个管理员）
func (s *UserService) Delete(ctx context.Context, actorID string, id primitive.ObjectID) *errno.Error {
	if actorID == id.Hex() {
		return errno.ErrUserSelfOp
	}
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	if u == nil {
		return errno.ErrUserNotFound
	}
	if u.Role == model.RoleAdmin {
		if err := s.ensureNotLastAdmin(ctx); err != nil {
			return err
		}
	}
	if err := s.users.Delete(ctx, id); err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	actorObjectID, _ := primitive.ObjectIDFromHex(actorID)
	s.audit.Log(ctx, actorObjectID, "user.delete", "删除用户 "+u.Username, nil, nil)
	return nil
}

func (s *UserService) ensureNotLastAdmin(ctx context.Context) *errno.Error {
	n, err := s.users.CountByRole(ctx, model.RoleAdmin)
	if err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	if n <= 1 {
		return errno.ErrLastAdmin
	}
	return nil
}
