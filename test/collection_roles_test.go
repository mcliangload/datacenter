package main

import (
	"context"
	"fmt"
	"testing"

	"datacenter/internal/models"
	"datacenter/internal/storage"
	"datacenter/pkg/rbac"
)

func TestCollectionRolesCreation(t *testing.T) {
	mongoURI := "mongodb://localhost:27017"
	testDB := "datacenter_test"

	rbacStorage, err := storage.NewRBACMongoDBStorage(mongoURI, testDB)
	if err != nil {
		t.Fatalf("Failed to create RBAC storage: %v", err)
	}
	collectionRBACStorage, err := storage.NewCollectionRBACStorage(mongoURI, testDB)
	if err != nil {
		t.Fatalf("Failed to create collection RBAC storage: %v", err)
	}

	rbacService := rbac.NewService(rbacStorage)
	collectionRBACService := rbac.NewCollectionRBACService(rbacStorage, collectionRBACStorage)

	testModule := "test_collection"

	cleanupTestData(t, collectionRBACStorage, rbacStorage, testModule)

	t.Run("CreateCollectionRoles", func(t *testing.T) {
		err := collectionRBACService.CreateCollectionRoles(context.Background(), testModule, "test_operator")
		if err != nil {
			t.Fatalf("Failed to create collection roles: %v", err)
		}

		roles, err := collectionRBACStorage.GetCollectionRolesByModule(testModule)
		if err != nil {
			t.Fatalf("Failed to get collection roles: %v", err)
		}

		if len(roles) != 3 {
			t.Errorf("Expected 3 roles, got %d", len(roles))
		}

		expectedRoles := []struct {
			code string
			name string
		}{
			{fmt.Sprintf("%sOwner", testModule), fmt.Sprintf("%sOwner", testModule)},
			{fmt.Sprintf("%sOperator", testModule), fmt.Sprintf("%sOperator", testModule)},
			{fmt.Sprintf("%sTourist", testModule), fmt.Sprintf("%sTourist", testModule)},
		}

		roleMap := make(map[string]models.CollectionRole)
		for _, role := range roles {
			roleMap[role.Code] = role
		}

		for _, expected := range expectedRoles {
			role, exists := roleMap[expected.code]
			if !exists {
				t.Errorf("Role %s not found", expected.code)
				continue
			}
			if role.Name != expected.name {
				t.Errorf("Expected role name %s, got %s", expected.name, role.Name)
			}
			if role.CollectionModule != testModule {
				t.Errorf("Expected collection module %s, got %s", testModule, role.CollectionModule)
			}
		}

		ownerRole := roleMap[fmt.Sprintf("%sOwner", testModule)]
		expectedOwnerPerms := []string{
			testModule + ":admin",
			testModule + ":read",
			testModule + ":write",
			testModule + ":delete",
			testModule + ":field:admin",
		}
		if !hasAllPermissions(ownerRole.PermissionIDs, expectedOwnerPerms) {
			t.Errorf("Owner role missing expected permissions. Got: %v, Expected: %v", ownerRole.PermissionIDs, expectedOwnerPerms)
		}

		operatorRole := roleMap[fmt.Sprintf("%sOperator", testModule)]
		expectedOperatorPerms := []string{
			testModule + ":read",
			testModule + ":write",
			testModule + ":delete",
		}
		if !hasAllPermissions(operatorRole.PermissionIDs, expectedOperatorPerms) {
			t.Errorf("Operator role missing expected permissions. Got: %v, Expected: %v", operatorRole.PermissionIDs, expectedOperatorPerms)
		}

		touristRole := roleMap[fmt.Sprintf("%sTourist", testModule)]
		expectedTouristPerms := []string{
			testModule + ":read",
		}
		if !hasAllPermissions(touristRole.PermissionIDs, expectedTouristPerms) {
			t.Errorf("Tourist role missing expected permissions. Got: %v, Expected: %v", touristRole.PermissionIDs, expectedTouristPerms)
		}
	})
}

func hasAllPermissions(actual, expected []string) bool {
	actualMap := make(map[string]bool)
	for _, perm := range actual {
		actualMap[perm] = true
	}
	for _, perm := range expected {
		if !actualMap[perm] {
			return false
		}
	}
	return true
}

func cleanupTestData(t *testing.T, crStorage storage.CollectionRBACStorage, rbacStorage storage.RBACStorage, module string) {
	roles, err := crStorage.GetCollectionRolesByModule(module)
	if err == nil {
		for _, role := range roles {
			crStorage.DeleteCollectionRole(role.ID.Hex())
		}
	}
}
