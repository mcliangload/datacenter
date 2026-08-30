package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"datacenter/internal/errno"
	"datacenter/internal/model"
	"datacenter/internal/store"
)

// CollectionService 集合管理服务。
// 公共权限（创建/删除/更换管理员）由 handler 的 RequireRole(admin) 保障；
// 集合级权限一律通过 PermissionChecker 逐集合判定。
type CollectionService struct {
	cols  *store.CollectionStore
	users *store.UserStore
	items *store.ItemStore
	tasks *store.TaskStore
	perm  *PermissionChecker
	audit *store.AuditStore
}

// NewCollectionService 构造集合管理服务
func NewCollectionService(cols *store.CollectionStore, users *store.UserStore,
	items *store.ItemStore, tasks *store.TaskStore, audit *store.AuditStore) *CollectionService {
	return &CollectionService{
		cols:  cols,
		users: users,
		items: items,
		tasks: tasks,
		perm:  NewPermissionChecker(cols, users, audit),
		audit: audit,
	}
}

// CreateCollectionReq 创建集合请求
type CreateCollectionReq struct {
	Name           string                `json:"name"`
	Description    string                `json:"description"`
	TagSchema      []model.TagDefinition `json:"tag_schema"`
	InitialAdminID string                `json:"initial_admin_id"`
}

// Create 创建集合（admin 专属）：名称全局唯一，初始集合管理员必须存在
func (s *CollectionService) Create(ctx context.Context, actorID primitive.ObjectID, req CreateCollectionReq) (*model.BusinessCollection, *errno.Error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errno.ErrParam
	}
	if err := ValidateTagSchema(req.TagSchema); err != nil {
		return nil, err
	}
	adminID, err := primitive.ObjectIDFromHex(req.InitialAdminID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	admin, err := s.users.FindByID(ctx, adminID)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if admin == nil {
		return nil, errno.ErrUserNotFound
	}

	c := &model.BusinessCollection{
		Name:        name,
		Description: req.Description,
		CreatedBy:   actorID,
		TagSchema:   req.TagSchema,
		Members: []model.Member{
			{UserID: adminID, Role: model.MemberRoleCollectionAdmin},
		},
	}
	// 规范化：空标签定义为空数组而非 null，保证 API 输出一致（前端依赖）
	if c.TagSchema == nil {
		c.TagSchema = []model.TagDefinition{}
	}
	if err := s.cols.Create(ctx, c); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, errno.ErrCollectionNameExists.WithCause(err)
		}
		return nil, errno.ErrInternal.WithCause(err)
	}
	s.audit.Log(ctx, actorID, "collection.create", "创建集合 "+c.Name, &c.ID, nil)
	return c, nil
}

// List 列表：admin 看全部，其他用户只看自己参与的集合
func (s *CollectionService) List(ctx context.Context, userID string, isAdmin bool, page, pageSize int) ([]*model.BusinessCollection, int64, *errno.Error) {
	filter := bson.M{}
	if !isAdmin {
		uid, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			return nil, 0, errno.ErrParam.WithCause(err)
		}
		filter["members.user_id"] = uid
	}
	cols, total, err := s.cols.List(ctx, filter, int64(page), int64(pageSize))
	if err != nil {
		return nil, 0, errno.ErrInternal.WithCause(err)
	}
	return cols, total, nil
}

// UpdateDeletePolicy 设置集合删除策略（集合管理员；传空策略 = 恢复默认）
func (s *CollectionService) UpdateDeletePolicy(ctx context.Context, userID string, id primitive.ObjectID, policy model.DeletePolicy) (*model.BusinessCollection, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	if _, err := s.perm.RequireRole(ctx, id, uid, model.MemberRoleCollectionAdmin); err != nil {
		return nil, err
	}
	if !model.ValidDeletePolicy(policy) {
		return nil, errno.ErrParam
	}
	if err := s.cols.UpdateDeletePolicy(ctx, id, &policy); err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	s.audit.Log(ctx, uid, "collection.delete_policy",
		"设置删除策略 children="+policy.Children+" incoming="+policy.Incoming, &id, nil)
	return s.findUpdated(ctx, id)
}

// Get 详情：集合成员可查看；admin 拥有全局只读浏览权（列表/DQL 已全量，详情同样放行，
// 与"admin 创建集合后分配集合管理员"的职责模型一致），写操作仍按集合角色判定
func (s *CollectionService) Get(ctx context.Context, userID string, id primitive.ObjectID, isAdmin bool) (*model.BusinessCollection, *errno.Error) {
	c, err := s.cols.FindByID(ctx, id)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if c == nil {
		return nil, errno.ErrCollectionNotFound
	}
	if isAdmin {
		return c, nil
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	if !c.IsMember(uid) {
		return nil, errno.ErrNotMember
	}
	return c, nil
}

// UpdateMeta 修改集合基础信息（集合管理员，仅限自己管理的集合）
func (s *CollectionService) UpdateMeta(ctx context.Context, userID string, id primitive.ObjectID, description string) (*model.BusinessCollection, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	if _, err := s.perm.RequireRole(ctx, id, uid, model.MemberRoleCollectionAdmin); err != nil {
		return nil, err
	}
	if err := s.cols.UpdateMeta(ctx, id, description); err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	s.audit.Log(ctx, uid, "collection.update", "修改集合基础信息", &id, nil)
	return s.findUpdated(ctx, id)
}

// findUpdated 读取集合并包装错误（仅供更新类方法返回用）
func (s *CollectionService) findUpdated(ctx context.Context, id primitive.ObjectID) (*model.BusinessCollection, *errno.Error) {
	c, err := s.cols.FindByID(ctx, id)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	return c, nil
}

// UpdateTagSchema 更新标签定义：replace=true 全量替换，false 增量合并
func (s *CollectionService) UpdateTagSchema(ctx context.Context, userID string, id primitive.ObjectID, schema []model.TagDefinition, replace bool) (*model.BusinessCollection, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	if _, err := s.perm.RequireRole(ctx, id, uid, model.MemberRoleCollectionAdmin); err != nil {
		return nil, err
	}
	if err := ValidateTagSchema(schema); err != nil {
		return nil, err
	}
	if !replace {
		c, err := s.cols.FindByID(ctx, id)
		if err != nil {
			return nil, errno.ErrInternal.WithCause(err)
		}
		if c == nil {
			return nil, errno.ErrCollectionNotFound
		}
		schema = mergeTagSchemas(c.TagSchema, schema)
		if err := ValidateTagSchema(schema); err != nil {
			return nil, err
		}
	}
	if err := s.cols.UpdateTagSchema(ctx, id, schema); err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	s.audit.Log(ctx, uid, "collection.tags", "更新标签定义", &id, nil)
	return s.findUpdated(ctx, id)
}

// mergeTagSchemas 增量合并：已有标签按名替换定义，新标签追加（删除仅可通过全量替换）
func mergeTagSchemas(existing, incoming []model.TagDefinition) []model.TagDefinition {
	byName := make(map[string]model.TagDefinition, len(existing))
	for _, t := range existing {
		byName[t.Name] = t
	}
	for _, t := range incoming {
		byName[t.Name] = t
	}
	merged := make([]model.TagDefinition, 0, len(byName))
	for _, t := range byName {
		merged = append(merged, t)
	}
	return merged
}

// UpdateScrapeScript 设置/替换集合刮削脚本（集合管理员）：路径必须为存在的绝对路径文件
func (s *CollectionService) UpdateScrapeScript(ctx context.Context, userID string, id primitive.ObjectID, path string) (*model.BusinessCollection, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	if _, err := s.perm.RequireRole(ctx, id, uid, model.MemberRoleCollectionAdmin); err != nil {
		return nil, err
	}
	if err := validateScriptPath(path); err != nil {
		return nil, err
	}
	script := &model.ScrapeScript{
		Path:      filepath.Clean(path),
		UpdatedBy: uid,
		UpdatedAt: time.Now(),
	}
	if err := s.cols.UpdateScrapeScript(ctx, id, script); err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	s.audit.Log(ctx, uid, "collection.script", "设置刮削脚本 "+script.Path, &id, nil)
	return s.findUpdated(ctx, id)
}
func validateScriptPath(path string) *errno.Error {
	if strings.TrimSpace(path) == "" {
		return errno.ErrParam
	}
	if !filepath.IsAbs(path) {
		return errno.New(errno.ErrTagSchemaInvalid.Code, errno.ErrTagSchemaInvalid.HTTPStatus, "刮削脚本路径必须是绝对路径")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errno.New(errno.ErrPathNotExist.Code, errno.ErrPathNotExist.HTTPStatus, "刮削脚本路径不存在")
		}
		return errno.ErrInternal.WithCause(err)
	}
	if info.IsDir() {
		return errno.New(errno.ErrTagSchemaInvalid.Code, errno.ErrTagSchemaInvalid.HTTPStatus, "刮削脚本路径必须是文件")
	}
	return nil
}

// GrantMember 授权操作工（集合管理员，仅限自己管理的集合）
func (s *CollectionService) GrantMember(ctx context.Context, userID string, id, targetUserID primitive.ObjectID) (*model.BusinessCollection, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	if _, err := s.perm.RequireRole(ctx, id, uid, model.MemberRoleCollectionAdmin); err != nil {
		return nil, err
	}
	target, err := s.users.FindByID(ctx, targetUserID)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if target == nil {
		return nil, errno.ErrUserNotFound
	}
	c, err := s.cols.FindByID(ctx, id)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if c == nil {
		return nil, errno.ErrCollectionNotFound
	}
	if c.IsMember(targetUserID) {
		return nil, errno.ErrMemberExists
	}
	if err := s.cols.AddMember(ctx, id, model.Member{UserID: targetUserID, Role: model.MemberRoleOperator}); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errno.ErrMemberExists
		}
		return nil, errno.ErrInternal.WithCause(err)
	}
	s.audit.Log(ctx, uid, "collection.grant", "授权操作工 "+target.Username, &id, nil)
	return s.findUpdated(ctx, id)
}

// RemoveMember 移除操作工（集合管理员）：集合管理员不可通过本接口移除
func (s *CollectionService) RemoveMember(ctx context.Context, userID string, id, targetUserID primitive.ObjectID) (*model.BusinessCollection, *errno.Error) {
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errno.ErrParam.WithCause(err)
	}
	if _, err := s.perm.RequireRole(ctx, id, uid, model.MemberRoleCollectionAdmin); err != nil {
		return nil, err
	}
	c, err := s.cols.FindByID(ctx, id)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if c == nil {
		return nil, errno.ErrCollectionNotFound
	}
	role, ok := c.RoleOf(targetUserID)
	if !ok {
		return nil, errno.ErrNotMemberOfCol
	}
	if role == model.MemberRoleCollectionAdmin {
		return nil, errno.ErrCannotRemoveAdmin
	}
	if err := s.cols.RemoveMember(ctx, id, targetUserID); err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	s.audit.Log(ctx, uid, "collection.revoke", "移除成员 "+targetUserID.Hex(), &id, nil)
	return s.findUpdated(ctx, id)
}

// AssignCollectionAdmin 更换集合管理员（admin 专属，公共权限）
func (s *CollectionService) AssignCollectionAdmin(ctx context.Context, actorID, id, newAdminID primitive.ObjectID) (*model.BusinessCollection, *errno.Error) {
	newAdmin, err := s.users.FindByID(ctx, newAdminID)
	if err != nil {
		return nil, errno.ErrInternal.WithCause(err)
	}
	if newAdmin == nil {
		return nil, errno.ErrUserNotFound
	}
	if err := s.cols.ReplaceAdmins(ctx, id, newAdminID); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errno.ErrCollectionNotFound
		}
		return nil, errno.ErrInternal.WithCause(err)
	}
	s.audit.Log(ctx, actorID, "collection.assign_admin", "更换集合管理员为 "+newAdmin.Username, &id, nil)
	return s.findUpdated(ctx, id)
}

// Delete 删除集合（admin 专属）：级联删除数据项与刮削任务元数据，不触碰 NFS 文件
func (s *CollectionService) Delete(ctx context.Context, id primitive.ObjectID) *errno.Error {
	if err := s.items.DeleteByCollection(ctx, id); err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	if err := s.tasks.DeleteByCollection(ctx, id); err != nil {
		return errno.ErrInternal.WithCause(err)
	}
	if err := s.cols.Delete(ctx, id); err != nil {
		if err == mongo.ErrNoDocuments {
			return errno.ErrCollectionNotFound
		}
		return errno.ErrInternal.WithCause(err)
	}
	return nil
}

// MemberRole 查询用户在某集合中的角色（供其他服务复用）
func (s *CollectionService) MemberRole(ctx context.Context, collectionID, userID primitive.ObjectID) (model.MemberRole, *errno.Error) {
	return s.perm.MemberRole(ctx, collectionID, userID)
}
