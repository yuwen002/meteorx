package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/common/contextx"
)

// fakePermissionChecker PermissionChecker 的测试桩。
type fakePermissionChecker struct {
	codes []string
	err   error
}

func (f *fakePermissionChecker) GetUserPermissionCodes(context.Context, string) ([]string, error) {
	return f.codes, f.err
}

func okNext() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

// TestPermissionGranted_StrictModeIgnoresSuperadminBypass 带 scope 的严格模式下即使有超管角色也只认收敛集合。
func TestPermissionGranted_StrictModeIgnoresSuperadminBypass(t *testing.T) {
	ctx := contextx.SetScopedPermissions(
		contextx.SetVars(context.Background(), "t-1", "u-1", []string{"superadmin"}),
		[]string{"user:list"},
	)
	checker := &fakePermissionChecker{err: errors.New("should not be called")}

	if ok, _ := permissionGranted(ctx, checker, "u-1", "user:list"); !ok {
		t.Fatalf("scope 内权限应放行")
	}
	if ok, _ := permissionGranted(ctx, checker, "u-1", "user:create"); ok {
		t.Fatalf("scope 外权限应拒绝，即使属主为超管")
	}
}

// TestPermissionGranted_EmptyScopedSetDeniesAll scope 收敛集合为空时拒绝一切。
func TestPermissionGranted_EmptyScopedSetDeniesAll(t *testing.T) {
	ctx := contextx.SetScopedPermissions(
		contextx.SetVars(context.Background(), "t-1", "u-1", []string{"superadmin"}),
		[]string{},
	)
	if ok, _ := permissionGranted(ctx, &fakePermissionChecker{}, "u-1", "user:list"); ok {
		t.Fatalf("空 scope 集合应拒绝")
	}
}

// TestPermissionGranted_SuperadminBypass 无 scope 时超管旁路放行。
func TestPermissionGranted_SuperadminBypass(t *testing.T) {
	ctx := contextx.SetVars(context.Background(), "t-1", "u-1", []string{"superadmin"})
	if ok, _ := permissionGranted(ctx, &fakePermissionChecker{}, "u-1", "anything:code"); !ok {
		t.Fatalf("无 scope 超管应旁路放行")
	}
}

// TestPermissionGranted_NormalUserLookup 普通用户按实时权限码判定。
func TestPermissionGranted_NormalUserLookup(t *testing.T) {
	ctx := contextx.SetVars(context.Background(), "t-1", "u-1", []string{"member"})
	checker := &fakePermissionChecker{codes: []string{"user:list"}}
	if ok, _ := permissionGranted(ctx, checker, "u-1", "user:list"); !ok {
		t.Fatalf("拥有该权限应放行")
	}
	if ok, _ := permissionGranted(ctx, checker, "u-1", "user:delete"); ok {
		t.Fatalf("缺少该权限应拒绝")
	}
}

// TestPermissionGranted_QueryErrorPropagates 权限查询失败返回系统错误。
func TestPermissionGranted_QueryErrorPropagates(t *testing.T) {
	ctx := contextx.SetVars(context.Background(), "t-1", "u-1", []string{"member"})
	_, err := permissionGranted(ctx, &fakePermissionChecker{err: errors.New("db down")}, "u-1", "user:list")
	if err == nil {
		t.Fatalf("期望返回错误")
	}
}

// TestRequirePermission_StrictModeDeniesSuperadminHTTP 端到端验证严格模式下 403。
func TestRequirePermission_StrictModeDeniesSuperadminHTTP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	ctx := contextx.SetScopedPermissions(
		contextx.SetVars(req.Context(), "t-1", "u-1", []string{"superadmin"}),
		[]string{"task:read"}, // 不含 user:list
	)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	RequirePermission(&fakePermissionChecker{}, "user:list")(okNext()).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("scope 外应 403，实际 %d", rr.Code)
	}
}

// TestMatchAllowedPath 路径段对齐匹配的各种边界。
func TestMatchAllowedPath(t *testing.T) {
	cases := []struct {
		path    string
		allowed []string
		want    bool
	}{
		{"/api/v1/users", []string{"/api/v1/users"}, true},
		{"/api/v1/users/123", []string{"/api/v1/users"}, true},
		{"/api/v1/users-admin", []string{"/api/v1/users"}, false}, // 不应被前缀误配
		{"/api/v1/users", []string{"/api/v1/users/"}, true},      // 尾部斜杠归一
		{"/api/v1/anything", []string{"/"}, true},                // "/" 放行全部
		{"/api/v1/users", nil, false},
		{"/api/v1/users", []string{""}, false},
	}
	for _, c := range cases {
		if got := matchAllowedPath(c.path, c.allowed); got != c.want {
			t.Errorf("matchAllowedPath(%q, %v) = %v, want %v", c.path, c.allowed, got, c.want)
		}
	}
}
