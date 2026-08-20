package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// buildTestRouter 构建与实际路由一致的 chi.Mux，用于权限码推导测试
func buildTestRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		// 租户侧用户路由
		r.Route("/users", func(r chi.Router) {
			r.Get("/", noop)
			r.Post("/", noop)
			r.Get("/deleted", noop)
			r.Get("/{id}/detail", noop)
			r.Put("/{id}/update", noop)
			r.Put("/{id}/reset-password", noop)
			r.Delete("/{id}/delete", noop)
			r.Put("/{id}/restore", noop)
			r.Delete("/{id}/permanent", noop)
			r.Put("/batch/status", noop)
			r.Delete("/batch/delete", noop)
		})

		// RBAC 路由
		r.Route("/rbac", func(r chi.Router) {
			// 角色管理
			r.Route("/roles", func(r chi.Router) {
				r.Get("/", noop)
				r.Post("/", noop)
				r.Get("/select", noop)
				r.Get("/system-admin", noop)
				r.Get("/deleted", noop)
				r.Put("/batch/status", noop)
				r.Delete("/batch/delete", noop)
				r.Put("/batch/permissions", noop)
				r.Delete("/batch/permissions", noop)
				r.Put("/{id}/restore", noop)
				r.Get("/{id}/detail", noop)
				r.Put("/{id}/update", noop)
				r.Put("/{id}/status", noop)
				r.Delete("/{id}/delete", noop)
				r.Put("/{id}/permissions", noop)
				r.Get("/{id}/permissions", noop)
				r.Delete("/{id}/permissions", noop)
				r.Delete("/{id}/permissions/batch", noop)
			})
			// 权限管理
			r.Route("/permissions", func(r chi.Router) {
				r.Get("/", noop)
				r.Post("/", noop)
				r.Put("/batch/status", noop)
				r.Delete("/batch/delete", noop)
				r.Get("/{id}/detail", noop)
				r.Put("/{id}/update", noop)
				r.Put("/{id}/status", noop)
				r.Delete("/{id}/delete", noop)
			})
			// 角色权限关系
			r.Route("/role-permissions", func(r chi.Router) {
				r.Get("/", noop)
			})
			// 用户角色管理
			r.Route("/user-roles", func(r chi.Router) {
				r.Get("/", noop)
				r.Post("/batch/assign", noop)
				r.Route("/{user_id}/roles", func(r chi.Router) {
					r.Post("/", noop)
					r.Get("/", noop)
					r.Delete("/", noop)
					r.Delete("/{role_id}", noop)
				})
				r.Route("/roles/{role_id}/users", func(r chi.Router) {
					r.Get("/", noop)
				})
			})
		})

		// Admin 路由（平台后台）
		r.Route("/admin", func(r chi.Router) {
			// 系统管理员
			r.Route("/users", func(r chi.Router) {
				r.Get("/", noop)
				r.Post("/", noop)
				r.Get("/deleted", noop)
				r.Get("/{id}/detail", noop)
				r.Put("/{id}/update", noop)
				r.Put("/{id}/status", noop)
				r.Delete("/{id}/delete", noop)
				r.Put("/{id}/restore", noop)
				r.Delete("/{id}/permanent", noop)
				r.Put("/batch/status", noop)
				r.Delete("/batch/delete", noop)
			})

			// 租户管理
			r.Route("/tenants", func(r chi.Router) {
				r.Get("/", noop)
				r.Post("/", noop)
				r.Get("/deleted", noop)
				r.Get("/{id}/detail", noop)
				r.Put("/{id}/update", noop)
				r.Delete("/{id}/delete", noop)
				r.Put("/{id}/status", noop)
				r.Put("/{id}/restore", noop)
				r.Put("/batch/status", noop)
				r.Delete("/batch", noop)
				r.Put("/{id}/plan", noop)
			})

			// 跨租户用户管理
			r.Route("/tenant-users", func(r chi.Router) {
				r.Post("/", noop)
				r.Get("/all", noop)
				r.Get("/deleted/all", noop)
				r.Get("/{tenantID}/list", noop)
				r.Get("/{tenantID}/deleted", noop)
				r.Put("/{tenantID}/{userID}/update", noop)
				r.Put("/{tenantID}/{userID}/status", noop)
				r.Put("/{tenantID}/{userID}/reset-password", noop)
				r.Put("/{tenantID}/{userID}/restore", noop)
				r.Delete("/{tenantID}/{userID}/delete", noop)
				r.Delete("/{tenantID}/{userID}/permanent", noop)
				r.Put("/{tenantID}/batch/status", noop)
				r.Delete("/{tenantID}/batch/delete", noop)
			})

			// 套餐管理
			r.Route("/plans", func(r chi.Router) {
				r.Get("/", noop)
				r.Post("/", noop)
				r.Get("/select", noop)
				r.Put("/{id}/update", noop)
				r.Delete("/{id}/delete", noop)
			})

			// 套餐分配（为租户分配套餐）
			r.Route("/tenants-plan", func(r chi.Router) {
				r.Get("/{id}", noop)
				r.Put("/{id}", noop)
			})
		})

		// 审计日志路由
		r.Route("/audit", func(r chi.Router) {
			r.Route("/logs", func(r chi.Router) {
				r.Get("/", noop)
				r.Get("/export", noop)
				r.Post("/", noop)
				r.Get("/{id}", noop)
				r.Delete("/cleanup", noop)
			})
		})
	})
	return r
}

func noop(_ http.ResponseWriter, _ *http.Request) {}

// testCase 表示一个权限码推导测试用例
type testCase struct {
	name   string
	method string
	path   string
	want   string
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
			got := doRequest(r, c.method, c.path)
			if got != c.want {
				t.Errorf("derivePermissionCode(%s %s) = %q, want %q", c.method, c.path, got, c.want)
			}
		})
	}
}

func TestDerivePermissionCode_RBAC(t *testing.T) {
	r := buildTestRouter()

	cases := []testCase{
		// 角色管理
		{"role list", http.MethodGet, "/api/v1/rbac/roles", "rbac:role:list"},
		{"role create", http.MethodPost, "/api/v1/rbac/roles", "rbac:role:create"},
		{"role list_select", http.MethodGet, "/api/v1/rbac/roles/select", "rbac:role:list_select"},
		{"role list_system_admin", http.MethodGet, "/api/v1/rbac/roles/system-admin", "rbac:role:list_system_admin"},
		{"role list_deleted", http.MethodGet, "/api/v1/rbac/roles/deleted", "rbac:role:list_deleted"},
		{"role batch_status", http.MethodPut, "/api/v1/rbac/roles/batch/status", "rbac:role:batch_status"},
		{"role batch_delete", http.MethodDelete, "/api/v1/rbac/roles/batch/delete", "rbac:role:batch_delete"},
		{"role batch_bind", http.MethodPut, "/api/v1/rbac/roles/batch/permissions", "rbac:role:batch_bind"},
		{"role batch_unbind", http.MethodDelete, "/api/v1/rbac/roles/batch/permissions", "rbac:role:batch_unbind"},
		{"role restore", http.MethodPut, "/api/v1/rbac/roles/{id}/restore", "rbac:role:restore"},
		{"role read", http.MethodGet, "/api/v1/rbac/roles/{id}/detail", "rbac:role:read"},
		{"role update", http.MethodPut, "/api/v1/rbac/roles/{id}/update", "rbac:role:update"},
		{"role status", http.MethodPut, "/api/v1/rbac/roles/{id}/status", "rbac:role:status"},
		{"role delete", http.MethodDelete, "/api/v1/rbac/roles/{id}/delete", "rbac:role:delete"},
		{"role bind_perm", http.MethodPut, "/api/v1/rbac/roles/{id}/permissions", "rbac:role:bind_perm"},
		{"role get_perms", http.MethodGet, "/api/v1/rbac/roles/{id}/permissions", "rbac:role:get_perms"},
		{"role unbind_perm", http.MethodDelete, "/api/v1/rbac/roles/{id}/permissions", "rbac:role:unbind_perm"},
		{"role batch_unbind_perm", http.MethodDelete, "/api/v1/rbac/roles/{id}/permissions/batch", "rbac:role:batch_unbind_perm"},

		// 权限管理
		{"perm list", http.MethodGet, "/api/v1/rbac/permissions", "rbac:perm:list"},
		{"perm create", http.MethodPost, "/api/v1/rbac/permissions", "rbac:perm:create"},
		{"perm batch_status", http.MethodPut, "/api/v1/rbac/permissions/batch/status", "rbac:perm:batch_status"},
		{"perm batch_delete", http.MethodDelete, "/api/v1/rbac/permissions/batch/delete", "rbac:perm:batch_delete"},
		{"perm read", http.MethodGet, "/api/v1/rbac/permissions/{id}/detail", "rbac:perm:read"},
		{"perm update", http.MethodPut, "/api/v1/rbac/permissions/{id}/update", "rbac:perm:update"},
		{"perm status", http.MethodPut, "/api/v1/rbac/permissions/{id}/status", "rbac:perm:status"},
		{"perm delete", http.MethodDelete, "/api/v1/rbac/permissions/{id}/delete", "rbac:perm:delete"},

		// 角色权限关系
		{"role_perm list", http.MethodGet, "/api/v1/rbac/role-permissions", "rbac:role_perm:list"},

		// 用户角色管理
		{"user_role list", http.MethodGet, "/api/v1/rbac/user-roles", "rbac:user_role:list"},
		{"user_role batch_assign", http.MethodPost, "/api/v1/rbac/user-roles/batch/assign", "rbac:user_role:batch_assign"},
		{"user_role assign", http.MethodPost, "/api/v1/rbac/user-roles/{user_id}/roles", "rbac:user_role:assign"},
		{"user_role get_roles", http.MethodGet, "/api/v1/rbac/user-roles/{user_id}/roles", "rbac:user_role:get_roles"},
		{"user_role remove_all", http.MethodDelete, "/api/v1/rbac/user-roles/{user_id}/roles", "rbac:user_role:remove_all"},
		{"user_role remove_one", http.MethodDelete, "/api/v1/rbac/user-roles/{user_id}/roles/{role_id}", "rbac:user_role:remove_one"},
		{"user_role get_users", http.MethodGet, "/api/v1/rbac/user-roles/roles/{role_id}/users", "rbac:user_role:get_users"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := doRequest(r, c.method, c.path)
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
			got := doRequest(r, c.method, c.path)
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
		{"tenant plan assign", http.MethodPut, "/api/v1/admin/tenants/{id}/plan", "admin:plan:assign"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := doRequest(r, c.method, c.path)
			if got != c.want {
				t.Errorf("derivePermissionCode(%s %s) = %q, want %q", c.method, c.path, got, c.want)
			}
		})
	}
}

func TestDerivePermissionCode_AdminTenantUser(t *testing.T) {
	r := buildTestRouter()

	cases := []testCase{
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
			got := doRequest(r, c.method, c.path)
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
		{"plan select", http.MethodGet, "/api/v1/admin/plans/select", "admin:plan:select"},
		{"plan update", http.MethodPut, "/api/v1/admin/plans/{id}/update", "admin:plan:update"},
		{"plan delete", http.MethodDelete, "/api/v1/admin/plans/{id}/delete", "admin:plan:delete"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := doRequest(r, c.method, c.path)
			if got != c.want {
				t.Errorf("derivePermissionCode(%s %s) = %q, want %q", c.method, c.path, got, c.want)
			}
		})
	}
}

func TestDerivePermissionCode_AdminTenantPlan(t *testing.T) {
	r := buildTestRouter()

	cases := []testCase{
		{"tenant plan read", http.MethodGet, "/api/v1/admin/tenants-plan/{id}", "admin:plan:read"},
		{"tenant plan assign", http.MethodPut, "/api/v1/admin/tenants-plan/{id}", "admin:plan:assign"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := doRequest(r, c.method, c.path)
			if got != c.want {
				t.Errorf("derivePermissionCode(%s %s) = %q, want %q", c.method, c.path, got, c.want)
			}
		})
	}
}

func TestDerivePermissionCode_AuditLog(t *testing.T) {
	r := buildTestRouter()

	cases := []testCase{
		{"log list", http.MethodGet, "/api/v1/audit/logs", "audit:log:list"},
		{"log export", http.MethodGet, "/api/v1/audit/logs/export", "audit:log:export"},
		{"log create", http.MethodPost, "/api/v1/audit/logs", "audit:log:create"},
		{"log read", http.MethodGet, "/api/v1/audit/logs/{id}", "audit:log:read"},
		{"log cleanup", http.MethodDelete, "/api/v1/audit/logs/cleanup", "audit:log:cleanup"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := doRequest(r, c.method, c.path)
			if got != c.want {
				t.Errorf("derivePermissionCode(%s %s) = %q, want %q", c.method, c.path, got, c.want)
			}
		})
	}
}

// doRequest 执行一次请求，返回中间件捕获的权限码
// 中间件将 derivePermissionCode 结果写入响应头 X-Derived-Perm
func doRequest(r *chi.Mux, method, path string) string {
	// 在顶层路由挂一个写 header 的中间件
	// 重新包装 router：通过拦截 ServeHTTP 调用
	// 更简单的方式：用一个闭包包裹每个 handler 写 header
	// 这里选择在 capturePermMiddleware 中写 header

	// 但 buildTestRouter 中的 handler 是 noop，所以我们不能在 handler 中写
	// 这里改为：在 capturePermMiddleware 中直接写响应头
	// 我们需要修改 capturePermMiddleware 的实现
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()

	// 用 chi.Match 来获取路由 pattern
	m := chi.NewMux()
	m.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			// handler 执行完毕后，RouteContext 可用
			code := derivePermissionCode(r)
			w.Header().Set("X-Derived-Perm", code)
		})
	})

	// 挂载原 router
	m.Handle("/api/v1/*", r)
	m.ServeHTTP(rec, req)

	return rec.Header().Get("X-Derived-Perm")
}