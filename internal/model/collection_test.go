package model

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRoleOfAndIsMember(t *testing.T) {
	adminID := primitive.NewObjectID()
	opID := primitive.NewObjectID()
	otherID := primitive.NewObjectID()

	c := &BusinessCollection{
		Members: []Member{
			{UserID: adminID, Role: MemberRoleCollectionAdmin},
			{UserID: opID, Role: MemberRoleOperator},
		},
	}

	if role, ok := c.RoleOf(adminID); !ok || role != MemberRoleCollectionAdmin {
		t.Errorf("admin 角色应为 collection_admin，实际 %q ok=%v", role, ok)
	}
	if role, ok := c.RoleOf(opID); !ok || role != MemberRoleOperator {
		t.Errorf("op 角色应为 operator，实际 %q ok=%v", role, ok)
	}
	if _, ok := c.RoleOf(otherID); ok {
		t.Error("非成员不应命中 RoleOf")
	}
	if !c.IsMember(adminID) || !c.IsMember(opID) {
		t.Error("成员应命中 IsMember")
	}
	if c.IsMember(otherID) {
		t.Error("非成员不应命中 IsMember")
	}
}
