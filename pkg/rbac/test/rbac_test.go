package rbac

import (
	"testing"

	"datacenter/pkg/rbac"
)

func TestCheckPermission(t *testing.T) {
	testCases := []struct {
		name       string
		roles      []string
		permission rbac.Permission
		expected   bool
	}{
		{
			name:       "root has manage_users",
			roles:      []string{"root"},
			permission: rbac.PermissionManageUsers,
			expected:   true,
		},
		{
			name:       "root has crud_data",
			roles:      []string{"root"},
			permission: rbac.PermissionCRUDData,
			expected:   true,
		},
		{
			name:       "datatypeowner has define_fields",
			roles:      []string{"datatypeowner"},
			permission: rbac.PermissionDefineFields,
			expected:   true,
		},
		{
			name:       "datatypeowner has grant_dataowner",
			roles:      []string{"datatypeowner"},
			permission: rbac.PermissionGrantDataOwner,
			expected:   true,
		},
		{
			name:       "datatypeowner has crud_data",
			roles:      []string{"datatypeowner"},
			permission: rbac.PermissionCRUDData,
			expected:   true,
		},
		{
			name:       "datatypeowner does not have manage_users",
			roles:      []string{"datatypeowner"},
			permission: rbac.PermissionManageUsers,
			expected:   false,
		},
		{
			name:       "dataowner has crud_data",
			roles:      []string{"dataowner"},
			permission: rbac.PermissionCRUDData,
			expected:   true,
		},
		{
			name:       "dataowner does not have define_fields",
			roles:      []string{"dataowner"},
			permission: rbac.PermissionDefineFields,
			expected:   false,
		},
		{
			name:       "dataowner does not have manage_users",
			roles:      []string{"dataowner"},
			permission: rbac.PermissionManageUsers,
			expected:   false,
		},
		{
			name:       "multiple roles",
			roles:      []string{"dataowner", "datatypeowner"},
			permission: rbac.PermissionDefineFields,
			expected:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := rbac.CheckPermission(tc.roles, tc.permission)
			if result != tc.expected {
				t.Errorf("CheckPermission() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestGetUserPermissions(t *testing.T) {
	testCases := []struct {
		name     string
		roles    []string
		expected int
	}{
		{
			name:     "root permissions",
			roles:    []string{"root"},
			expected: 5,
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
			name:     "multiple roles",
			roles:    []string{"dataowner", "datatypeowner"},
			expected: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			permissions := rbac.GetUserPermissions(tc.roles)
			if len(permissions) != tc.expected {
				t.Errorf("GetUserPermissions() length = %d, want %d", len(permissions), tc.expected)
			}
		})
	}
}
