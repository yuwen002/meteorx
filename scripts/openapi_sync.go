//go:build ignore

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"meteorx/internal/middleware"
	"meteorx/internal/routetable"

	"github.com/go-chi/chi/v5"
)

// openapi_sync 是一个以 //go:build ignore 隔离的开发工具脚本：
// 从代码构建出路由集合，与 docs/apifox 下的 OpenAPI、Apifox 文档对比，
// 加 -apply 参数可自动补齐缺失路由、删除多余路由，保持文档与代码一致。
func main() {
	applyFlag := flag.Bool("apply", false, "自动修复文档（添加缺失路由 / 删除多余路由）")
	checkFlag := flag.Bool("check", false, "门禁模式：存在差异时以非 0 退出码结束（供 CI 使用，与 -apply 同用时不做失败判定）")
	flag.Parse()

	codeSet := buildCodeRouteSet()

	openapiDiff := syncOpenAPI(codeSet, *applyFlag)
	apifoxDiff := syncApifox(codeSet, *applyFlag)

	// 不加 -check 时保持旧行为（仅打印差异、退出码为 0）；CI 用 -check 把漂移变成失败。
	if *checkFlag && !*applyFlag && (openapiDiff || apifoxDiff) {
		fmt.Fprintln(os.Stderr, "\n❌ 文档与代码路由不一致（-check 模式判定失败）")
		os.Exit(1)
	}
}

// routeEntry 描述一条路由的方法与已归一化的路径。
type routeEntry struct {
	method  string
	pattern string
}

// buildCodeRouteSet 遍历代码路由，构建 "METHOD /path" 到路由条目的集合。
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

// syncOpenAPI 对比代码路由与 OpenAPI 文档差异，apply 为 true 时将修复写回文件。
// 返回值为 true 表示本次对比发现差异（无论是否已修复）。
func syncOpenAPI(codeSet map[string]routeEntry, apply bool) bool {
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
		return false
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
	return true
}

// addMissingRoute 向 OpenAPI paths 中补充一条缺失路由的操作定义。
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
	description := fmt.Sprintf("权限码: `%s`", perm)
	if doc := inferDocPath(pattern); doc != "" {
		description += fmt.Sprintf("\n\n接口文档: `%s`", doc)
	}

	op := map[string]interface{}{
		"summary":     inferSummary(method, pattern),
		"tags":        tags,
		"description": description,
		"responses": map[string]interface{}{
			"200": map[string]interface{}{
				"description": "成功",
			},
		},
	}

	methodsMap[methodLower] = op
}

// inferDocPath 根据路径前缀返回对应的 Markdown 接口文档路径（无对应文档时返回空串）。
func inferDocPath(pattern string) string {
	switch {
	case strings.HasPrefix(pattern, "/api/v1/chat"):
		return "docs/api/chat-api.md"
	case strings.HasPrefix(pattern, "/api/v1/messages"):
		return "docs/api/message-api.md"
	case strings.HasPrefix(pattern, "/api/v1/invitations"):
		return "docs/api/invitation-api.md"
	case strings.HasPrefix(pattern, "/api/v1/settings") || strings.HasPrefix(pattern, "/api/v1/admin/settings"):
		return "docs/api/settings-api.md"
	case strings.HasPrefix(pattern, "/api/v1/auth"):
		return "docs/api/auth-api.md"
	case strings.HasPrefix(pattern, "/api/v1/dashboard"):
		return "docs/api/dashboard-api.md"
	}
	return ""
}

// inferTags 根据路径前缀推断 OpenAPI 分组标签。
func inferTags(pattern string) []string {
	if strings.HasPrefix(pattern, "/health") || strings.HasPrefix(pattern, "/metrics") {
		return []string{"System"}
	}
	if strings.HasPrefix(pattern, "/api/v1/auth") || strings.HasPrefix(pattern, "/api/v1/oauth") {
		return []string{"Auth"}
	}
	if strings.HasPrefix(pattern, "/api/v1/csp-report") {
		return []string{"System"}
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
	if strings.HasPrefix(pattern, "/api/v1/admin/settings") {
		return []string{"Settings"}
	}
	if strings.HasPrefix(pattern, "/api/v1/settings") {
		return []string{"Settings"}
	}
	if strings.HasPrefix(pattern, "/api/v1/chat") {
		return []string{"Chat"}
	}
	if strings.HasPrefix(pattern, "/api/v1/messages") {
		return []string{"Message"}
	}
	if strings.HasPrefix(pattern, "/api/v1/invitations") {
		return []string{"Invitation"}
	}
	if strings.HasPrefix(pattern, "/api/v1/dashboard") {
		return []string{"Dashboard"}
	}
	if strings.HasPrefix(pattern, "/api/v1/admin/oauth") {
		return []string{"Auth"}
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
	if strings.HasPrefix(pattern, "/api/v1/tasks") {
		return []string{"Task"}
	}
	if strings.HasPrefix(pattern, "/api/v1/exports") {
		return []string{"Export"}
	}
	return []string{"API"}
}

// inferSummary 优先使用已知路由的中文摘要，否则根据方法与末段路径生成。
func inferSummary(method, pattern string) string {
	knownSummaries := map[string]string{
		"GET /health":                                             "健康检查",
		"GET /health/ready":                                       "就绪检查",
		"GET /metrics":                                            "Prometheus 指标",
		"GET /api/v1/ws":                                          "WebSocket 连接",
		"POST /api/v1/auth/register":                              "用户注册",
		"POST /api/v1/auth/login":                                 "用户登录",
		"POST /api/v1/auth/logout":                                "用户登出",
		"POST /api/v1/auth/forgot-password":                       "忘记密码（发送重置邮件）",
		"POST /api/v1/auth/reset-password":                        "重置密码（通过邮件令牌）",
		"GET /api/v1/auth/tokens":                                 "API Token 列表",
		"POST /api/v1/auth/tokens":                                "创建 API Token",
		"POST /api/v1/auth/tokens/revoke":                         "撤销 API Token",
		"GET /api/v1/auth/oauth/{provider}/redirect":              "获取 OAuth2 跳转链接",
		"POST /api/v1/auth/oauth/callback":                        "OAuth2 登录回调",
		"GET /api/v1/auth/oauth/tenants":                          "获取可用租户列表",
		"GET /api/v1/auth/oauth/accounts":                         "已绑定的第三方账号列表",
		"POST /api/v1/auth/oauth/bind":                            "绑定第三方账号",
		"POST /api/v1/auth/oauth/unbind":                          "解绑第三方账号",
		"POST /api/v1/auth/oauth/token/refresh":                   "刷新 OAuth Token",
		"GET /api/v1/exports":                                     "导出任务列表",
		"POST /api/v1/exports":                                    "创建导出任务",
		"GET /api/v1/exports/{id}":                                "导出任务详情",
		"GET /api/v1/exports/{id}/download":                       "下载导出文件",
		"DELETE /api/v1/exports/{id}":                             "删除导出任务",
		"POST /api/v1/users/import":                               "批量导入用户（CSV/XLSX）",
		"POST /api/v1/admin/tenant-users/import":                  "跨租户批量导入用户（CSV/XLSX）",
		"GET /api/v1/dashboard/tenant-overview":                   "租户工作台总览",
		"GET /api/v1/settings/public":                             "公开运行时配置",
		"GET /api/v1/admin/settings":                              "平台配置列表",
		"PUT /api/v1/admin/settings":                              "批量更新平台配置",
		"POST /api/v1/auth/email/send-verification":               "发送邮箱验证邮件",
		"POST /api/v1/auth/email/verify":                          "验证邮箱",
		"POST /api/v1/auth/2fa/setup":                             "初始化 2FA（生成密钥）",
		"POST /api/v1/auth/2fa/enable":                            "确认启用 2FA",
		"POST /api/v1/auth/2fa/disable":                           "关闭 2FA",
		"GET /api/v1/auth/2fa/recovery-codes":                     "查询备份恢复码状态",
		"POST /api/v1/auth/2fa/recovery-codes/regenerate":         "重新生成备份恢复码",
		"GET /api/v1/messages":                                    "收件箱列表",
		"GET /api/v1/messages/preferences":                        "接收偏好",
		"PUT /api/v1/messages/preferences":                        "更新接收偏好",
		"GET /api/v1/messages/unread-count":                       "未读消息数量",
		"PUT /api/v1/messages/read-all":                           "全部标记已读",
		"PUT /api/v1/messages/batch/read":                         "批量标记已读",
		"DELETE /api/v1/messages/batch/delete":                    "批量删除消息",
		"PUT /api/v1/messages/{id}/read":                          "标记单条消息已读",
		"DELETE /api/v1/messages/{id}":                            "删除消息",
		"GET /api/v1/invitations":                                 "邀请列表",
		"POST /api/v1/invitations":                                "创建邀请",
		"POST /api/v1/invitations/batch":                          "批量创建邀请",
		"GET /api/v1/invitations/roles":                           "可分配角色列表",
		"PUT /api/v1/invitations/{id}/cancel":                     "取消邀请",
		"PUT /api/v1/invitations/{id}/resend":                     "重发邀请",
		"DELETE /api/v1/invitations/{id}/delete":                  "删除邀请",
		"POST /api/v1/invitations/accept":                         "接受邀请",
		"GET /api/v1/invitations/info":                            "邀请信息",
		"POST /api/v1/chat/conversations/direct":                  "打开或创建单聊会话",
		"POST /api/v1/chat/conversations/group":                   "创建群聊会话",
		"GET /api/v1/chat/conversations":                          "会话列表",
		"PUT /api/v1/chat/conversations/{id}":                     "修改群名称",
		"GET /api/v1/chat/conversations/{id}/messages":            "历史消息分页",
		"POST /api/v1/chat/conversations/{id}/messages":           "发送消息",
		"DELETE /api/v1/chat/conversations/{id}/messages/{msgId}": "撤回消息",
		"GET /api/v1/chat/conversations/{id}/members":             "群成员列表",
		"POST /api/v1/chat/conversations/{id}/members":            "添加群成员",
		"DELETE /api/v1/chat/conversations/{id}/members/{userId}": "移除群成员",
		"POST /api/v1/chat/conversations/{id}/leave":              "退出群聊",
		"POST /api/v1/chat/conversations/{id}/read":               "标记会话已读",
		"GET /api/v1/chat/conversations/{id}/read-status":         "会话已读状态",
		"POST /api/v1/chat/conversations/{id}/typing":             "输入中提示",
		"GET /api/v1/chat/presence":                               "在线状态",
		"POST /api/v1/csp-report":                                 "上报 CSP 违规",
		"GET /api/v1/admin/oauth-providers":                       "OAuth 提供商列表",
		"POST /api/v1/admin/oauth-providers":                      "创建 OAuth 提供商",
		"GET /api/v1/admin/oauth-providers/{code}":                "OAuth 提供商详情",
		"PUT /api/v1/admin/oauth-providers/{code}":                "更新 OAuth 提供商",
		"DELETE /api/v1/admin/oauth-providers/{code}":             "删除 OAuth 提供商",
		"PUT /api/v1/admin/oauth-providers/{code}/status":         "启用/停用 OAuth 提供商",
		"GET /api/v1/tasks/ai/capabilities":                       "AI 能力探测",
		"POST /api/v1/tasks/ai/parse":                             "AI 解析自然语言为任务",
		"POST /api/v1/tasks/ai/suggest":                           "AI 任务建议",
		"POST /api/v1/tasks/ai/expand":                            "AI 任务拆解",
		"POST /api/v1/tasks/ai/summary":                           "AI 任务总结",
		"POST /api/v1/tasks/batch/assign":                         "批量指派任务",
		"POST /api/v1/tasks/batch/status":                         "批量更新任务状态",
		"POST /api/v1/tasks/batch/complete":                       "批量完成任务",
		"POST /api/v1/tasks/batch/delete":                         "批量删除任务",
		"GET /api/v1/wiki/pending-reviews":                        "待审核文档列表",
		"POST /api/v1/wiki/documents/{id}/submit-review":          "提交文档审核",
		"POST /api/v1/wiki/documents/{id}/approve":                "审核通过文档",
		"POST /api/v1/wiki/documents/{id}/reject":                 "驳回文档审核",
		"POST /api/v1/wiki/documents/{id}/publish":                "发布文档",
		"POST /api/v1/wiki/documents/{id}/unpublish":              "取消发布文档",
		"POST /api/v1/wiki/documents/{id}/archive":                "归档文档",
		"GET /api/v1/wiki/documents/{id}/review-comments":         "文档审核评论列表",
		"POST /api/v1/wiki/documents/comments/{id}/reactions":     "评论点赞切换",
		"POST /api/v1/wiki/documents/comments/{id}/resolve":       "标记评论已解决",
	}
	key := method + " " + pattern
	if s, ok := knownSummaries[key]; ok {
		return s
	}

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

// normalizePattern 统一路径参数占位符并去除末尾斜杠，用于跨文档比较。
func normalizePattern(p string) string {
	p = strings.ReplaceAll(p, "{param}", "{id}")
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = p[:len(p)-1]
	}
	return p
}

// buildRouter 从 internal/routetable 取路由树：该表由各模块导出的纯注册函数拼装，
// 与生产 bootstrap.InitRouter 的一致性由 bootstrap 包的 TestRouteTable_MatchesProductionRouter 强校验。
func buildRouter() *chi.Mux {
	return routetable.Build()
}

// 返回值为 true 表示本次对比发现差异（无论是否已修复）。
func syncApifox(codeSet map[string]routeEntry, apply bool) bool {
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
	var duplicateRoutes []string

	// 以文档内现有最大数字 ID 初始化计数器：计数器是进程级的，若固定从 900000000 起，
	// 第二次运行新增条目会与上一次已写入的 ID 撞车（历史上已发生过）。
	existingIDs := collectApifoxIDs(doc)
	initApifoxIDCounter(existingIDs)

	var collectRoutes func(items []interface{}, groupID string)
	collectRoutes = func(items []interface{}, groupID string) {
		for _, item := range items {
			obj, _ := item.(map[string]interface{})
			if api, ok := obj["api"].(map[string]interface{}); ok {
				method := strings.ToUpper(api["method"].(string))
				path := api["path"].(string)
				key := method + " " + normalizePattern(path)
				if apifoxSet[key] {
					// 集合用 map 去重，重复条目不会体现为缺失/多余，必须单独记账否则永远静默。
					duplicateRoutes = append(duplicateRoutes, key)
				}
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

	if len(duplicateRoutes) > 0 {
		fmt.Printf("⚠️  Apifox 存在 %d 条重复条目：\n", len(duplicateRoutes))
		for _, k := range duplicateRoutes {
			fmt.Printf("  = %s\n", k)
		}
		fmt.Println()
	}

	if len(missingKeys) == 0 && len(extraKeys) == 0 && len(duplicateRoutes) == 0 {
		fmt.Println("\n🎉 完美匹配！")
		return false
	}

	if apply {
		fmt.Println("\n===== 应用修复 =====")

		idFloor := apifoxIDCounter
		addedCount := 0
		rootFolder := firstApifoxRootFolder(collection)
		for _, key := range missingKeys {
			e := codeSet[key]
			newItem := buildApifoxAPIItem(e.method, e.pattern)
			targetGroup := findApifoxGroup(collection, e.pattern)
			if targetGroup == nil && rootFolder != nil {
				targetGroup = createApifoxGroup(rootFolder, inferApifoxGroupName(e.pattern))
			}
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

		// 同一路由在文档中重复出现时，Apifox 导入会相互覆盖，保留首条并清理其余。
		dupRemoved := dedupeApifoxRoutes(collection)

		// 写回前自检：本次新分配的 ID 不得与文档既有 ID 重叠，否则中止写回避免再次污染。
		var collisions []int64
		for id := idFloor + 1; id <= apifoxIDCounter; id++ {
			if existingIDs[id] {
				collisions = append(collisions, id)
			}
		}
		if len(collisions) > 0 {
			fmt.Fprintf(os.Stderr, "❌ 新增条目 ID 与既有 ID 冲突 %d 个（如 %d），已中止写回\n", len(collisions), collisions[0])
			os.Exit(1)
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
		fmt.Printf("\n✅ 已修复：+%d / -%d（重复条目 -%d）\n", addedCount, removedCount, dupRemoved)
	} else {
		fmt.Println("\n💡 加 -apply 参数可自动修复")
	}
	return true
}

// findApifoxGroup 根据推断的分组名在 Apifox 集合中定位目标分组。
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

// firstApifoxRootFolder 返回 apiCollection 的首个根集合项（其 items 为各业务分组）。
func firstApifoxRootFolder(collection []interface{}) map[string]interface{} {
	for _, item := range collection {
		if obj, ok := item.(map[string]interface{}); ok {
			if _, hasItems := obj["items"]; hasItems {
				return obj
			}
		}
	}
	return nil
}

// createApifoxGroup 在根集合项下新建一个业务分组文件夹并返回。
func createApifoxGroup(rootFolder map[string]interface{}, groupName string) map[string]interface{} {
	group := map[string]interface{}{
		"id":                    nextApifoxID(),
		"name":                  groupName,
		"description":           "",
		"items":                 []interface{}{},
		"preProcessors":         []interface{}{},
		"postProcessors":        []interface{}{},
		"inheritPreProcessors":  map[string]interface{}{},
		"inheritPostProcessors": map[string]interface{}{},
	}
	items, _ := rootFolder["items"].([]interface{})
	rootFolder["items"] = append(items, group)
	fmt.Printf("+ 新建分组 %s\n", groupName)
	return group
}

// inferApifoxGroupName 根据路径前缀推断 Apifox 中文分组名。
func inferApifoxGroupName(pattern string) string {
	if strings.HasPrefix(pattern, "/health") || strings.HasPrefix(pattern, "/metrics") {
		return "系统"
	}
	if strings.HasPrefix(pattern, "/api/v1/auth/tokens") {
		return "认证"
	}
	if strings.HasPrefix(pattern, "/api/v1/auth/oauth") || strings.HasPrefix(pattern, "/api/v1/oauth") || strings.HasPrefix(pattern, "/api/v1/admin/oauth") {
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
	if strings.HasPrefix(pattern, "/api/v1/tasks") {
		return "任务管理"
	}
	if strings.HasPrefix(pattern, "/api/v1/chat") {
		return "即时消息"
	}
	if strings.HasPrefix(pattern, "/api/v1/messages") {
		return "站内消息"
	}
	if strings.HasPrefix(pattern, "/api/v1/invitations") {
		return "成员邀请"
	}
	if strings.HasPrefix(pattern, "/api/v1/settings") || strings.HasPrefix(pattern, "/api/v1/admin/settings") {
		return "系统配置"
	}
	if strings.HasPrefix(pattern, "/api/v1/dashboard") {
		return "工作台"
	}
	if strings.HasPrefix(pattern, "/api/v1/exports") {
		return "数据导出"
	}
	if strings.HasPrefix(pattern, "/api/v1/ws") || strings.HasPrefix(pattern, "/api/v1/csp-report") {
		return "系统"
	}
	// 兜底归入「认证」仅为保持旧行为；未登记前缀意味着该路由会落错分组，必须显式提醒。
	fmt.Printf("⚠️  未识别路径前缀 %s，暂归入「认证」分组，请在 inferApifoxGroupName 补登记\n", pattern)
	return "认证"
}

// apifoxIDCounter 为新增 Apifox 条目递增生成唯一 ID；实际起点由 initApifoxIDCounter 抬高到现有最大 ID 之上。
var apifoxIDCounter int64 = 900000000

// nextApifoxID 返回下一个可用的 Apifox 条目 ID。
func nextApifoxID() string {
	apifoxIDCounter++
	return fmt.Sprintf("%d", apifoxIDCounter)
}

// collectApifoxIDs 递归收集文档中所有键为 "id" 的整数标识。
func collectApifoxIDs(v interface{}) map[int64]bool {
	ids := make(map[int64]bool)
	var walk func(node interface{})
	walk = func(node interface{}) {
		switch t := node.(type) {
		case map[string]interface{}:
			for k, val := range t {
				if k == "id" {
					if n, ok := apifoxNumericID(val); ok {
						ids[n] = true
					}
				}
				walk(val)
			}
		case []interface{}:
			for _, item := range t {
				walk(item)
			}
		}
	}
	walk(v)
	return ids
}

// initApifoxIDCounter 把自增计数器抬到现有最大 ID 之上；无历史 ID 时保持 900000000 起点。
func initApifoxIDCounter(ids map[int64]bool) {
	base := int64(900000000)
	for id := range ids {
		if id > base {
			base = id
		}
	}
	apifoxIDCounter = base
}

// apifoxNumericID 把 JSON 数字或数字字符串转为 int64；非整数或超出安全整数范围时返回 false。
func apifoxNumericID(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case float64:
		if n < 0 || n > 1<<53 {
			return 0, false
		}
		i := int64(n)
		if float64(i) != n {
			return 0, false
		}
		return i, true
	case string:
		if i, err := strconv.ParseInt(n, 10, 64); err == nil {
			return i, true
		}
	}
	return 0, false
}

// buildApifoxAPIItem 构造一条符合 Apifox 结构的 API 条目。
func buildApifoxAPIItem(method, pattern string) map[string]interface{} {
	methodLower := strings.ToLower(method)
	tags := inferTags(pattern)
	perm := middleware.DerivePermissionCode(method, pattern)
	summary := inferSummary(method, pattern)
	description := fmt.Sprintf("权限码: `%s`", perm)
	if doc := inferDocPath(pattern); doc != "" {
		description += fmt.Sprintf("\n\n接口文档: `%s`", doc)
	}

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
			"id":     apiID,
			"method": methodLower,
			"path":   pattern,
			"parameters": map[string]interface{}{
				"path":   []interface{}{},
				"query":  []interface{}{},
				"cookie": []interface{}{},
				"header": []interface{}{},
			},
			"auth":             map[string]interface{}{},
			"securityScheme":   map[string]interface{}{},
			"commonParameters": map[string]interface{}{},
			"responses": []interface{}{
				map[string]interface{}{
					"id":            respID,
					"code":          "200",
					"headers":       []interface{}{},
					"jsonSchema":    map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
					"itemSchema":    map[string]interface{}{},
					"description":   "成功",
					"contentType":   "json",
					"mediaType":     "application/json",
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
			"description": description,
			"tags":        tagStrs,
			"status":      "released",
			"serverId":    "",
			"operationId": "",
			"sourceUrl":   "",
			"ordering":    100,
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
					"requestResult":   nil,
					"visibility":      "INHERITED",
					"moduleId":        7675120,
					"categoryId":      0,
					"tagIds":          []interface{}{},
					"apiTestDataList": []interface{}{},
					"preProcessors":   []interface{}{},
					"postProcessors":  []interface{}{},
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
			"mocks":                 []interface{}{},
			"customApiFields":       "{}",
			"advancedSettings":      map[string]interface{}{"disabledSystemHeaders": map[string]interface{}{}},
			"mockScript":            map[string]interface{}{},
			"codeSamples":           []interface{}{},
			"commonResponseStatus":  map[string]interface{}{},
			"responseChildren":      []interface{}{},
			"visibility":            "INHERITED",
			"moduleId":              7675120,
			"oasExtensions":         "",
			"type":                  "http",
			"preProcessors":         []interface{}{},
			"postProcessors":        []interface{}{},
			"inheritPostProcessors": map[string]interface{}{},
			"inheritPreProcessors":  map[string]interface{}{},
		},
	}
}

// removeApifoxRoute 从 Apifox 集合中递归删除匹配方法+路径的条目，返回是否删除成功。
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

// dedupeApifoxRoutes 清理同一 方法+路径 的重复条目（保留首次出现），返回删除数量。
func dedupeApifoxRoutes(collection []interface{}) int {
	seen := make(map[string]bool)
	removed := 0

	var walkItems func(items []interface{}) []interface{}
	walkItems = func(items []interface{}) []interface{} {
		kept := make([]interface{}, 0, len(items))
		for _, item := range items {
			obj, _ := item.(map[string]interface{})
			if api, ok := obj["api"].(map[string]interface{}); ok {
				key := strings.ToUpper(api["method"].(string)) + " " + normalizePattern(api["path"].(string))
				if seen[key] {
					removed++
					fmt.Printf("- 删除重复条目 %s\n", key)
					continue
				}
				seen[key] = true
				kept = append(kept, item)
				continue
			}
			if sub, ok := obj["items"].([]interface{}); ok {
				obj["items"] = walkItems(sub)
			}
			kept = append(kept, item)
		}
		return kept
	}

	for _, item := range collection {
		obj, _ := item.(map[string]interface{})
		items, _ := obj["items"].([]interface{})
		obj["items"] = walkItems(items)
	}
	return removed
}
