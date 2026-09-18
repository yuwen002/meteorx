//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	auth "meteorx/internal/modules/auth"
	audit "meteorx/internal/modules/audit"
	dashboard "meteorx/internal/modules/dashboard"
	notification "meteorx/internal/modules/notification"
	oauth "meteorx/internal/modules/oauth"
	plan "meteorx/internal/modules/plan"
	rbac "meteorx/internal/modules/rbac"
	tenant "meteorx/internal/modules/tenant"
	user "meteorx/internal/modules/user"
	authHandler "meteorx/internal/modules/auth/handler"
	auditHandler "meteorx/internal/modules/audit/handler"
	dashboardHandler "meteorx/internal/modules/dashboard/handler"
	notificationHandler "meteorx/internal/modules/notification/handler"
	oauthHandler "meteorx/internal/modules/oauth/handler"
	planHandler "meteorx/internal/modules/plan/handler"
	rbacHandler "meteorx/internal/modules/rbac/handler"
	tenantHandler "meteorx/internal/modules/tenant/handler"
	userHandler "meteorx/internal/modules/user/handler"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := buildRouter()

	type routeEntry struct {
		method  string
		pattern string
	}
	var entries []routeEntry

	chi.Walk(r, func(method, pattern string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if method == "" {
			return nil
		}
		entries = append(entries, routeEntry{method: method, pattern: pattern})
		return nil
	})

	codeSet := make(map[string]bool)
	for _, e := range entries {
		key := strings.ToUpper(e.method) + " " + normalizePattern(e.pattern)
		codeSet[key] = true
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].pattern == entries[j].pattern {
			return entries[i].method < entries[j].method
		}
		return entries[i].pattern < entries[j].pattern
	})

	data, err := os.ReadFile("docs/apifox/MeteorX-backend.openapi.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取 OpenAPI 文件失败: %v\n", err)
		os.Exit(1)
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(data, &spec); err != nil {
		fmt.Fprintf(os.Stderr, "解析 OpenAPI JSON 失败: %v\n", err)
		os.Exit(1)
	}

	pathsObj, _ := spec["paths"].(map[string]interface{})
	openapiSet := make(map[string]bool)
	openapiPathMethods := make(map[string][]string)

	for path, methodsObj := range pathsObj {
		methods, _ := methodsObj.(map[string]interface{})
		for m := range methods {
			mUpper := strings.ToUpper(m)
			key := mUpper + " " + normalizePattern(path)
			openapiSet[key] = true
			openapiPathMethods[path] = append(openapiPathMethods[path], mUpper)
		}
	}

	fmt.Println("======== 代码路由 vs OpenAPI 对比 ========\n")

	var missingInOpenAPI []string
	var extraInOpenAPI []string

	for key := range codeSet {
		if !openapiSet[key] {
			missingInOpenAPI = append(missingInOpenAPI, key)
		}
	}
	for key := range openapiSet {
		if !codeSet[key] {
			extraInOpenAPI = append(extraInOpenAPI, key)
		}
	}

	sort.Strings(missingInOpenAPI)
	sort.Strings(extraInOpenAPI)

	fmt.Printf("代码路由总数: %d\n", len(codeSet))
	fmt.Printf("OpenAPI 路由总数: %d\n\n", len(openapiSet))

	if len(missingInOpenAPI) > 0 {
		fmt.Printf("⚠️  OpenAPI 缺失 %d 个路由（代码有，文档没有）：\n", len(missingInOpenAPI))
		for _, k := range missingInOpenAPI {
			fmt.Printf("  + %s\n", k)
		}
		fmt.Println()
	} else {
		fmt.Println("✅ OpenAPI 无缺失路由")
	}

	if len(extraInOpenAPI) > 0 {
		fmt.Printf("⚠️  OpenAPI 多出 %d 个路由（文档有，代码没有）：\n", len(extraInOpenAPI))
		for _, k := range extraInOpenAPI {
			fmt.Printf("  - %s\n", k)
		}
		fmt.Println()
	} else {
		fmt.Println("✅ OpenAPI 无多余路由")
	}

	if len(missingInOpenAPI) == 0 && len(extraInOpenAPI) == 0 {
		fmt.Println("🎉 完美匹配！")
	}
}

func normalizePattern(p string) string {
	return strings.ReplaceAll(p, "{param}", "{id}")
}

func buildRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/health", noop())
	r.Get("/health/ready", noop())
	r.Handle("/metrics", noop())
	r.Handle("/uploads/*", noop())

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			auth.RegisterRoutes(r, &authHandler.AuthHandler{})
			tenant.RegisterPublicRoutes(r, &tenantHandler.TenantHandler{})
			oauth.RegisterRoutes(r, &oauthHandler.OAuthHandler{})
			registerWikiPublicShare(r)
		})

		r.Group(func(r chi.Router) {
			r.Get("/ws", noop())

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

func registerFileRoutes(r chi.Router) {
	r.Route("/files", func(r chi.Router) {
		r.Post("/upload", noop())
		r.Get("/", noop())
		r.Get("/my", noop())
		r.Get("/{id}", noop())
		r.Get("/{id}/download", noop())
		r.Put("/{id}", noop())
		r.Delete("/{id}", noop())
		r.Post("/batch/delete", noop())
		r.Get("/deleted", noop())
		r.Put("/{id}/restore", noop())
		r.Delete("/{id}/permanent", noop())
	})
}

func registerWikiRoutes(r chi.Router) {
	r.Route("/wiki", func(r chi.Router) {
		r.Get("/stats", noop())
		r.Get("/search", noop())
		r.Get("/trash", noop())
		r.Post("/trash/{id}/restore", noop())
		r.Delete("/trash/{id}", noop())

		r.Route("/spaces", func(r chi.Router) {
			r.Get("/", noop())
			r.Post("/", noop())
			r.Get("/{id}", noop())
			r.Put("/{id}", noop())
			r.Delete("/{id}", noop())

			r.Route("/{spaceId}/nodes", func(r chi.Router) {
				r.Get("/tree", noop())
				r.Post("/", noop())
				r.Get("/{id}", noop())
				r.Put("/{id}", noop())
				r.Delete("/{id}", noop())
				r.Post("/{id}/move", noop())
				r.Put("/{id}/sort", noop())
				r.Get("/{id}/permissions", noop())
				r.Post("/{id}/permissions", noop())
				r.Delete("/{id}/permissions/{userId}/{permission}", noop())
			})

			r.Route("/{spaceId}/members", func(r chi.Router) {
				r.Get("/", noop())
				r.Post("/", noop())
				r.Delete("/{userId}", noop())
			})

			r.Post("/tags", noop())
			r.Get("/tags", noop())
			r.Delete("/tags/{id}", noop())
			r.Post("/nodes/batch", noop())
			r.Post("/templates", noop())
			r.Get("/templates", noop())
			r.Get("/templates/{id}", noop())
			r.Put("/templates/{id}", noop())
			r.Delete("/templates/{id}", noop())
			r.Get("/notifications", noop())
			r.Put("/notifications/read-all", noop())
			r.Get("/notifications/unread-count", noop())
			r.Put("/notifications/{id}/read", noop())
			r.Get("/subscriptions", noop())
		})

		r.Route("/documents", func(r chi.Router) {
			r.Post("/preview", noop())
			r.Post("/nodes/{nodeId}", noop())
			r.Get("/nodes/{nodeId}", noop())
			r.Put("/{id}", noop())
			r.Delete("/{id}", noop())
			r.Get("/{documentId}/revisions", noop())
			r.Get("/{documentId}/revisions/{version}", noop())
			r.Post("/{documentId}/revisions/{version}/restore", noop())
			r.Post("/attachments", noop())
			r.Get("/{documentId}/attachments", noop())
			r.Delete("/attachments/{id}", noop())
			r.Post("/{id}/tags/{tagId}", noop())
			r.Delete("/{id}/tags/{tagId}", noop())
			r.Get("/{id}/tags", noop())
			r.Post("/{id}/comments", noop())
			r.Get("/{id}/comments", noop())
			r.Put("/comments/{id}", noop())
			r.Delete("/comments/{id}", noop())
			r.Post("/{id}/share", noop())
			r.Get("/{id}/shares", noop())
			r.Delete("/shares/{id}", noop())
			r.Get("/{id}/stats", noop())
			r.Get("/{id}/access-logs", noop())
			r.Post("/{id}/subscribe", noop())
			r.Delete("/{id}/subscribe", noop())
			r.Post("/{id}/edit-lock", noop())
			r.Delete("/{id}/edit-lock", noop())
			r.Put("/{id}/edit-lock", noop())
			r.Get("/{id}/edit-lock", noop())
			r.Post("/{id}/export", noop())
			r.Post("/{id}/import", noop())
			r.Get("/{id}/revisions/compare", noop())
		})
	})
}

func registerWikiPublicShare(r chi.Router) {
	r.Get("/wiki/share/{token}", noop())
}

func noop() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}