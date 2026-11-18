package perms

import (
	"testing"

	"github.com/Subilan/go-aliyunmc/internal/common/perm_util"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
)

func TestCatalogDefaults(t *testing.T) {
	if len(All()) != 34 {
		t.Fatalf("permission catalog size = %d, want 34", len(All()))
	}

	for _, code := range DefaultCodes() {
		permission, ok := Get(code)
		if !ok {
			t.Fatalf("default permission %q missing from catalog", code)
		}
		if permission.Scope != ScopeSelf {
			t.Fatalf("default permission %q should be self scoped", code)
		}
	}

	for _, code := range []string{"admin.access", "thread.pin", "settings.manage"} {
		if !Exists(code) {
			t.Fatalf("catalog missing %q", code)
		}
	}
}

func TestEnforcerScopeMatching(t *testing.T) {
	e, err := newEnforcer([][]string{
		{"user:1", "*", "thread.create"},
		{"user:1", "board:2", "thread.pin"},
		{"user:3", "board:4", "thread.lock"},
	})
	if err != nil {
		t.Fatalf("newEnforcer: %v", err)
	}

	cases := []struct {
		sub, dom, perm string
		want           bool
	}{
		{"user:1", "*", "thread.create", true},
		{"user:1", "board:2", "thread.create", true},
		{"user:1", "board:3", "thread.create", true},
		{"user:1", "*", "thread.pin", false},
		{"user:1", "board:2", "thread.pin", true},
		{"user:1", "board:3", "thread.pin", false},
		{"user:3", "board:4", "thread.lock", true},
		{"user:3", "board:5", "thread.lock", false},
	}

	for _, tc := range cases {
		got, err := e.Enforce(tc.sub, tc.dom, tc.perm)
		if err != nil {
			t.Fatalf("Enforce(%s,%s,%s): %v", tc.sub, tc.dom, tc.perm, err)
		}
		if got != tc.want {
			t.Fatalf("Enforce(%s,%s,%s) = %v, want %v", tc.sub, tc.dom, tc.perm, got, tc.want)
		}
	}
}

func TestSuperuserBypass(t *testing.T) {
	superuser := &models.User{Role: perm_util.RoleSuperuser}
	superuser.ID = 99
	if !IsSuperuser(superuser) {
		t.Fatal("superuser not detected")
	}
	if !Allow(superuser, "*", "settings.manage") {
		t.Fatal("superuser should pass global permission")
	}
	if !Allow(superuser, "board:1", "thread.pin") {
		t.Fatal("superuser should pass board permission")
	}
}
