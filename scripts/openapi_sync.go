//go:build ignore

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"meteorx/internal/middleware"
	auth "meteorx/internal/modules/auth"
	authHandler "meteorx/internal/modules/auth/handler"
	audit "meteorx/internal/modules/audit"
	auditHandler "meteorx/internal/modules/audit/handler"
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

func main() {
	applyFlag := flag.Bool("apply", false, "自动修复 OpenAPI 文件（添加缺失路由 / 删除多余路由）")
	flag.Parse()

	r := buildRouter()

	type routeEntry struct {
		method  string
		pattern string
	}
	var entries []routeEntry
	validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true}

	chi.Walk(r, func(method, pattern string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if method == "" {
			return nil
		}
		mUpper := strings.ToUpper(method)
		if !validMethods[mUpper] {
			return nil
		}
		entries = append(entries, routeEntry{method: mUpper, pattern: pattern})
		return nil
	})

	codeSet := make(map[string]routeEntry)
	for _, e := range entries {
		key := e.method + " " + normalizePattern(e.pattern)
		codeSet[key] = routeEntry{method: e.method, pattern: normalizePattern(e.pattern)}
	}

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

	for path, methodsObj := range pathsObj {
		methods, _ := methodsObj.(map[string]interface{})
		for m := range methods {
			mUpper := strings.ToUpper(m)
			key := mUpper + " " + normalizePattern(path)
			openapiSet[key] = true
		}
	}

	fmt.Println("======== 代码路由 vs OpenAPI 对比 ========\n")

	var missingKeys []string
	var extraKeys []string

	for key := range codeSet {
		if !openapiSet[key] {
			missingKeys = append(missingKeys, key)
		}
	}
	for key := range openapiSet {
		if !codeSet[key] {
			extraKeys = append(extraKeys, key)
		}
	}

	sort.Strings(missingKeys)
	sort.Strings(extraKeys)

	fmt.Printf("代码路由总数: %d\n", len(codeSet))
	fmt.Printf("OpenAPI 路由总数: %d\n\n", len(openapiSet))

	if len(missingKeys) > 0 {
		fmt.Printf("⚠️  OpenAPI 缺失 %d 个路由：\n", len(missingKeys))
		for _, k := range missingKeys {
			fmt.Printf("  + %s\n", k)
		}
		fmt.Println()
	} else {
		fmt.Println("✅ OpenAPI 无缺失路由")
	}

	if len(extraKeys) > 0 {
		fmt.Printf("⚠️  OpenAPI 多出 %d 个路由：\n", len(extraKeys))
		for _, k := range extraKeys {
			fmt.Printf("  - %s\n", k)
		}
		fmt.Println()
	} else {
		fmt.Println("✅ OpenAPI 无多余路由")
	}

	if len(missingKeys) == 0 && len(extraKeys) == 0 {
		fmt.Println("\n🎉 完美匹配！")
		return
	}

	if *applyFlag {
		fmt.Println("\n===== 应用修复 =====")

		for _, key := range missingKeys {
			e := codeSet[key]
			addMissingRoute(pathsObj, e.method, e.pattern)
			fmt.Printf("+ 添加 %s %s\n", e.method, e.pattern)
		}

		removedCount := 0
		for _, key := range extraKeys {
			parts := strings.SplitN(key, " ", 2)
			method := strings.ToLower(parts[0])
			path := parts[1]
			if methodsObj, ok := pathsObj[path].(map[string]interface{}); ok {
				delete(methodsObj, method)
				if len(methodsObj) == 0 {
					delete(pathsObj, path)
				}
				removedCount++
				fmt.Printf("- 删除 %s %s\n", strings.ToUpper(method), path)
			}
		}

		out, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "序列化失败: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile("docs/apifox/MeteorX-backend.openapi.json", out, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "写入失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n✅ 已修复：+%d / -%d\n", len(missingKeys), removedCount)
	} else {
		fmt.Println("\n💡 加 -apply 参数可自动修复")
	}
}

func addMissingRoute(paths map[string]interface{}, method, pattern string) {
	pathObj, exists := paths[pattern]
	if !exists {
		pathObj = map[string]interface{}{}
		paths[pattern] = pathObj
	}
	methodsMap := pathObj.(map[string]interface{})
	methodLower := strings.ToLower(method)

	tags := inferTags(pattern)
	perm := middleware.DerivePermissionCode(method, pattern)

	op := map[string]interface{}{
		"summary":     inferSummary(method, pattern),
		"tags":        tags,
		"description": fmt.Sprintf("权限码: `%s`", perm),
		"responses": map[string]interface{}{
			"200": map[string]interface{}{
				"description": "成功",
			},
		},
	}

	methodsMap[methodLower] = op
}

func inferTags(pattern string) []string {
	if strings.HasPrefix(pattern, "/health") || strings.HasPrefix(pattern, "/metrics") {
		return []string{"System"}
	}
	if strings.HasPrefix(pattern, "/api/v1/auth") || strings.HasPrefix(pattern, "/api/v1/oauth") {
		return []string{"Auth"}
	}
	if strings.HasPrefix(pattern, "/api/v1/users") {
		return []string{"User"}
	}
	if strings.HasPrefix(pattern, "/api/v1/tenants") || strings.HasPrefix(pattern, "/api/v1/tenant") {
		return []string{"Tenant"}
	}
	if strings.HasPrefix(pattern, "/api/v1/rbac") {
		return []string{"RBAC"}
	}
	if strings.HasPrefix(pattern, "/api/v1/audit") {
		return []string{"Audit"}
	}
	if strings.HasPrefix(pattern, "/api/v1/wiki") {
		return []string{"Wiki"}
	}
	if strings.HasPrefix(pattern, "/api/v1/files") {
		return []string{"File"}
	}
	if strings.HasPrefix(pattern, "/api/v1/admin") {
		return []string{"Admin"}
	}
	if strings.HasPrefix(pattern, "/api/v1/plan") {
		return []string{"Plan"}
	}
	if strings.HasPrefix(pattern, "/api/v1/announcements") || strings.HasPrefix(pattern, "/api/v1/notification") {
		return []string{"Notification"}
	}
	return []string{"API"}
}

func inferSummary(method, pattern string) string {
	methodCN := map[string]string{
		"GET": "获取", "POST": "创建", "PUT": "更新", "DELETE": "删除", "PATCH": "部分更新",
	}
	m := methodCN[method]
	if m == "" {
		m = method
	}
	parts := strings.Split(strings.Trim(pattern, "/"), "/")
	if len(parts) == 0 {
		return m
	}
	last := parts[len(parts)-1]
	last = strings.ReplaceAll(last, "{id}", "资源")
	return m + last
}

func normalizePattern(p string) string {
	p = strings.ReplaceAll(p, "{param}", "{id}")
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = p[:len(p)-1]
	}
	return p
}

func buildRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/health", noop())
	r.Get("/health/ready", noop())
	r.Get("/metrics", noop())

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