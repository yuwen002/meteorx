package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/common/contextx"

	"github.com/stretchr/testify/assert"
)

// TestRequiresMasterAdmin 确认平台超管网关中间件的行为：
// 仅 superadmin 角色放行，其余（含普通角色、无角色）一律 403。
// 该中间件被 wiki 内联超管端点（/wiki/admin/trash/expire）使用，本测试验证其网关正确性。
func TestRequiresMasterAdmin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := RequiresMasterAdmin()

	cases := []struct {
		name  string
		roles []string
		want  int
	}{
		{"超管放行", []string{"superadmin"}, http.StatusOK},
		{"普通角色拒绝", []string{"member"}, http.StatusForbidden},
		{"无角色拒绝", nil, http.StatusForbidden},
		{"多角色含超管放行", []string{"member", "superadmin"}, http.StatusOK},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := contextx.SetVars(httptest.NewRequest(http.MethodGet, "/x", nil).Context(), "t-1", "u-1", c.roles)
			req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
			w := httptest.NewRecorder()
			mw(next).ServeHTTP(w, req)
			assert.Equal(t, c.want, w.Code)
		})
	}

	// 即使带 scope 严格收敛集合，superadmin 角色仍应直接放行（与 AutoRequirePermission 一致）
	t.Run("带scope的超管仍放行", func(t *testing.T) {
		ctx := contextx.SetScopedPermissions(
			contextx.SetVars(httptest.NewRequest(http.MethodGet, "/x", nil).Context(), "t-1", "u-1", []string{"superadmin"}),
			[]string{"task:read"},
		)
		req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
		w := httptest.NewRecorder()
		mw(next).ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
