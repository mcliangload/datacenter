package rbac

import (
	"time"
)

// Role 角色定义
type Role string

const (
	// RoleRoot 系统最高权限
	RoleRoot Role = "root"
	// RoleDataTypeOwner 数据库类型所有者
	RoleDataTypeOwner Role = "datatypeowner"
	// RoleDataOwner 数据所有者
	RoleDataOwner Role = "dataowner"
)

// Permission 权限定义
type Permission string

const (
	// PermissionManageUsers 管理用户
	PermissionManageUsers Permission = "manage_users"
	// PermissionManageDatabases 管理数据库
	PermissionManageDatabases Permission = "manage_databases"
	// PermissionDefineFields 定义字段
	PermissionDefineFields Permission = "define_fields"
	// PermissionGrantDataOwner 授予dataowner权限
	PermissionGrantDataOwner Permission = "grant_dataowner"
	// PermissionCRUDData 对数据进行增删改查
	PermissionCRUDData Permission = "crud_data"
)

// AuditLog 审计日志
type AuditLog struct {
	ID        string    `json:"_id" bson:"_id"`
	UserID    string    `json:"user_id" bson:"user_id"`
	Action    string    `json:"action" bson:"action"`
	Resource  string    `json:"resource" bson:"resource"`
	Details   string    `json:"details" bson:"details"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	IP        string    `json:"ip" bson:"ip"`
}

// RolePermissions 角色权限映射（仅作为缓存参考，实际数据从数据库读取）
var RolePermissions = map[Role][]Permission{
	RoleRoot: {
		PermissionManageUsers,
		PermissionManageDatabases,
		PermissionDefineFields,
		PermissionGrantDataOwner,
		PermissionCRUDData,
	},
	RoleDataTypeOwner: {
		PermissionDefineFields,
		PermissionGrantDataOwner,
		PermissionCRUDData,
	},
	RoleDataOwner: {
		PermissionCRUDData,
	},
}

// DefaultRoles 默认角色定义
var DefaultRoles = []string{"root", "datatypeowner", "dataowner"}

// DefaultPermissions 默认权限定义
var DefaultPermissions = []string{
	"manage_users",
	"manage_databases",
	"define_fields",
	"grant_dataowner",
	"crud_data",
}

// CheckPermission 检查用户是否具有指定权限（基于内存中的角色权限映射）
func CheckPermission(roles []string, permission Permission) bool {
	for _, role := range roles {
		if permissions, ok := RolePermissions[Role(role)]; ok {
			for _, p := range permissions {
				if p == permission {
					return true
				}
			}
		}
	}
	return false
}

// GetUserPermissions 获取用户所有权限（基于内存中的角色权限映射）
func GetUserPermissions(roles []string) []Permission {
	permissionMap := make(map[Permission]bool)
	for _, role := range roles {
		if permissions, ok := RolePermissions[Role(role)]; ok {
			for _, p := range permissions {
				permissionMap[p] = true
			}
		}
	}

	permissions := make([]Permission, 0, len(permissionMap))
	for p := range permissionMap {
		permissions = append(permissions, p)
	}

	return permissions
}

// GetPermissionsByRoleCode 根据角色代码获取权限列表
func GetPermissionsByRoleCode(roleCode string) []Permission {
	if permissions, ok := RolePermissions[Role(roleCode)]; ok {
		return permissions
	}
	return nil
}

// IsValidRole 检查角色代码是否有效
func IsValidRole(roleCode string) bool {
	for _, r := range DefaultRoles {
		if r == roleCode {
			return true
		}
	}
	return false
}

// IsValidPermission 检查权限代码是否有效
func IsValidPermission(permissionCode string) bool {
	for _, p := range DefaultPermissions {
		if p == permissionCode {
			return true
		}
	}
	return false
}
