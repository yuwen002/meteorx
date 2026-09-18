package bootstrap

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"

	"meteorx/internal/middleware"
	auditHandler "meteorx/internal/modules/audit/handler"
	authHandler "meteorx/internal/modules/auth/handler"
	dashboardHandler "meteorx/internal/modules/dashboard/handler"
	fileHandler "meteorx/internal/modules/file/handler"
	notificationHandler "meteorx/internal/modules/notification/handler"
	oauthHandler "meteorx/internal/modules/oauth/handler"
	planHandler "meteorx/internal/modules/plan/handler"
	rbacHandler "meteorx/internal/modules/rbac/handler"
	tenantHandler "meteorx/internal/modules/tenant/handler"
	userHandler "meteorx/internal/modules/user/handler"
	wikiHandler "meteorx/internal/modules/wiki/handler"

	"github.com/go-chi/chi/v5"
)

// ========================= P1: 路由注册 + 权限码推导校验 =========================

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

// buildTestRouter 构建与 InitRouter 相同结构的路由树，
// 使用 zero-value handler 实例（Walk 只遍历 pattern，不调用 handler）
func buildTestRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/health", noopHandler())
	r.Get("/health/ready", noopHandler())
	r.Handle("/metrics", noopHandler())
	r.Handle("/uploads/*", noopHandler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			authHandler.RegisterRoutes(r, &authHandler.AuthHandler{})
			tenantHandler.RegisterPublicRoutes(r, &tenantHandler.TenantHandler{})
			oauthHandler.RegisterRoutes(r, &oauthHandler.OAuthHandler{})
			wikiHandler.RegisterPublicShareRoute(r, nil, nil, nil)
		})

		r.Group(func(r chi.Router) {
			r.Get("/ws", noopHandler())

			tenantHandler.RegisterPrivateRoutes(r, &tenantHandler.TenantHandler{})
			tenantHandler.RegisterTenantSettingsRoutes(r, &tenantHandler.TenantSettingsHandler{})

			userHandler.RegisterRoutes(r, &userHandler.UserHandler{}, nil)
			userHandler.RegisterProfileRoutes(r, &userHandler.UserHandler{})

			planHandler.RegisterPrivateRoutes(r, &planHandler.PlanHandler{})

			fileHandler.RegisterRoutes(r, nil, nil)

			wikiHandler.RegisterRoutes(r, &wikiHandler.WikiHandler{}, nil, nil)

			notificationHandler.RegisterTenantRoutes(r, &notificationHandler.AnnouncementHandler{})

			r.Group(func(r chi.Router) {
				tenantHandler.RegisterAdminRoutes(r, &tenantHandler.TenantHandler{}, nil)
				userHandler.RegisterAdminRoutes(r, &userHandler.UserHandler{}, nil)
				rbacHandler.RegisterRoutes(r, &rbacHandler.RBACHandler{}, nil)
				auditHandler.RegisterRoutes(r, &auditHandler.AuditHandler{}, &auditHandler.AlertHandler{}, &auditHandler.SessionHandler{}, nil, nil)
				planHandler.RegisterAdminRoutes(r, &planHandler.PlanHandler{}, nil)
				dashboardHandler.RegisterRoutes(r, &dashboardHandler.DashboardHandler{}, nil)
				notificationHandler.RegisterRoutes(r, &notificationHandler.AnnouncementHandler{}, nil)
			})
		})
	})

	return r
}

func noopHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}