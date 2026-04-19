package rbac

import (
	"context"
	"time"

	"datacenter/internal/models"
	"datacenter/internal/storage"
)

type Role string

type Permission string

type AuditLog struct {
	ID        string    `json:"_id" bson:"_id"`
	UserID    string    `json:"user_id" bson:"user_id"`
	Action    string    `json:"action" bson:"action"`
	Resource  string    `json:"resource" bson:"resource"`
	Details   string    `json:"details" bson:"details"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	IP        string    `json:"ip" bson:"ip"`
}

type Service struct {
	storage storage.RBACStorage
}

func NewService(storage storage.RBACStorage) *Service {
	return &Service{
		storage: storage,
	}
}

func (s *Service) CheckPermission(ctx context.Context, roles []string, permission Permission) (bool, error) {
	for _, roleCode := range roles {
		role, err := s.storage.GetRoleByCode(roleCode)
		if err != nil {
			continue
		}

		permissions, err := s.storage.GetRolePermissions(role.ID.Hex())
		if err != nil {
			continue
		}

		for _, perm := range permissions {
			if perm.Code == string(permission) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *Service) GetUserPermissions(ctx context.Context, roles []string) ([]Permission, error) {
	permissionMap := make(map[string]bool)

	for _, roleCode := range roles {
		role, err := s.storage.GetRoleByCode(roleCode)
		if err != nil {
			continue
		}

		permissions, err := s.storage.GetRolePermissions(role.ID.Hex())
		if err != nil {
			continue
		}

		for _, perm := range permissions {
			permissionMap[perm.Code] = true
		}
	}

	permissions := make([]Permission, 0, len(permissionMap))
	for p := range permissionMap {
		permissions = append(permissions, Permission(p))
	}

	return permissions, nil
}

func (s *Service) GetPermissionsByRoleCode(ctx context.Context, roleCode string) ([]Permission, error) {
	role, err := s.storage.GetRoleByCode(roleCode)
	if err != nil {
		return nil, err
	}

	permissions, err := s.storage.GetRolePermissions(role.ID.Hex())
	if err != nil {
		return nil, err
	}

	result := make([]Permission, len(permissions))
	for i, perm := range permissions {
		result[i] = Permission(perm.Code)
	}

	return result, nil
}

func (s *Service) GetUserRoles(ctx context.Context, userID string) ([]models.Role, error) {
	return s.storage.GetUserRoles(userID)
}

func (s *Service) IsValidRole(ctx context.Context, roleCode string) (bool, error) {
	_, err := s.storage.GetRoleByCode(roleCode)
	return err == nil, nil
}

func (s *Service) IsValidPermission(ctx context.Context, permissionCode string) (bool, error) {
	_, err := s.storage.GetPermissionByCode(permissionCode)
	return err == nil, nil
}

func CheckPermission(roles []string, permission Permission) bool {
	return false
}

func GetUserPermissions(roles []string) []Permission {
	return nil
}

func GetPermissionsByRoleCode(roleCode string) []Permission {
	return nil
}

func IsValidRole(roleCode string) bool {
	return false
}

func IsValidPermission(permissionCode string) bool {
	return false
}
