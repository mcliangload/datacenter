package service

import (
	"testing"

	"datacenter/internal/model"
)

func TestRoleSatisfies(t *testing.T) {
	cases := []struct {
		name     string
		actual   model.MemberRole
		required model.MemberRole
		want     bool
	}{
		{"集合管理员满足操作工", model.MemberRoleCollectionAdmin, model.MemberRoleOperator, true},
		{"操作工满足操作工", model.MemberRoleOperator, model.MemberRoleOperator, true},
		{"集合管理员满足集合管理员", model.MemberRoleCollectionAdmin, model.MemberRoleCollectionAdmin, true},
		{"操作工不满足集合管理员", model.MemberRoleOperator, model.MemberRoleCollectionAdmin, false},
		{"非成员不满足任何角色", "", model.MemberRoleOperator, false},
		{"非成员不满足集合管理员", "", model.MemberRoleCollectionAdmin, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := roleSatisfies(c.actual, c.required); got != c.want {
				t.Fatalf("roleSatisfies(%q, %q) = %v, 期望 %v", c.actual, c.required, got, c.want)
			}
		})
	}
}
