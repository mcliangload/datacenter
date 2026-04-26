package rbac

import (
	"context"
	"fmt"
	"time"

	"datacenter/internal/logger"
	"datacenter/internal/models"
	"datacenter/internal/storage"
)

type CollectionPermission string

const (
	CollectionPermissionAdmin      CollectionPermission = ":admin"
	CollectionPermissionRead       CollectionPermission = ":read"
	CollectionPermissionWrite      CollectionPermission = ":write"
	CollectionPermissionDelete     CollectionPermission = ":delete"
	CollectionPermissionFieldAdmin CollectionPermission = ":field:admin"
)

const (
	SystemPermissionRoot = "system:admin"
)

type CollectionRBACService struct {
	rbacStorage           storage.RBACStorage
	collectionRBACStorage storage.CollectionRBACStorage
}

func NewCollectionRBACService(rbacStorage storage.RBACStorage, collectionRBACStorage storage.CollectionRBACStorage) *CollectionRBACService {
	return &CollectionRBACService{
		rbacStorage:           rbacStorage,
		collectionRBACStorage: collectionRBACStorage,
	}
}

func (s *CollectionRBACService) CheckCollectionPermission(ctx context.Context, userID, module string, requiredPermission CollectionPermission) (bool, error) {
	user, err := s.rbacStorage.GetUserByID(userID)
	if err != nil {
		return false, err
	}

	for _, roleID := range user.RoleIDs {
		role, err := s.rbacStorage.GetRoleByID(roleID)
		if err != nil {
			continue
		}
		for _, permID := range role.PermissionIDs {
			perm, err := s.rbacStorage.GetPermissionByID(permID)
			logger.Error("perm is :%s", CollectionPermission(perm.Code))
			// 检查系统权限（如 system:admin）
			if perm.Code == SystemPermissionRoot {
				return true, nil
			}
			// 检查集合权限（如 movie:read）
			if err == nil {
				// 情况1：系统权限代码直接匹配（如 ":read" == ":read"）
				if requiredPermission == CollectionPermission(perm.Code) {
					return true, nil
				}
				// 情况2：系统权限代码包含模块名（如 "movie:read"）
				if string(requiredPermission) == perm.Code[len(module):] {
					return true, nil
				}
			}
		}
	}

	collectionRoleAssignment, err := s.collectionRBACStorage.GetUserCollectionRole(userID, module)
	if err != nil || collectionRoleAssignment == nil {
		return false, nil
	}

	collectionRole, err := s.collectionRBACStorage.GetCollectionRoleByID(collectionRoleAssignment.CollectionRoleID)
	if err != nil {
		return false, err
	}

	fullPermissionCode := module + string(requiredPermission)

	for _, permID := range collectionRole.PermissionIDs {
		if permID == fullPermissionCode {
			return true, nil
		}
	}

	return false, nil
}

func (s *CollectionRBACService) CreateCollectionRoles(ctx context.Context, module, operatorID string) error {
	permDefs := []struct {
		Code string
		Name string
	}{
		{Code: module + ":read", Name: module + " 读取"},
		{Code: module + ":write", Name: module + " 写入"},
		{Code: module + ":delete", Name: module + " 删除"},
		{Code: module + ":admin", Name: module + " 管理"},
		{Code: module + ":field:admin", Name: module + " 字段管理"},
	}

	// 创建权限并记录其真实 ObjectID
	createdPermIDs := make(map[string]string)
	for _, pd := range permDefs {
		perm := &models.Permission{
			Name:        pd.Name,
			Code:        pd.Code,
			Description: pd.Name + " 权限",
		}
		perm.CreatedBy = operatorID
		perm.CreatedAt = time.Now()
		perm.UpdatedAt = time.Now()
		if err := s.rbacStorage.CreatePermission(perm); err != nil {
			return fmt.Errorf("failed to create permission %s: %w", pd.Code, err)
		}
		createdPermIDs[pd.Code] = perm.ID.Hex()
	}

	// 角色模板 + 对应的权限ObjectID列表
	type roleTemplate struct {
		Type        string
		Code        string
		Name        string
		Description string
		PermCodes   []string
	}
	roleTemplates := []roleTemplate{
		{
			Type:        models.CollectionRoleTypeOwner,
			Code:        module + "Owner",
			Name:        module + "集合管理员",
			Description: "拥有" + module + "集合的所有权限，包括管理自定义字段",
			PermCodes:   []string{module + ":admin", module + ":read", module + ":write", module + ":delete", module + ":field:admin"},
		},
		{
			Type:        models.CollectionRoleTypeOperator,
			Code:        module + "Operator",
			Name:        module + "数据操作员",
			Description: "拥有" + module + "集合数据的增删改查权限，不能修改自定义字段",
			PermCodes:   []string{module + ":read", module + ":write", module + ":delete"},
		},
		{
			Type:        models.CollectionRoleTypeTourist,
			Code:        module + "Tourist",
			Name:        module + "普通用户",
			Description: "仅拥有" + module + "集合数据的读取权限",
			PermCodes:   []string{module + ":read"},
		},
	}

	for _, rt := range roleTemplates {
		// 收集权限 ObjectID
		permObjIDs := make([]string, len(rt.PermCodes))
		for i, pc := range rt.PermCodes {
			permObjIDs[i] = createdPermIDs[pc]
		}

		// 1. 在 rbac.roles 表创建系统角色
		sysRole := &models.Role{
			Name:          rt.Name,
			Code:          rt.Code,
			Description:   rt.Description,
			PermissionIDs: permObjIDs,
		}
		sysRole.CreatedBy = operatorID
		sysRole.CreatedAt = time.Now()
		sysRole.UpdatedAt = time.Now()
		if err := s.rbacStorage.CreateRole(sysRole); err != nil {
			return fmt.Errorf("failed to create system role %s: %w", rt.Code, err)
		}

		// 2. 在 rbac.collection_roles 表创建集合角色（权限code用于中间件匹配）
		permCodes := make([]string, len(rt.PermCodes))
		for i, pc := range rt.PermCodes {
			permCodes[i] = pc
		}
		colRole := &models.CollectionRole{
			CollectionModule: module,
			Name:             rt.Name,
			Code:             rt.Code,
			Type:             rt.Type,
			Description:      rt.Description,
			PermissionIDs:    permCodes,
			CreatedBy:        operatorID,
		}
		if err := s.collectionRBACStorage.CreateCollectionRole(colRole); err != nil {
			return fmt.Errorf("failed to create collection role %s: %w", rt.Code, err)
		}
	}

	return nil
}

func (s *CollectionRBACService) DeleteCollectionRoles(ctx context.Context, module string) error {
	roles, err := s.collectionRBACStorage.GetCollectionRolesByModule(module)
	if err != nil {
		return err
	}

	for _, role := range roles {
		if err := s.collectionRBACStorage.DeleteCollectionRole(role.ID.Hex()); err != nil {
			return err
		}

		// 同步删除 rbac.roles 中的系统角色
		sysRole, err := s.rbacStorage.GetRoleByCode(role.Code)
		if err == nil && sysRole != nil {
			s.rbacStorage.DeleteRole(sysRole.ID.Hex())
		}
	}

	return nil
}

func (s *CollectionRBACService) AssignCollectionRole(ctx context.Context, userID, module, roleID, operatorID string) error {
	// 1. 分配集合角色
	assignment := &models.CollectionRoleAssignment{
		UserID:           userID,
		CollectionModule: module,
		CollectionRoleID: roleID,
		CreatedBy:        operatorID,
	}
	if err := s.collectionRBACStorage.AssignCollectionRole(assignment); err != nil {
		return err
	}

	// 2. 同步分配 rbac.roles 中的系统角色给用户
	colRole, err := s.collectionRBACStorage.GetCollectionRoleByID(roleID)
	if err == nil && colRole != nil {
		sysRole, rErr := s.rbacStorage.GetRoleByCode(colRole.Code)
		if rErr == nil && sysRole != nil {
			s.rbacStorage.AssignRoleToUser(userID, sysRole.ID.Hex(), operatorID)
		}
	}

	return nil
}

func (s *CollectionRBACService) RemoveCollectionRole(ctx context.Context, userID, module, roleID, operatorID string) error {
	// 同步移除 rbac.roles 中的系统角色
	colRole, err := s.collectionRBACStorage.GetCollectionRoleByID(roleID)
	if err == nil && colRole != nil {
		sysRole, rErr := s.rbacStorage.GetRoleByCode(colRole.Code)
		if rErr == nil && sysRole != nil {
			s.rbacStorage.RemoveRoleFromUser(userID, sysRole.ID.Hex())
		}
	}

	return s.collectionRBACStorage.RemoveCollectionRoleAssignment(userID, module, roleID)
}

func (s *CollectionRBACService) GetUserCollectionRoles(ctx context.Context, userID string) ([]models.CollectionRole, error) {
	assignments, err := s.collectionRBACStorage.GetUserCollectionRoles(userID)
	if err != nil {
		return nil, err
	}

	var roles []models.CollectionRole
	for _, assignment := range assignments {
		role, err := s.collectionRBACStorage.GetCollectionRoleByID(assignment.CollectionRoleID)
		if err != nil {
			continue
		}
		roles = append(roles, *role)
	}

	return roles, nil
}

func (s *CollectionRBACService) GetCollectionRoles(ctx context.Context, module string) ([]models.CollectionRole, error) {
	return s.collectionRBACStorage.GetCollectionRolesByModule(module)
}

func (s *CollectionRBACService) GetCollectionRoleAssignments(ctx context.Context, module string) ([]models.CollectionRoleAssignment, error) {
	return s.collectionRBACStorage.GetCollectionRoleAssignments(module)
}

func (s *CollectionRBACService) LogAction(ctx context.Context, userID, username, action, resource, resourceID, details, ipAddress, userAgent string) error {
	log := &models.AuditLog{
		UserID:     userID,
		Username:   username,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	}

	return s.collectionRBACStorage.CreateAuditLog(log)
}
