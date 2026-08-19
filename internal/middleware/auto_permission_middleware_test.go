package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// buildTestRouter 构建一个可用于 derivePermissionCode 测试的 chi.Router
// 它会注册与实际路由一致的 pattern，以便 chi.RouteContext 能正确识别
func buildTestRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		// 租户侧用户路由
		r.Route("/users", func(r chi.Router) {
			r.Get("/", func(_ http.ResponseWriter, _ *http.Request) {})
			r.Post("/", func(_ http.ResponseWriter, _ *http.Request) {})
			r.Get("/deleted", func(_ http.ResponseWriter, _ *http.Request) {})
			r.Get("/{id}/detail", func(_ http.ResponseWriter, _ *http.Request) {})
			r.Put("/{id}/update", func(_ http.ResponseWriter, _ *http.Request) {})
			r.Put("/{id}/reset-password", func(_ http.ResponseWriter, _ *http.Request) {})
			r.Delete("/{id}/delete", func(_ http.ResponseWriter, _ *http.Request) {})
			r.Put("/{id}/restore", func(_ http.ResponseWriter, _ *http.Request) {})
			r.Delete("/{id}/permanent", func(_ http.ResponseWriter, _ *http.Request) {})
			r.Put("/batch/status", func(_ http.ResponseWriter, _ *http.Request) {})
			r.Delete("/batch/delete", func(_ http.ResponseWriter, _ *http.Request) {})
		})

		// RBAC 路由
		r.Route("/rbac", func(r chi.Router) {
			r.Route("/roles", func(r chi.Router) {
				r.Get("/", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Post("/", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Get("/{id}/detail", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{id}/update", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Delete("/{id}/delete", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{id}/permissions", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Delete("/{id}/permissions", func(_ http.ResponseWriter, _ *http.Request) {})
			})
			r.Route("/user-roles", func(r chi.Router) {
				r.Get("/", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Post("/{user_id}/roles", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Get("/{user_id}/roles", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Delete("/{user_id}/roles", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Delete("/{user_id}/roles/{role_id}", func(_ http.ResponseWriter, _ *http.Request) {})
			})
		})

		// Admin 路由（平台后台）
		r.Route("/admin", func(r chi.Router) {
			// 系统管理员
			r.Route("/users", func(r chi.Router) {
				r.Get("/", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Post("/", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Get("/deleted", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Get("/{id}/detail", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{id}/update", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{id}/status", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Delete("/{id}/delete", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{id}/restore", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Delete("/{id}/permanent", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/batch/status", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Delete("/batch/delete", func(_ http.ResponseWriter, _ *http.Request) {})
			})

			// 租户管理
			r.Route("/tenants", func(r chi.Router) {
				r.Get("/", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Post("/", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Get("/deleted", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Get("/{id}/detail", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{id}/update", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Delete("/{id}/delete", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{id}/status", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{id}/restore", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/batch/status", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Delete("/batch", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{id}/plan", func(_ http.ResponseWriter, _ *http.Request) {})
			})

			// 跨租户用户管理
			r.Route("/tenant-users", func(r chi.Router) {
				r.Post("/", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Get("/all", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Get("/deleted/all", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Get("/{tenantID}/list", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Get("/{tenantID}/deleted", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{tenantID}/{userID}/update", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{tenantID}/{userID}/status", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{tenantID}/{userID}/reset-password", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{tenantID}/{userID}/restore", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Delete("/{tenantID}/{userID}/delete", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Delete("/{tenantID}/{userID}/permanent", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{tenantID}/batch/status", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Delete("/{tenantID}/batch/delete", func(_ http.ResponseWriter, _ *http.Request) {})
			})

			// 套餐管理
			r.Route("/plans", func(r chi.Router) {
				r.Get("/", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Post("/", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Get("/select", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Put("/{id}/update", func(_ http.ResponseWriter, _ *http.Request) {})
				r.Delete("/{id}/delete", func(_ http.ResponseWriter, _ *http.Request) {})
			})
		})
	})
	return r
}

// testCase 表示一个权限码推导测试用例
type testCase struct {
	name   string
	method string
	path   string
	want   string
}

// resolveRoute 让 chi 执行一次路由匹配，使 RouteContext 可用
func resolveRoute(r *chi.Mux, method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return req
}

func TestDerivePermissionCode_TenantUser(t *testing.T) {
	r := buildTestRouter()

	cases := []testCase{
		{"list", http.MethodGet, "/api/v1/users", "user:list"},
		{"create", http.MethodPost, "/api/v1/users", "user:create"},
		{"list_deleted", http.MethodGet, "/api/v1/users/deleted", "user:list_deleted"},
		{"read", http.MethodGet, "/api/v1/users/{id}/detail", "user:read"},
		{"update", http.MethodPut, "/api/v1/users/{id}/update", "user:update"},
		{"reset_password", http.MethodPut, "/api/v1/users/{id}/reset-password", "user:reset_password"},
		{"delete", http.MethodDelete, "/api/v1/users/{id}/delete", "user:delete"},
		{"restore", http.MethodPut, "/api/v1/users/{id}/restore", "user:restore"},
		{"permanent_delete", http.MethodDelete, "/api/v1/users/{id}/permanent", "user:permanent_delete"},
		{"batch_status", http.MethodPut, "/api/v1/users/batch/status", "user:batch_status"},
		{"batch_delete", http.MethodDelete, "/api/v1/users/batch/delete", "user:batch_delete"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := resolveRoute(r, c.method, c.path)
			got := derivePermissionCode(req)
			if got != c.want {
				t.Errorf("derivePermissionCode(%s %s) = %q, want %q", c.method, c.path, got, c.want)
			}
		})
	}
}

func TestDerivePermissionCode_RBAC(t *testing.T) {
	r := buildTestRouter()

	cases := []testCase{
		{"role list", http.MethodGet, "/api/v1/rbac/roles", "rbac:role:list"},
		{"role create", http.MethodPost, "/api/v1/rbac/roles", "rbac:role:create"},
		{"role read", http.MethodGet, "/api/v1/rbac/roles/{id}/detail", "rbac:role:read"},
		{"role update", http.MethodPut, "/api/v1/rbac/roles/{id}/update", "rbac:role:update"},
		{"role delete", http.MethodDelete, "/api/v1/rbac/roles/{id}/delete", "rbac:role:delete"},
		{"role bind_perm", http.MethodPut, "/api/v1/rbac/roles/{id}/permissions", "rbac:role:bind_perm"},
		{"role unbind_perm", http.MethodDelete, "/api/v1/rbac/roles/{id}/permissions", "rbac:role:unbind_perm"},
		{"user_role list", http.MethodGet, "/api/v1/rbac/user-roles", "rbac:user_role:list"},
		{"user_role assign", http.MethodPost, "/api/v1/rbac/user-roles/{user_id}/roles", "rbac:user_role:assign"},
		{"user_role get_roles", http.MethodGet, "/api/v1/rbac/user-roles/{user_id}/roles", "rbac:user_role:get_roles"},
		{"user_role remove_all", http.MethodDelete, "/api/v1/rbac/user-roles/{user_id}/roles", "rbac:user_role:remove_all"},
		{"user_role remove_one", http.MethodDelete, "/api/v1/rbac/user-roles/{user_id}/roles/{role_id}", "rbac:user_role:remove_one"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := resolveRoute(r, c.method, c.path)
			got := derivePermissionCode(req)
			if got != c.want {
				t.Errorf("derivePermissionCode(%s %s) = %q, want %q", c.method, c.path, got, c.want)
			}
		})
	}
}

func TestDerivePermissionCode_AdminMaster(t *testing.T) {
	r := buildTestRouter()

	cases := []testCase{
		{"master list", http.MethodGet, "/api/v1/admin/users", "admin:master:list"},
		{"master create", http.MethodPost, "/api/v1/admin/users", "admin:master:create"},
		{"master list_deleted", http.MethodGet, "/api/v1/admin/users/deleted", "admin:master:list_deleted"},
		{"master read", http.MethodGet, "/api/v1/admin/users/{id}/detail", "admin:master:read"},
		{"master update", http.MethodPut, "/api/v1/admin/users/{id}/update", "admin:master:update"},
		{"master status", http.MethodPut, "/api/v1/admin/users/{id}/status", "admin:master:status"},
		{"master delete", http.MethodDelete, "/api/v1/admin/users/{id}/delete", "admin:master:delete"},
		{"master restore", http.MethodPut, "/api/v1/admin/users/{id}/restore", "admin:master:restore"},
		{"master permanent_delete", http.MethodDelete, "/api/v1/admin/users/{id}/permanent", "admin:master:permanent_delete"},
		{"master batch_status", http.MethodPut, "/api/v1/admin/users/batch/status", "admin:master:batch_status"},
		{"master batch_delete", http.MethodDelete, "/api/v1/admin/users/batch/delete", "admin:master:batch_delete"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := resolveRoute(r, c.method, c.path)
			got := derivePermissionCode(req)
			if got != c.want {
				t.Errorf("derivePermissionCode(%s %s) = %q, want %q", c.method, c.path, got, c.want)
			}
		})
	}
}

func TestDerivePermissionCode_AdminTenant(t *testing.T) {
	r := buildTestRouter()

	cases := []testCase{
		{"tenant list", http.MethodGet, "/api/v1/admin/tenants", "admin:tenant:list"},
		{"tenant create", http.MethodPost, "/api/v1/admin/tenants", "admin:tenant:create"},
		{"tenant list_deleted", http.MethodGet, "/api/v1/admin/tenants/deleted", "admin:tenant:list_deleted"},
		{"tenant read", http.MethodGet, "/api/v1/admin/tenants/{id}/detail", "admin:tenant:read"},
		{"tenant update", http.MethodPut, "/api/v1/admin/tenants/{id}/update", "admin:tenant:update"},
		{"tenant delete", http.MethodDelete, "/api/v1/admin/tenants/{id}/delete", "admin:tenant:delete"},
		{"tenant status", http.MethodPut, "/api/v1/admin/tenants/{id}/status", "admin:tenant:status"},
		{"tenant restore", http.MethodPut, "/api/v1/admin/tenants/{id}/restore", "admin:tenant:restore"},
		{"tenant batch_status", http.MethodPut, "/api/v1/admin/tenants/batch/status", "admin:tenant:batch_status"},
		{"tenant batch_delete", http.MethodDelete, "/api/v1/admin/tenants/batch", "admin:tenant:batch_delete"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := resolveRoute(r, c.method, c.path)
			got := derivePermissionCode(req)
			if got != c.want {
				t.Errorf("derivePermissionCode(%s %s) = %q, want %q", c.method, c.path, got, c.want)
			}
		})
	}
}

func TestDerivePermissionCode_AdminTenantUser(t *testing.T) {
	r := buildTestRouter()

	cases := []testCase{
		{"tenant_user list", http.MethodGet, "/api/v1/admin/tenant-users/all", "admin:tenant_user:list"},
		{"tenant_user create", http.MethodPost, "/api/v1/admin/tenant-users", "admin:tenant_user:create"},
		{"tenant_user list_all", http.MethodGet, "/api/v1/admin/tenant-users/all", "admin:tenant_user:list"},
		{"tenant_user list_deleted_all", http.MethodGet, "/api/v1/admin/tenant-users/deleted/all", "admin:tenant_user:list_deleted"},
		{"tenant_user list by tenant", http.MethodGet, "/api/v1/admin/tenant-users/{tenantID}/list", "admin:tenant_user:list"},
		{"tenant_user list_deleted by tenant", http.MethodGet, "/api/v1/admin/tenant-users/{tenantID}/deleted", "admin:tenant_user:list_deleted"},
		{"tenant_user update", http.MethodPut, "/api/v1/admin/tenant-users/{tenantID}/{userID}/update", "admin:tenant_user:update"},
		{"tenant_user status", http.MethodPut, "/api/v1/admin/tenant-users/{tenantID}/{userID}/status", "admin:tenant_user:status"},
		{"tenant_user reset_password", http.MethodPut, "/api/v1/admin/tenant-users/{tenantID}/{userID}/reset-password", "admin:tenant_user:reset_password"},
		{"tenant_user restore", http.MethodPut, "/api/v1/admin/tenant-users/{tenantID}/{userID}/restore", "admin:tenant_user:restore"},
		{"tenant_user delete", http.MethodDelete, "/api/v1/admin/tenant-users/{tenantID}/{userID}/delete", "admin:tenant_user:delete"},
		{"tenant_user permanent_delete", http.MethodDelete, "/api/v1/admin/tenant-users/{tenantID}/{userID}/permanent", "admin:tenant_user:permanent_delete"},
		{"tenant_user batch_status", http.MethodPut, "/api/v1/admin/tenant-users/{tenantID}/batch/status", "admin:tenant_user:batch_status"},
		{"tenant_user batch_delete", http.MethodDelete, "/api/v1/admin/tenant-users/{tenantID}/batch/delete", "admin:tenant_user:batch_delete"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := resolveRoute(r, c.method, c.path)
			got := derivePermissionCode(req)
			if got != c.want {
				t.Errorf("derivePermissionCode(%s %s) = %q, want %q", c.method, c.path, got, c.want)
			}
		})
	}
}

func TestDerivePermissionCode_AdminPlan(t *testing.T) {
	r := buildTestRouter()

	cases := []testCase{
		{"plan list", http.MethodGet, "/api/v1/admin/plans", "admin:plan:list"},
		{"plan create", http.MethodPost, "/api/v1/admin/plans", "admin:plan:create"},
		{"plan select", http.MethodGet, "/api/v1/admin/plans/select", "admin:plan:list"},
		{"plan update", http.MethodPut, "/api/v1/admin/plans/{id}/update", "admin:plan:update"},
		{"plan delete", http.MethodDelete, "/api/v1/admin/plans/{id}/delete", "admin:plan:delete"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := resolveRoute(r, c.method, c.path)
			got := derivePermissionCode(req)
			if got != c.want {
				t.Errorf("derivePermissionCode(%s %s) = %q, want %q", c.method, c.path, got, c.want)
			}
		})
	}
}