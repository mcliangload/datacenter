package rbac

import (
	"context"
	"testing"

	"datacenter/internal/models"
	"datacenter/pkg/rbac"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type mockStorage struct {
	permissions map[string]*models.Permission
	roles       map[string]*models.Role
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		permissions: make(map[string]*models.Permission),
		roles:       make(map[string]*models.Role),
	}
}

func (m *mockStorage) createTestPermission(code string) *models.Permission {
	perm := &models.Permission{
		BaseModel: models.BaseModel{
			ID: primitive.NewObjectID(),
		},
		Code: code,
	}
	m.permissions[code] = perm
	return perm
}

func (m *mockStorage) createTestRole(code string) *models.Role {
	role := &models.Role{
		BaseModel: models.BaseModel{
			ID: primitive.NewObjectID(),
		},
		Code:          code,
		PermissionIDs: []string{},
	}
	m.roles[code] = role
	return role
}

func (m *mockStorage) assignPermissionToRole(roleCode, permissionCode string) {
	role := m.roles[roleCode]
	perm := m.permissions[permissionCode]
	if role != nil && perm != nil {
		role.PermissionIDs = append(role.PermissionIDs, perm.ID.Hex())
	}
}

func (m *mockStorage) CreateUser(user *models.User) error                      { return nil }
func (m *mockStorage) GetUserByID(id string) (*models.User, error)             { return nil, nil }
func (m *mockStorage) GetUserByUsername(username string) (*models.User, error) { return nil, nil }
func (m *mockStorage) UpdateUser(user *models.User) error                      { return nil }
func (m *mockStorage) DeleteUser(id string) error                              { return nil }
func (m *mockStorage) GetUsers(skip, limit int64) ([]models.User, error)       { return nil, nil }
func (m *mockStorage) GetUsersCount() (int64, error)                           { return 0, nil }

func (m *mockStorage) CreateFieldDefinition(field *models.FieldDefinition) error { return nil }
func (m *mockStorage) GetFieldDefinitionByID(id string) (*models.FieldDefinition, error) {
	return nil, nil
}
func (m *mockStorage) GetFieldDefinitionsByModule(module string) ([]models.FieldDefinition, error) {
	return nil, nil
}
func (m *mockStorage) UpdateFieldDefinition(field *models.FieldDefinition) error { return nil }
func (m *mockStorage) DeleteFieldDefinition(id string) error                     { return nil }

func (m *mockStorage) CreateBusinessData(data *models.BusinessData) error          { return nil }
func (m *mockStorage) GetBusinessDataByID(id string) (*models.BusinessData, error) { return nil, nil }
func (m *mockStorage) GetBusinessDataByModule(module string, filter bson.M, skip, limit int64) ([]models.BusinessData, error) {
	return nil, nil
}
func (m *mockStorage) UpdateBusinessData(data *models.BusinessData) error { return nil }
func (m *mockStorage) DeleteBusinessData(id string, userID string) error  { return nil }

func (m *mockStorage) GetDeletedDataByID(id string) (*models.DeletedData, error) { return nil, nil }
func (m *mockStorage) GetDeletedDataByModule(module string, skip, limit int64) ([]models.DeletedData, error) {
	return nil, nil
}
func (m *mockStorage) RecoverDeletedData(id string, userID string) error { return nil }
func (m *mockStorage) CleanupDeletedData(olderThan interface{}) error    { return nil }

func (m *mockStorage) CreatePermission(permission *models.Permission) error { return nil }
func (m *mockStorage) GetPermissionByID(id string) (*models.Permission, error) {
	for _, perm := range m.permissions {
		if perm.ID.Hex() == id {
			return perm, nil
		}
	}
	return nil, mongo.ErrNoDocuments
}
func (m *mockStorage) GetPermissionByCode(code string) (*models.Permission, error) {
	perm, exists := m.permissions[code]
	if !exists {
		return nil, mongo.ErrNoDocuments
	}
	return perm, nil
}
func (m *mockStorage) GetPermissions(skip, limit int64) ([]models.Permission, error) { return nil, nil }
func (m *mockStorage) GetPermissionsCount() (int64, error)                        { return int64(len(m.permissions)), nil }
func (m *mockStorage) UpdatePermission(permission *models.Permission) error          { return nil }
func (m *mockStorage) DeletePermission(id string) error                              { return nil }

func (m *mockStorage) CreateRole(role *models.Role) error { return nil }
func (m *mockStorage) GetRoleByID(id string) (*models.Role, error) {
	for _, role := range m.roles {
		if role.ID.Hex() == id {
			return role, nil
		}
	}
	return nil, mongo.ErrNoDocuments
}
func (m *mockStorage) GetRoleByCode(code string) (*models.Role, error) {
	role, exists := m.roles[code]
	if !exists {
		return nil, mongo.ErrNoDocuments
	}
	return role, nil
}
func (m *mockStorage) GetRoles(skip, limit int64) ([]models.Role, error) { return nil, nil }
func (m *mockStorage) GetRolesCount() (int64, error)                         { return int64(len(m.roles)), nil }
func (m *mockStorage) UpdateRole(role *models.Role) error                { return nil }
func (m *mockStorage) DeleteRole(id string) error                        { return nil }

func (m *mockStorage) AssignRoleToUser(userID, roleID, operatorID string) error { return nil }
func (m *mockStorage) RemoveRoleFromUser(userID, roleID string) error           { return nil }
func (m *mockStorage) GetUserRoles(userID string) ([]models.Role, error)        { return nil, nil }

func (m *mockStorage) AssignPermissionToRole(roleID, permissionID, operatorID string) error {
	return nil
}
func (m *mockStorage) RemovePermissionFromRole(roleID, permissionID string) error { return nil }
func (m *mockStorage) GetRolePermissions(roleID string) ([]models.Permission, error) {
	for _, role := range m.roles {
		if role.ID.Hex() == roleID {
			permissions := make([]models.Permission, 0, len(role.PermissionIDs))
			for _, permID := range role.PermissionIDs {
				for _, perm := range m.permissions {
					if perm.ID.Hex() == permID {
						permissions = append(permissions, *perm)
						break
					}
				}
			}
			return permissions, nil
		}
	}
	return nil, mongo.ErrNoDocuments
}

func (m *mockStorage) InitDefaultData() error { return nil }

func TestCheckPermission(t *testing.T) {
	ms := newMockStorage()

	ms.createTestPermission("manage_users")
	ms.createTestPermission("crud_data")
	ms.createTestPermission("define_fields")
	ms.createTestPermission("grant_dataowner")

	ms.createTestRole("root")
	ms.createTestRole("datatypeowner")
	ms.createTestRole("dataowner")

	ms.assignPermissionToRole("root", "manage_users")
	ms.assignPermissionToRole("root", "crud_data")
	ms.assignPermissionToRole("root", "define_fields")
	ms.assignPermissionToRole("root", "grant_dataowner")

	ms.assignPermissionToRole("datatypeowner", "crud_data")
	ms.assignPermissionToRole("datatypeowner", "define_fields")
	ms.assignPermissionToRole("datatypeowner", "grant_dataowner")

	ms.assignPermissionToRole("dataowner", "crud_data")

	rbacService := rbac.NewService(ms)
	ctx := context.Background()

	testCases := []struct {
		name       string
		roles      []string
		permission string
		expected   bool
	}{
		{
			name:       "root has manage_users",
			roles:      []string{"root"},
			permission: "manage_users",
			expected:   true,
		},
		{
			name:       "root has crud_data",
			roles:      []string{"root"},
			permission: "crud_data",
			expected:   true,
		},
		{
			name:       "datatypeowner has define_fields",
			roles:      []string{"datatypeowner"},
			permission: "define_fields",
			expected:   true,
		},
		{
			name:       "datatypeowner has grant_dataowner",
			roles:      []string{"datatypeowner"},
			permission: "grant_dataowner",
			expected:   true,
		},
		{
			name:       "datatypeowner has crud_data",
			roles:      []string{"datatypeowner"},
			permission: "crud_data",
			expected:   true,
		},
		{
			name:       "datatypeowner does not have manage_users",
			roles:      []string{"datatypeowner"},
			permission: "manage_users",
			expected:   false,
		},
		{
			name:       "dataowner has crud_data",
			roles:      []string{"dataowner"},
			permission: "crud_data",
			expected:   true,
		},
		{
			name:       "dataowner does not have define_fields",
			roles:      []string{"dataowner"},
			permission: "define_fields",
			expected:   false,
		},
		{
			name:       "dataowner does not have manage_users",
			roles:      []string{"dataowner"},
			permission: "manage_users",
			expected:   false,
		},
		{
			name:       "multiple roles - dataowner inherits from datatypeowner",
			roles:      []string{"dataowner", "datatypeowner"},
			permission: "define_fields",
			expected:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := rbacService.CheckPermission(ctx, tc.roles, rbac.Permission(tc.permission))
			if err != nil {
				t.Errorf("CheckPermission() error = %v", err)
				return
			}
			if result != tc.expected {
				t.Errorf("CheckPermission() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestGetUserPermissions(t *testing.T) {
	ms := newMockStorage()

	ms.createTestPermission("manage_users")
	ms.createTestPermission("crud_data")
	ms.createTestPermission("define_fields")
	ms.createTestPermission("grant_dataowner")

	ms.createTestRole("root")
	ms.createTestRole("datatypeowner")
	ms.createTestRole("dataowner")

	ms.assignPermissionToRole("root", "manage_users")
	ms.assignPermissionToRole("root", "crud_data")
	ms.assignPermissionToRole("root", "define_fields")
	ms.assignPermissionToRole("root", "grant_dataowner")

	ms.assignPermissionToRole("datatypeowner", "crud_data")
	ms.assignPermissionToRole("datatypeowner", "define_fields")
	ms.assignPermissionToRole("datatypeowner", "grant_dataowner")

	ms.assignPermissionToRole("dataowner", "crud_data")

	rbacService := rbac.NewService(ms)
	ctx := context.Background()

	testCases := []struct {
		name     string
		roles    []string
		expected int
	}{
		{
			name:     "root permissions",
			roles:    []string{"root"},
			expected: 4,
		},
		{
			name:     "datatypeowner permissions",
			roles:    []string{"datatypeowner"},
			expected: 3,
		},
		{
			name:     "dataowner permissions",
			roles:    []string{"dataowner"},
			expected: 1,
		},
		{
			name:     "multiple roles - union of permissions",
			roles:    []string{"dataowner", "datatypeowner"},
			expected: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			permissions, err := rbacService.GetUserPermissions(ctx, tc.roles)
			if err != nil {
				t.Errorf("GetUserPermissions() error = %v", err)
				return
			}
			if len(permissions) != tc.expected {
				t.Errorf("GetUserPermissions() length = %d, want %d", len(permissions), tc.expected)
			}
		})
	}
}

func TestGetPermissionsByRoleCode(t *testing.T) {
	ms := newMockStorage()

	ms.createTestPermission("manage_users")
	ms.createTestPermission("crud_data")

	ms.createTestRole("testrole")

	ms.assignPermissionToRole("testrole", "manage_users")
	ms.assignPermissionToRole("testrole", "crud_data")

	rbacService := rbac.NewService(ms)
	ctx := context.Background()

	permissions, err := rbacService.GetPermissionsByRoleCode(ctx, "testrole")
	if err != nil {
		t.Errorf("GetPermissionsByRoleCode() error = %v", err)
		return
	}

	if len(permissions) != 2 {
		t.Errorf("GetPermissionsByRoleCode() length = %d, want 2", len(permissions))
	}
}

func TestIsValidRole(t *testing.T) {
	ms := newMockStorage()

	ms.createTestRole("validrole")

	rbacService := rbac.NewService(ms)
	ctx := context.Background()

	valid, err := rbacService.IsValidRole(ctx, "validrole")
	if err != nil {
		t.Errorf("IsValidRole() error = %v", err)
		return
	}
	if !valid {
		t.Errorf("IsValidRole() = %v, want true", valid)
	}

	valid, err = rbacService.IsValidRole(ctx, "invalidrole")
	if err != nil {
		t.Errorf("IsValidRole() error = %v", err)
		return
	}
	if valid {
		t.Errorf("IsValidRole() = %v, want false", valid)
	}
}

func TestIsValidPermission(t *testing.T) {
	ms := newMockStorage()

	ms.createTestPermission("validperm")

	rbacService := rbac.NewService(ms)
	ctx := context.Background()

	valid, err := rbacService.IsValidPermission(ctx, "validperm")
	if err != nil {
		t.Errorf("IsValidPermission() error = %v", err)
		return
	}
	if !valid {
		t.Errorf("IsValidPermission() = %v, want true", valid)
	}

	valid, err = rbacService.IsValidPermission(ctx, "invalidperm")
	if err != nil {
		t.Errorf("IsValidPermission() error = %v", err)
		return
	}
	if valid {
		t.Errorf("IsValidPermission() = %v, want false", valid)
	}
}

func TestManyToManyUserRoleRelationship(t *testing.T) {
	ms := newMockStorage()

	ms.createTestRole("role1")
	ms.createTestRole("role2")
	ms.createTestRole("role3")

	rbacService := rbac.NewService(ms)
	ctx := context.Background()

	roles, err := rbacService.GetUserRoles(ctx, "test-user-id")
	if err != nil {
		t.Errorf("GetUserRoles() error = %v", err)
		return
	}

	if len(roles) != 0 {
		t.Errorf("GetUserRoles() = %d roles, want 0", len(roles))
	}
}

func TestManyToManyRolePermissionRelationship(t *testing.T) {
	ms := newMockStorage()

	ms.createTestPermission("perm1")
	ms.createTestPermission("perm2")
	ms.createTestPermission("perm3")

	ms.createTestRole("testrole")

	ms.assignPermissionToRole("testrole", "perm1")
	ms.assignPermissionToRole("testrole", "perm2")

	rbacService := rbac.NewService(ms)
	ctx := context.Background()

	permissions, err := rbacService.GetPermissionsByRoleCode(ctx, "testrole")
	if err != nil {
		t.Errorf("GetPermissionsByRoleCode() error = %v", err)
		return
	}

	if len(permissions) != 2 {
		t.Errorf("GetPermissionsByRoleCode() = %d permissions, want 2", len(permissions))
	}

	hasPerm1 := false
	hasPerm2 := false
	for _, p := range permissions {
		if p == "perm1" {
			hasPerm1 = true
		}
		if p == "perm2" {
			hasPerm2 = true
		}
	}
	if !hasPerm1 || !hasPerm2 {
		t.Errorf("Missing expected permissions")
	}
}
