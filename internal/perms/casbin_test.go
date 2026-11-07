package perms

import "testing"

const createCustomInstanceObj = "create-custom-instance"

func TestEmbeddedPolicyLoads(t *testing.T) {
	MustInitialize()

	cases := []struct {
		role Role
		obj  string
		act  string
		want bool
	}{
		{RoleBasic, "/user/profile", "GET", true},
		{RoleBasic, "/user/s", "GET", false},
		{RoleOperator, "/user/s", "GET", true},
		{RoleOperator, "/task/:id", "GET", true},
		{RoleBasic, "/task/42/output", "GET", true},
		{RoleBasic, "/task/42/interrupt", "POST", false},
		{RoleOperator, createCustomInstanceObj, "execute", true},
		{RoleBasic, createCustomInstanceObj, "execute", false},
		{RoleBasic, "/monitor/auto-archive-idle", "GET", true},
		{RoleBasic, "/monitor/auto-archive-idle/pause", "POST", false},
		{RoleOperator, "/monitor/auto-archive-idle/pause", "POST", true},
		{RoleOperator, "/monitor/auto-archive-idle", "POST", true},
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
