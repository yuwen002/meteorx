package bootstrap

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"

	"meteorx/internal/middleware"
	audit "meteorx/internal/modules/audit"
	auditHandler "meteorx/internal/modules/audit/handler"
	auth "meteorx/internal/modules/auth"
	authHandler "meteorx/internal/modules/auth/handler"
	dashboard "meteorx/internal/modules/dashboard"
	dashboardHandler "meteorx/internal/modules/dashboard/handler"
	notification "meteorx/internal/modules/notification"
	notificationHandler "meteorx/internal/modules/notification/handler"
	oauth "meteorx/internal/modules/oauth"
	oauthHandler "meteorx/internal/modules/oauth/handler"
	plan "meteorx/internal/modules/plan"
	planHandler "meteorx/internal/modules/plan/handler"
	rbac "meteorx/internal/modules/rbac"
	rbacHandler "meteorx/internal/modules/rbac/handler"
	tenant "meteorx/internal/modules/tenant"
	tenantHandler "meteorx/internal/modules/tenant/handler"
	user "meteorx/internal/modules/user"
	userHandler "meteorx/internal/modules/user/handler"

	"github.com/go-chi/chi/v5"
)

func TestAllRoutes_DerivePermissionCode(t *testing.T) {
	r := buildTestRouter()

	type routeEntry struct {
		method  string
		pattern string
	}
	var entries []routeEntry

	err := chi.Walk(r, func(method, pattern string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if method == "" {
			return nil
		}
		entries = append(entries, routeEntry{method: method, pattern: pattern})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].pattern == entries[j].pattern {
			return entries[i].method < entries[j].method
		}
		return entries[i].pattern < entries[j].pattern
	})

	apiCount := 0
	rootCount := 0
	var noPermRoutes []string
	var permResults []string

	for _, e := range entries {
		if strings.HasPrefix(e.pattern, "/api/v1/") {
			apiCount++
			code := middleware.DerivePermissionCode(e.method, e.pattern)
			if code == "" {
				noPermRoutes = append(noPermRoutes, fmt.Sprintf("%s %s", e.method, e.pattern))
			}
			permResults = append(permResults, fmt.Sprintf("%-6s %-55s → %s", e.method, e.pattern, code))
		} else {
			rootCount++
		}
	}

	fmt.Fprintf(os.Stderr, "\n===== 路由总数统计 =====\n")
	fmt.Fprintf(os.Stderr, "API 路由 (/api/v1/*): %d\n", apiCount)
	fmt.Fprintf(os.Stderr, "根路由: %d\n", rootCount)
	fmt.Fprintf(os.Stderr, "总计: %d\n\n", apiCount+rootCount)

	fmt.Fprintf(os.Stderr, "===== 权限码推导结果 =====\n")
	for _, line := range permResults {
		fmt.Fprintf(os.Stderr, "%s\n", line)
	}

	if len(noPermRoutes) > 0 {
		t.Errorf("有 %d 个 /api/v1 路由无法自动推导权限码：\n", len(noPermRoutes))
		for _, r := range noPermRoutes {
			t.Errorf("  %s", r)
		}
	}
}

func buildTestRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/health", noopHandler())
	r.Get("/health/ready", noopHandler())
	r.Handle("/metrics", noopHandler())
	r.Handle("/uploads/*", noopHandler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			auth.RegisterRoutes(r, &authHandler.AuthHandler{})
			tenant.RegisterPublicRoutes(r, &tenantHandler.TenantHandler{})
			oauth.RegisterRoutes(r, &oauthHandler.OAuthHandler{})
			registerWikiPublicShare(r)
		})

		r.Group(func(r chi.Router) {
			r.Get("/ws", noopHandler())

			tenant.RegisterPrivateRoutes(r, &tenantHandler.TenantHandler{})
			tenant.RegisterTenantSettingsRoutes(r, &tenantHandler.TenantSettingsHandler{})

			user.RegisterRoutes(r, &userHandler.UserHandler{}, nil)
			user.RegisterProfileRoutes(r, &userHandler.UserHandler{})

			plan.RegisterPrivateRoutes(r, &planHandler.PlanHandler{})

			registerFileRoutes(r)

			registerWikiRoutes(r)

			notification.RegisterTenantRoutes(r, &notificationHandler.AnnouncementHandler{})

			r.Group(func(r chi.Router) {
				tenant.RegisterAdminRoutes(r, &tenantHandler.TenantHandler{}, nil)
				user.RegisterAdminRoutes(r, &userHandler.UserHandler{}, nil)
				rbac.RegisterRoutes(r, &rbacHandler.RBACHandler{}, nil)
				audit.RegisterRoutes(r, &auditHandler.AuditHandler{}, &auditHandler.AlertHandler{}, &auditHandler.SessionHandler{}, nil, nil)
				plan.RegisterAdminRoutes(r, &planHandler.PlanHandler{}, nil)
				dashboard.RegisterRoutes(r, &dashboardHandler.DashboardHandler{}, nil)
				notification.RegisterRoutes(r, &notificationHandler.AnnouncementHandler{}, nil)
			})
		})
	})

	return r
}

// registerFileRoutes 手动注册 file 模块路由（因 file.RegisterRoutes 依赖 DB+cfg）
func registerFileRoutes(r chi.Router) {
	r.Route("/files", func(r chi.Router) {
		r.Post("/upload", noopHandler())
		r.Get("/", noopHandler())
		r.Get("/my", noopHandler())
		r.Get("/{id}", noopHandler())
		r.Get("/{id}/download", noopHandler())
		r.Put("/{id}", noopHandler())
		r.Delete("/{id}", noopHandler())
		r.Post("/batch/delete", noopHandler())
		r.Get("/deleted", noopHandler())
		r.Put("/{id}/restore", noopHandler())
		r.Delete("/{id}/permanent", noopHandler())
	})
}

// registerWikiRoutes 手动注册 wiki 模块路由（因 wiki.InitModule 依赖 DB+cfg）
func registerWikiRoutes(r chi.Router) {
	r.Route("/wiki", func(r chi.Router) {
		r.Get("/stats", noopHandler())
		r.Get("/search", noopHandler())
		r.Get("/trash", noopHandler())
		r.Post("/trash/{id}/restore", noopHandler())
		r.Delete("/trash/{id}", noopHandler())

		r.Route("/spaces", func(r chi.Router) {
			r.Get("/", noopHandler())
			r.Post("/", noopHandler())
			r.Get("/{id}", noopHandler())
			r.Put("/{id}", noopHandler())
			r.Delete("/{id}", noopHandler())

			r.Route("/{spaceId}/nodes", func(r chi.Router) {
				r.Get("/tree", noopHandler())
				r.Post("/", noopHandler())
				r.Get("/{id}", noopHandler())
				r.Put("/{id}", noopHandler())
				r.Delete("/{id}", noopHandler())
				r.Post("/{id}/move", noopHandler())
				r.Put("/{id}/sort", noopHandler())
				r.Get("/{id}/permissions", noopHandler())
				r.Post("/{id}/permissions", noopHandler())
				r.Delete("/{id}/permissions/{userId}/{permission}", noopHandler())
			})

			r.Route("/{spaceId}/members", func(r chi.Router) {
				r.Get("/", noopHandler())
				r.Post("/", noopHandler())
				r.Delete("/{userId}", noopHandler())
			})

			r.Post("/tags", noopHandler())
			r.Get("/tags", noopHandler())
			r.Delete("/tags/{id}", noopHandler())
			r.Post("/nodes/batch", noopHandler())
			r.Post("/templates", noopHandler())
			r.Get("/templates", noopHandler())
			r.Get("/templates/{id}", noopHandler())
			r.Put("/templates/{id}", noopHandler())
			r.Delete("/templates/{id}", noopHandler())
			r.Get("/notifications", noopHandler())
			r.Put("/notifications/read-all", noopHandler())
			r.Get("/notifications/unread-count", noopHandler())
			r.Put("/notifications/{id}/read", noopHandler())
			r.Get("/subscriptions", noopHandler())
		})

		r.Route("/documents", func(r chi.Router) {
			r.Post("/preview", noopHandler())
			r.Post("/nodes/{nodeId}", noopHandler())
			r.Get("/nodes/{nodeId}", noopHandler())
			r.Put("/{id}", noopHandler())
			r.Delete("/{id}", noopHandler())
			r.Get("/{documentId}/revisions", noopHandler())
			r.Get("/{documentId}/revisions/{version}", noopHandler())
			r.Post("/{documentId}/revisions/{version}/restore", noopHandler())
			r.Post("/attachments", noopHandler())
			r.Get("/{documentId}/attachments", noopHandler())
			r.Delete("/attachments/{id}", noopHandler())
			r.Post("/{id}/tags/{tagId}", noopHandler())
			r.Delete("/{id}/tags/{tagId}", noopHandler())
			r.Get("/{id}/tags", noopHandler())
			r.Post("/{id}/comments", noopHandler())
			r.Get("/{id}/comments", noopHandler())
			r.Put("/comments/{id}", noopHandler())
			r.Delete("/comments/{id}", noopHandler())
			r.Post("/{id}/share", noopHandler())
			r.Get("/{id}/shares", noopHandler())
			r.Delete("/shares/{id}", noopHandler())
			r.Get("/{id}/stats", noopHandler())
			r.Get("/{id}/access-logs", noopHandler())
			r.Post("/{id}/subscribe", noopHandler())
			r.Delete("/{id}/subscribe", noopHandler())
			r.Post("/{id}/edit-lock", noopHandler())
			r.Delete("/{id}/edit-lock", noopHandler())
			r.Put("/{id}/edit-lock", noopHandler())
			r.Get("/{id}/edit-lock", noopHandler())
			r.Post("/{id}/export", noopHandler())
			r.Post("/{id}/import", noopHandler())
			r.Get("/{id}/revisions/compare", noopHandler())
		})
	})
}

// registerWikiPublicShare 手动注册 wiki 公开分享路由
func registerWikiPublicShare(r chi.Router) {
	r.Get("/wiki/share/{token}", noopHandler())
}

func noopHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}
