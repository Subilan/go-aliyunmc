package perms

import (
	"testing"

	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"
)

const createCustomInstanceObj = "create-custom-instance"

func TestEmbeddedPolicyLoads(t *testing.T) {
	MustInitialize()

	cases := []struct {
		role perm_util.Role
		obj  string
		act  string
		want bool
	}{
		{perm_util.RoleBasic, "/lab-v1/user/profile", "GET", true},
		{perm_util.RoleBasic, "/lab-v1/user/s", "GET", false},
		{perm_util.RoleBasic, "/lab-v1/user/create", "POST", false},
		{perm_util.RoleOperator, "/lab-v1/user/s", "GET", true},
		{perm_util.RoleOperator, "/lab-v1/user/create", "POST", true},
		{perm_util.RoleOperator, "/lab-v1/user/role", "POST", false},
		{perm_util.RoleSuperuser, "/lab-v1/user/role", "POST", true},
		{perm_util.RoleOperator, "/lab-v1/task/:id", "GET", true},
		{perm_util.RoleBasic, "/lab-v1/task/42/output", "GET", true},
		{perm_util.RoleBasic, "/lab-v1/task/42/interrupt", "POST", false},
		{perm_util.RoleOperator, createCustomInstanceObj, "execute", true},
		{perm_util.RoleBasic, createCustomInstanceObj, "execute", false},
		{perm_util.RoleBasic, "/lab-v1/monitor/auto-archive-idle", "GET", true},
		{perm_util.RoleBasic, "/lab-v1/monitor/auto-archive-idle/pause", "POST", false},
		{perm_util.RoleOperator, "/lab-v1/monitor/auto-archive-idle/pause", "POST", true},
		{perm_util.RoleOperator, "/lab-v1/monitor/auto-archive-idle", "POST", true},
	}

	for _, c := range cases {
		got, err := Enforce(c.role, c.obj, c.act)
		if err != nil {
			t.Fatalf("enforce(%s, %s, %s) failed: %v", c.role, c.obj, c.act, err)
		}
		if got != c.want {
			t.Fatalf("enforce(%s, %s, %s) = %v, want %v", c.role, c.obj, c.act, got, c.want)
		}
	}
}
