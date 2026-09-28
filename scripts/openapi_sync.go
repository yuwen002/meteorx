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

func main() {
	applyFlag := flag.Bool("apply", false, "自动修复文档（添加缺失路由 / 删除多余路由）")
	flag.Parse()

	codeSet := buildCodeRouteSet()

	syncOpenAPI(codeSet, *applyFlag)
	syncApifox(codeSet, *applyFlag)
}

type routeEntry struct {
	method  string
	pattern string
}

func buildCodeRouteSet() map[string]routeEntry {
	r := buildRouter()
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
	return codeSet
}

func syncOpenAPI(codeSet map[string]routeEntry, apply bool) {
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
		if _, exists := codeSet[key]; !exists {
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

	if apply {
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

			auth.RegisterAPITokenRoutes(r, &authHandler.APITokenHandler{})
			oauth.RegisterProtectedRoutes(r, &oauthHandler.OAuthHandler{})
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

func syncApifox(codeSet map[string]routeEntry, apply bool) {
	data, err := os.ReadFile("docs/apifox/MeteorX-backend.apifox.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取 Apifox 文件失败: %v\n", err)
		os.Exit(1)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "解析 Apifox JSON 失败: %v\n", err)
		os.Exit(1)
	}

	collection, _ := doc["apiCollection"].([]interface{})
	apifoxSet := make(map[string]bool)
	groupMap := make(map[string]map[string]interface{})

	var collectRoutes func(items []interface{}, groupID string)
	collectRoutes = func(items []interface{}, groupID string) {
		for _, item := range items {
			obj, _ := item.(map[string]interface{})
			if api, ok := obj["api"].(map[string]interface{}); ok {
				method := strings.ToUpper(api["method"].(string))
				path := api["path"].(string)
				key := method + " " + normalizePattern(path)
				apifoxSet[key] = true
				if groupID != "" {
					groupMap[key] = obj
				}
			} else if subItems, ok := obj["items"].([]interface{}); ok {
				subID := fmt.Sprintf("%v", obj["id"])
				collectRoutes(subItems, subID)
			}
		}
	}

	for _, item := range collection {
		obj, _ := item.(map[string]interface{})
		rootID := fmt.Sprintf("%v", obj["id"])
		items, _ := obj["items"].([]interface{})
		collectRoutes(items, rootID)
	}

	fmt.Println("\n======== 代码路由 vs Apifox 对比 ========\n")

	var missingKeys []string
	var extraKeys []string

	for key := range codeSet {
		if !apifoxSet[key] {
			missingKeys = append(missingKeys, key)
		}
	}
	for key := range apifoxSet {
		if _, exists := codeSet[key]; !exists {
			extraKeys = append(extraKeys, key)
		}
	}

	sort.Strings(missingKeys)
	sort.Strings(extraKeys)

	fmt.Printf("代码路由总数: %d\n", len(codeSet))
	fmt.Printf("Apifox 路由总数: %d\n\n", len(apifoxSet))

	if len(missingKeys) > 0 {
		fmt.Printf("⚠️  Apifox 缺失 %d 个路由：\n", len(missingKeys))
		for _, k := range missingKeys {
			fmt.Printf("  + %s\n", k)
		}
		fmt.Println()
	} else {
		fmt.Println("✅ Apifox 无缺失路由")
	}

	if len(extraKeys) > 0 {
		fmt.Printf("⚠️  Apifox 多出 %d 个路由：\n", len(extraKeys))
		for _, k := range extraKeys {
			fmt.Printf("  - %s\n", k)
		}
		fmt.Println()
	} else {
		fmt.Println("✅ Apifox 无多余路由")
	}

	if len(missingKeys) == 0 && len(extraKeys) == 0 {
		fmt.Println("\n🎉 完美匹配！")
		return
	}

	if apply {
		fmt.Println("\n===== 应用修复 =====")

		addedCount := 0
		for _, key := range missingKeys {
			e := codeSet[key]
			newItem := buildApifoxAPIItem(e.method, e.pattern)
			targetGroup := findApifoxGroup(collection, e.pattern)
			if targetGroup != nil {
				items, _ := targetGroup["items"].([]interface{})
				targetGroup["items"] = append(items, newItem)
				addedCount++
				fmt.Printf("+ 添加 %s %s -> 分组 %s\n", e.method, e.pattern, targetGroup["name"])
			} else {
				fmt.Printf("⚠️  无法确定 %s %s 的分组，跳过\n", e.method, e.pattern)
			}
		}

		removedCount := 0
		for _, key := range extraKeys {
			parts := strings.SplitN(key, " ", 2)
			method := parts[0]
			path := parts[1]
			if removeApifoxRoute(collection, method, path) {
				removedCount++
				fmt.Printf("- 删除 %s %s\n", method, path)
			}
		}

		out, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "序列化失败: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile("docs/apifox/MeteorX-backend.apifox.json", out, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "写入失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n✅ 已修复：+%d / -%d\n", addedCount, removedCount)
	} else {
		fmt.Println("\n💡 加 -apply 参数可自动修复")
	}
}

func findApifoxGroup(collection []interface{}, pattern string) map[string]interface{} {
	groupName := inferApifoxGroupName(pattern)

	for _, item := range collection {
		obj, _ := item.(map[string]interface{})
		items, _ := obj["items"].([]interface{})
		for _, sub := range items {
			subObj, _ := sub.(map[string]interface{})
			name, _ := subObj["name"].(string)
			if name == groupName {
				return subObj
			}
		}
	}
	return nil
}

func inferApifoxGroupName(pattern string) string {
	if strings.HasPrefix(pattern, "/health") || strings.HasPrefix(pattern, "/metrics") {
		return "系统"
	}
	if strings.HasPrefix(pattern, "/api/v1/auth/tokens") {
		return "认证"
	}
	if strings.HasPrefix(pattern, "/api/v1/auth/oauth") || strings.HasPrefix(pattern, "/api/v1/oauth") {
		return "认证"
	}
	if strings.HasPrefix(pattern, "/api/v1/auth") {
		return "认证"
	}
	if strings.HasPrefix(pattern, "/api/v1/profile") {
		return "个人中心"
	}
	if strings.HasPrefix(pattern, "/api/v1/admin/tenants-plan") || strings.HasPrefix(pattern, "/api/v1/admin/cancel-requests") {
		return "租户管理"
	}
	if strings.HasPrefix(pattern, "/api/v1/admin/tenants") {
		return "租户管理"
	}
	if strings.HasPrefix(pattern, "/api/v1/tenants") {
		return "租户管理"
	}
	if strings.HasPrefix(pattern, "/api/v1/tenant-settings") {
		return "租户独立配置"
	}
	if strings.HasPrefix(pattern, "/api/v1/admin/tenant-users") {
		return "跨租户用户管理"
	}
	if strings.HasPrefix(pattern, "/api/v1/admin/users") {
		return "管理员用户管理"
	}
	if strings.HasPrefix(pattern, "/api/v1/users") {
		return "租户用户管理"
	}
	if strings.HasPrefix(pattern, "/api/v1/admin/stats") || strings.HasPrefix(pattern, "/api/v1/admin/dashboard") {
		return "运营看板"
	}
	if strings.HasPrefix(pattern, "/api/v1/rbac") {
		return "角色与权限"
	}
	if strings.HasPrefix(pattern, "/api/v1/audit/alert") {
		return "告警管理"
	}
	if strings.HasPrefix(pattern, "/api/v1/audit/sessions") {
		return "会话分析"
	}
	if strings.HasPrefix(pattern, "/api/v1/audit") {
		return "审计日志"
	}
	if strings.HasPrefix(pattern, "/api/v1/files") {
		return "文件管理"
	}
	if strings.HasPrefix(pattern, "/api/v1/plan") || strings.HasPrefix(pattern, "/api/v1/admin/plans") || strings.HasPrefix(pattern, "/api/v1/admin/subscriptions") {
		return "套餐管理"
	}
	if strings.HasPrefix(pattern, "/api/v1/admin/announcements") || strings.HasPrefix(pattern, "/api/v1/announcements") {
		return "通知公告"
	}
	if strings.HasPrefix(pattern, "/api/v1/wiki") {
		return "知识库"
	}
	if strings.HasPrefix(pattern, "/api/v1/ws") {
		return "系统"
	}
	return "认证"
}

var apifoxIDCounter int64 = 900000000

func nextApifoxID() string {
	apifoxIDCounter++
	return fmt.Sprintf("%d", apifoxIDCounter)
}

func buildApifoxAPIItem(method, pattern string) map[string]interface{} {
	methodLower := strings.ToLower(method)
	tags := inferTags(pattern)
	perm := middleware.DerivePermissionCode(method, pattern)
	summary := inferSummary(method, pattern)

	apiID := nextApifoxID()
	respID := nextApifoxID()
	caseID := nextApifoxID()

	tagStrs := make([]interface{}, len(tags))
	for i, t := range tags {
		tagStrs[i] = t
	}

	return map[string]interface{}{
		"name": summary,
		"api": map[string]interface{}{
			"id":       apiID,
			"method":   methodLower,
			"path":     pattern,
			"parameters": map[string]interface{}{
				"path":   []interface{}{},
				"query":  []interface{}{},
				"cookie": []interface{}{},
				"header": []interface{}{},
			},
			"auth":            map[string]interface{}{},
			"securityScheme":  map[string]interface{}{},
			"commonParameters": map[string]interface{}{},
			"responses": []interface{}{
				map[string]interface{}{
					"id":          respID,
					"code":        "200",
					"headers":     []interface{}{},
					"jsonSchema":  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
					"itemSchema":  map[string]interface{}{},
					"description": "成功",
					"contentType": "json",
					"mediaType":   "application/json",
					"oasExtensions": "",
				},
			},
			"responseExamples": []interface{}{},
			"requestBody": map[string]interface{}{
				"type":                   "application/json",
				"parameters":             []interface{}{},
				"jsonSchema":             map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
				"mediaType":              "application/json",
				"oasExtensions":          "",
				"required":               true,
				"additionalContentTypes": []interface{}{},
			},
			"description":  fmt.Sprintf("权限码: `%s`", perm),
			"tags":         tagStrs,
			"status":       "released",
			"serverId":     "",
			"operationId":  "",
			"sourceUrl":    "",
			"ordering":     100,
			"cases": []interface{}{
				map[string]interface{}{
					"id":         caseID,
					"type":       "DEBUG_CASE",
					"path":       nil,
					"name":       "成功",
					"responseId": respID,
					"parameters": map[string]interface{}{
						"path":   []interface{}{},
						"query":  []interface{}{},
						"cookie": []interface{}{},
						"header": []interface{}{},
					},
					"commonParameters": map[string]interface{}{},
					"requestBody": map[string]interface{}{
						"parameters":   []interface{}{},
						"data":         "",
						"type":         "application/json",
						"generateMode": "normal",
					},
					"auth":           map[string]interface{}{},
					"securityScheme": map[string]interface{}{},
					"advancedSettings": map[string]interface{}{
						"disabledSystemHeaders": map[string]interface{}{},
					},
					"requestResult": nil,
					"visibility":    "INHERITED",
					"moduleId":      7675120,
					"categoryId":    0,
					"tagIds":        []interface{}{},
					"apiTestDataList": []interface{}{},
					"preProcessors":  []interface{}{},
					"postProcessors": []interface{}{},
					"inheritPostProcessors": map[string]interface{}{
						"enable":        map[string]interface{}{},
						"defaultEnable": map[string]interface{}{},
					},
					"inheritPreProcessors": map[string]interface{}{
						"enable":        map[string]interface{}{},
						"defaultEnable": map[string]interface{}{},
					},
				},
			},
			"mocks":              []interface{}{},
			"customApiFields":    "{}",
			"advancedSettings":   map[string]interface{}{"disabledSystemHeaders": map[string]interface{}{}},
			"mockScript":         map[string]interface{}{},
			"codeSamples":        []interface{}{},
			"commonResponseStatus": map[string]interface{}{},
			"responseChildren":   []interface{}{},
			"visibility":         "INHERITED",
			"moduleId":           7675120,
			"oasExtensions":      "",
			"type":               "http",
			"preProcessors":      []interface{}{},
			"postProcessors":     []interface{}{},
			"inheritPostProcessors": map[string]interface{}{},
			"inheritPreProcessors":  map[string]interface{}{},
		},
	}
}

func removeApifoxRoute(collection []interface{}, method, pattern string) bool {
	methodLower := strings.ToLower(method)
	normalizedPath := normalizePattern(pattern)

	var removeFromItems func(items []interface{}) bool
	removeFromItems = func(items []interface{}) bool {
		for i, item := range items {
			obj, _ := item.(map[string]interface{})
			if api, ok := obj["api"].(map[string]interface{}); ok {
				apiMethod := api["method"].(string)
				apiPath := api["path"].(string)
				if apiMethod == methodLower && normalizePattern(apiPath) == normalizedPath {
					items = append(items[:i], items[i+1:]...)
					return true
				}
			} else if subItems, ok := obj["items"].([]interface{}); ok {
				if removeFromItems(subItems) {
					obj["items"] = subItems
					return true
				}
			}
		}
		return false
	}

	for _, item := range collection {
		obj, _ := item.(map[string]interface{})
		items, _ := obj["items"].([]interface{})
		if removeFromItems(items) {
			obj["items"] = items
			return true
		}
	}
	return false
}

func registerWikiPublicShare(r chi.Router) {
	r.Get("/wiki/share/{token}", noop())
}

func noop() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}