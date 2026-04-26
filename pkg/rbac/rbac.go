package rbac

import (
	"context"
	"strings"

	"datacenter/internal/storage"
)

type Permission string

const (
	PermissionUserRead   Permission = "user:read"
	PermissionUserWrite  Permission = "user:write"
	PermissionUserManage Permission = "user:manage"

	PermissionRoleRead   Permission = "role:read"
	PermissionRoleWrite  Permission = "role:write"
	PermissionRoleManage Permission = "role:manage"

	PermissionPermissionRead   Permission = "permission:read"
	PermissionPermissionWrite  Permission = "permission:write"
	PermissionPermissionManage Permission = "permission:manage"

	PermissionCollectionRead   Permission = "collection:read"
	PermissionCollectionWrite  Permission = "collection:write"
	PermissionCollectionManage Permission = "collection:manage"

	PermissionFieldRead   Permission = "field:read"
	PermissionFieldWrite  Permission = "field:write"
	PermissionFieldManage Permission = "field:manage"

	PermissionDataRead   Permission = "data:read"
	PermissionDataWrite  Permission = "data:write"
	PermissionDataManage Permission = "data:manage"

	PermissionScrapeRead   Permission = "scrape:read"
	PermissionScrapeWrite  Permission = "scrape:write"
	PermissionScrapeManage Permission = "scrape:manage"

	PermissionSystemAdmin Permission = "system:admin"
)

var PermissionAll = []Permission{
	PermissionUserRead, PermissionUserWrite, PermissionUserManage,
	PermissionRoleRead, PermissionRoleWrite, PermissionRoleManage,
	PermissionPermissionRead, PermissionPermissionWrite, PermissionPermissionManage,
	PermissionCollectionRead, PermissionCollectionWrite, PermissionCollectionManage,
	PermissionFieldRead, PermissionFieldWrite, PermissionFieldManage,
	PermissionDataRead, PermissionDataWrite, PermissionDataManage,
	PermissionScrapeRead, PermissionScrapeWrite, PermissionScrapeManage,
	PermissionSystemAdmin,
}

type Service struct {
	storage storage.RBACStorage
}

func NewService(s storage.RBACStorage) *Service {
	return &Service{storage: s}
}

func (s *Service) CheckPermission(ctx context.Context, userID string, requiredPermission Permission) (bool, error) {
	user, err := s.storage.GetUserByID(userID)
	if err != nil {
		return false, err
	}

	if len(user.RoleIDs) == 0 {
		return false, nil
	}

	// 检查用户是否拥有超级管理员权限
	for _, roleID := range user.RoleIDs {
		role, err := s.storage.GetRoleByID(roleID)
		if err != nil {
			continue
		}

		for _, permID := range role.PermissionIDs {
			perm, err := s.storage.GetPermissionByID(permID)
			if err != nil {
				continue
			}

			// 如果用户拥有超级管理员权限，则允许访问所有资源
			if perm.Code == string(PermissionSystemAdmin) {
				return true, nil
			}

			// 检查是否匹配所需权限
			if s.matchPermission(perm.Code, string(requiredPermission)) {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *Service) matchPermission(userPermCode, requiredPermCode string) bool {
	if userPermCode == requiredPermCode {
		return true
	}

	if strings.HasSuffix(userPermCode, ":*") {
		prefix := strings.TrimSuffix(userPermCode, "*")
		if strings.HasPrefix(requiredPermCode, prefix) {
			return true
		}
	}

	return false
}

func (s *Service) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	user, err := s.storage.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	permMap := make(map[string]bool)

	for _, roleID := range user.RoleIDs {
		role, err := s.storage.GetRoleByID(roleID)
		if err != nil {
			continue
		}

		for _, permID := range role.PermissionIDs {
			perm, err := s.storage.GetPermissionByID(permID)
			if err != nil {
				continue
			}
			permMap[perm.Code] = true
		}
	}

	perms := make([]string, 0, len(permMap))
	for p := range permMap {
		perms = append(perms, p)
	}

	return perms, nil
}

func (s *Service) HasAnyPermission(ctx context.Context, userID string, requiredPermissions []Permission) (bool, error) {
	for _, perm := range requiredPermissions {
		has, err := s.CheckPermission(ctx, userID, perm)
		if err != nil {
			return false, err
		}
		if has {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) HasAllPermissions(ctx context.Context, userID string, requiredPermissions []Permission) (bool, error) {
	for _, perm := range requiredPermissions {
		has, err := s.CheckPermission(ctx, userID, perm)
		if err != nil {
			return false, err
		}
		if !has {
			return false, nil
		}
	}
	return true, nil
}

func GetAPIPermission(method, path string) Permission {
	path = strings.TrimPrefix(path, "/api/")

	if strings.HasPrefix(path, "users") {
		switch method {
		case "GET":
			return PermissionUserRead
		case "POST", "PUT", "DELETE":
			return PermissionUserWrite
		}
	}

	if strings.HasPrefix(path, "roles") {
		switch method {
		case "GET":
			return PermissionRoleRead
		case "POST", "PUT", "DELETE":
			return PermissionRoleWrite
		}
	}

	if strings.HasPrefix(path, "permissions") {
		switch method {
		case "GET":
			return PermissionPermissionRead
		case "POST", "PUT", "DELETE":
			return PermissionPermissionWrite
		}
	}

	if strings.HasPrefix(path, "collections") {
		switch method {
		case "GET":
			return PermissionCollectionRead
		case "POST", "PUT", "DELETE":
			return PermissionCollectionWrite
		}
	}

	if strings.HasPrefix(path, "fields") {
		switch method {
		case "GET":
			return PermissionFieldRead
		case "POST", "PUT", "DELETE":
			return PermissionFieldWrite
		}
	}

	if strings.HasPrefix(path, "business") {
		switch method {
		case "GET":
			return PermissionDataRead
		case "POST", "PUT", "DELETE":
			return PermissionDataWrite
		}
	}

	if strings.HasPrefix(path, "scraper") || strings.HasPrefix(path, "deleted-scraper") {
		switch method {
		case "GET":
			return PermissionScrapeRead
		case "POST", "PUT", "DELETE":
			return PermissionScrapeWrite
		}
	}

	if strings.HasPrefix(path, "deleted") && !strings.HasPrefix(path, "deleted-scraper") {
		switch method {
		case "GET":
			return PermissionDataRead
		case "POST":
			return PermissionDataWrite
		}
	}

	return PermissionSystemAdmin
}
