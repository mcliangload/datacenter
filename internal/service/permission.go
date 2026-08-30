package service

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"datacenter/internal/errno"
	"datacenter/internal/model"
	"datacenter/internal/store"
)

// PermissionChecker 集合级权限判定。
// 核心原则（见需求分解 §4）：集合权限必须以「权限点 × collection_id」为单位，
// 逐集合查询该集合自己的 members；**全局 admin 拥有最高权限**——对任意集合的
// 查看/修改全部放行（v0.12），其余用户绝不做全局集合权限判定。
type PermissionChecker struct {
	cols  *store.CollectionStore
	users *store.UserStore
	audit *store.AuditStore // 安全增强 P1-6：权限拒绝审计
}

// NewPermissionChecker 构造集合权限判定器
func NewPermissionChecker(cols *store.CollectionStore, users *store.UserStore, audit *store.AuditStore) *PermissionChecker {
	return &PermissionChecker{cols: cols, users: users, audit: audit}
}

// isGlobalAdmin 判断用户是否为全局 admin
func (p *PermissionChecker) isGlobalAdmin(ctx context.Context, userID primitive.ObjectID) (bool, error) {
	u, err := p.users.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}
	return u != nil && u.Role == model.RoleAdmin, nil
}

// MemberRole 返回用户在该集合中的角色；集合不存在返回 ErrCollectionNotFound。
// 全局 admin 恒返回集合管理员角色（最高集合权限，等价于可查看/修改该集合全部资源）。
func (p *PermissionChecker) MemberRole(ctx context.Context, collectionID, userID primitive.ObjectID) (model.MemberRole, *errno.Error) {
	c, err := p.cols.FindByID(ctx, collectionID)
	if err != nil {
		return "", errno.ErrInternal.WithCause(err)
	}
	if c == nil {
		return "", errno.ErrCollectionNotFound
	}
	isAdmin, err := p.isGlobalAdmin(ctx, userID)
	if err != nil {
		return "", errno.ErrInternal.WithCause(err)
	}
	if isAdmin {
		return model.MemberRoleCollectionAdmin, nil
	}
	role, ok := c.RoleOf(userID)
	if !ok {
		return "", errno.ErrNotMember
	}
	return role, nil
}

// RequireRole 校验用户在该集合中满足所需角色（集合管理员包含操作工权限），返回集合
func (p *PermissionChecker) RequireRole(ctx context.Context, collectionID, userID primitive.ObjectID, required model.MemberRole) (*model.BusinessCollection, *errno.Error) {
	role, err := p.MemberRole(ctx, collectionID, userID)
	if err != nil {
		return nil, err
	}
	if !roleSatisfies(role, required) {
		// 安全增强 P1-6：权限拒绝审计
		roleDesc := string(role)
		if role == "" {
			roleDesc = "非成员"
		}
		p.audit.Log(ctx, userID, "auth.permission_denied",
			fmt.Sprintf("权限拒绝: 需要角色 %s 实际 %s（集合 %s）", required, roleDesc, collectionID.Hex()),
			&collectionID, nil)
		return nil, errno.ErrNoPermission
	}
	return p.mustCollection(ctx, collectionID)
}

// roleSatisfies 判断实际角色是否满足所需角色（集合管理员拥有操作工的全部权限）
func roleSatisfies(actual, required model.MemberRole) bool {
	if required == model.MemberRoleOperator && actual == model.MemberRoleCollectionAdmin {
		return true
	}
	return actual == required
}

func (p *PermissionChecker) mustCollection(ctx context.Context, collectionID primitive.ObjectID) (*model.BusinessCollection, *errno.Error) {
	c, err := p.cols.FindByID(ctx, collectionID)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if c == nil {
		return nil, errno.ErrCollectionNotFound
	}
	return c, nil
}
