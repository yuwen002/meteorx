package middleware

import (
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// AutoRequirePermission 自动权限校验中间件：
// 根据当前请求的 HTTP Method + URL Pattern 自动推导所需的权限码，
// 然后调用 PermissionChecker 校验用户是否拥有该权限。
//
// 推导规则（约定）：
//   URL:     /api/v1/users/{id}/detail
//   Method:  GET
//   步骤:   1) 去掉前缀 "/api/v1/"，得到 "users/{id}/detail"
//           2) 识别命名空间前缀：rbac / admin / 其他
//           3) 提取第一个段作为"模块/资源"，根据命名空间映射得到资源名
//           4) 分析 Method + 剩余路径，推导动作
//           5) 组合为: "rbac:{资源}:{动作}" / "admin:{资源}:{动作}" / "{资源}:{动作}"
//
// 推导示例：
//   GET    /api/v1/users                   → user:list
//   POST   /api/v1/users                   → user:create
//   GET    /api/v1/users/{id}/detail       → user:read
//   PUT    /api/v1/users/{id}/update       → user:update
//   DELETE /api/v1/users/{id}/delete       → user:delete
//   GET    /api/v1/rbac/roles              → rbac:role:list
//   POST   /api/v1/rbac/roles              → rbac:role:create
//   PUT    /api/v1/rbac/roles/{id}/permissions → rbac:role:bind_perm
//   GET    /api/v1/rbac/role-permissions   → rbac:role_perm:list
//   GET    /api/v1/rbac/user-roles         → rbac:user_role:list
//   POST   /api/v1/rbac/user-roles/{user_id}/roles → rbac:user_role:assign
//   GET    /api/v1/rbac/user-roles/{user_id}/roles → rbac:user_role:get_roles
//   DELETE /api/v1/rbac/user-roles/{user_id}/roles/{role_id} → rbac:user_role:remove_one
//   DELETE /api/v1/rbac/user-roles/{user_id}/roles → rbac:user_role:remove_all
//   GET    /api/v1/admin/users             → admin:master:list
//   POST   /api/v1/admin/users             → admin:master:create
//   GET    /api/v1/admin/tenants           → admin:tenant:list
//   POST   /api/v1/admin/tenants           → admin:tenant:create
//   GET    /api/v1/admin/tenant-users      → admin:tenant_user:list
//   POST   /api/v1/admin/tenant-users      → admin:tenant_user:create
//   GET    /api/v1/admin/plans             → admin:plan:list
//   PUT    /api/v1/admin/tenants/{id}/plan → admin:plan:assign
//
// 用法示例：
//
//   r.Route("/users", func(r chi.Router) {
//       r.Use(middleware.AutoRequirePermission(checker))
//       r.Get("/", h.ListUsers)              // → 自动需要 user:list
//       r.Post("/", h.CreateUser)            // → 自动需要 user:create
//       r.Get("/{id}/detail", h.GetUser)     // → 自动需要 user:read
//       r.Put("/{id}/update", h.UpdateUser)  // → 自动需要 user:update
//       r.Delete("/{id}/delete", h.DeleteUser)// → 自动需要 user:delete
//   })
//
// 超级管理员（contextx.HasRole(ctx, "superadmin")）直接放行。
func AutoRequirePermission(checker PermissionChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 超级管理员直接放行
			if contextx.HasRole(r.Context(), "superadmin") {
				next.ServeHTTP(w, r)
				return
			}

			userID := contextx.GetUserID(r.Context())
			if userID == "" {
				response.Fail(w, http.StatusForbidden, "权限不足")
				return
			}

			// 1. 从 chi 获取匹配到的路由 pattern，推导权限码
			permCode := derivePermissionCode(r)
			if permCode == "" {
				// 无法自动推导权限码（如未匹配到 chi 路由），
				// 直接放行由业务层判断
				next.ServeHTTP(w, r)
				return
			}

			// 2. 查询用户权限并校验
			codes, err := checker.GetUserPermissionCodes(r.Context(), userID)
			if err != nil {
				response.Fail(w, http.StatusInternalServerError, "权限检查失败")
				return
			}
			for _, code := range codes {
				if code == permCode {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Fail(w, http.StatusForbidden, "权限不足，缺少权限: "+permCode)
		})
	}
}

// derivePermissionCode 根据 HTTP 请求推导权限码
// 返回空字符串表示无法推导（此时中间件放行，业务层自行判断）
func derivePermissionCode(r *http.Request) string {
	rc := chi.RouteContext(r.Context())
	if rc == nil || len(rc.RoutePatterns) == 0 {
		return ""
	}

	// 使用 URL 路径作为权限码推导源（chi 已完成路由匹配，保证路径合法）
	urlPath := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")

	if len(parts) == 0 {
		return ""
	}

	// ====== Step 1: 判断命名空间前缀 ======
	// rbac 前缀 → 生成 rbac:{resource}:{action}
	// admin 前缀 → 生成 admin:{resource}:{action}（平台后台管理接口）
	// 其他 → 生成 {resource}:{action}
	hasRBACPrefix := parts[0] == "rbac"
	hasAdminPrefix := parts[0] == "admin"

	// 核心路径段（去掉命名空间前缀后剩余）
	coreParts := parts
	if hasRBACPrefix || hasAdminPrefix {
		coreParts = parts[1:]
	}
	if len(coreParts) == 0 {
		return ""
	}

	// ====== Step 2: 从核心路径中提取资源名 ======
	// admin 前缀下的资源需要特殊映射，使其与后端权限码常量保持一致
	var resource string
	if hasAdminPrefix {
		resource = adminResourceName(coreParts[0])
	} else {
		resource = singularize(coreParts[0])
	}

	// ====== Step 3: 分析剩余路径段推导 action ======
	// 注意：由于使用 URL 路径（而非 chi pattern），剩余段中不含 {xxx} 占位符，
	// deriveAction 中对 {xxx} 的检查在非参数化路径上仍然有效（如 batch/delete、
	// deleted、restore、detail、update 等关键字）。
	// 对于确实依赖 {id} 占位符的逻辑（如 remove_one vs remove_all），
	// 我们额外将参数位置替换为 {param} 以兼容。
	remaining := strings.Join(coreParts[1:], "/")

	// 规范化：将 chi URL 参数对应的段替换为 {param}
	// 收集所有非空参数值
	paramValues := make(map[string]bool)
	for _, p := range rc.RoutePatterns {
		// 遍历所有 pattern，提取其中的 {xxx} 参数名
		for {
			start := strings.Index(p, "{")
			if start == -1 {
				break
			}
			end := strings.Index(p[start:], "}")
			if end == -1 {
				break
			}
			name := p[start+1 : start+end]
			val := rc.URLParam(name)
			if val != "" {
				paramValues[val] = true
			}
			p = p[start+end:]
		}
	}

	// 规范化 remaining 中的参数值 → {param}
	if len(paramValues) > 0 && remaining != "" {
		normParts := strings.Split(remaining, "/")
		for i, seg := range normParts {
			if paramValues[seg] {
				normParts[i] = "{param}"
			}
		}
		remaining = strings.Join(normParts, "/")
	}

	method := strings.ToUpper(r.Method)

	action := deriveAction(method, remaining)
	if action == "" {
		return ""
	}

	// ====== Step 4: 组合权限码 ======
	switch {
	case hasRBACPrefix:
		return "rbac:" + resource + ":" + action
	case hasAdminPrefix:
		return "admin:" + resource + ":" + action
	default:
		return resource + ":" + action
	}
}

// singularize 将复数形式资源名转为单数
func singularize(part string) string {
	switch part {
	case "users":
		return "user"
	case "roles":
		return "role"
	case "permissions":
		return "perm"
	case "role-permissions", "role_permissions":
		return "role_perm"
	case "user-roles", "user_roles":
		return "user_role"
	case "tenant-users", "tenant_users":
		return "user"
	default:
		return part
	}
}

// adminResourceName 将 admin 命名空间下的资源路径段映射为后端权限码使用的资源名
// 例如：users → master（系统管理员）、tenants → tenant、tenant-users → tenant_user、plans → plan
func adminResourceName(part string) string {
	switch part {
	case "users":
		return "master"
	case "tenants":
		return "tenant"
	case "tenant-users", "tenant_users":
		return "tenant_user"
	case "plans":
		return "plan"
	default:
		return singularize(part)
	}
}

// deriveAction 根据 method + 剩余路径推导动作
// remaining 是去掉资源段后的路径（含 {param} 占位符或具体关键字如 batch/deleted）
func deriveAction(method, remaining string) string {
	// ===== 特殊路径：精确匹配（优先级最高）=====

	// 永久删除：/{param}/permanent 或 /permanent
	if strings.Contains(remaining, "permanent") {
		if method == "DELETE" {
			return "permanent_delete"
		}
	}

	// 重置密码：reset-password 或 reset_password
	if strings.Contains(remaining, "reset-password") || strings.Contains(remaining, "reset_password") {
		if method == "PUT" || method == "POST" {
			return "reset_password"
		}
	}

	// 批量操作：batch 可以作为独立段、前缀、后缀或中间段
	// 如 "batch"、"batch/status"、"{param}/batch/delete"、"{param}/batch" 等
	if strings.Contains(remaining, "batch") {
		switch {
		case strings.Contains(remaining, "status"):
			return "batch_status"
		case strings.Contains(remaining, "delete"):
			return "batch_delete"
		case strings.Contains(remaining, "permissions"):
			if method == "PUT" {
				return "batch_bind"
			}
			if method == "DELETE" {
				return "batch_unbind"
			}
		case strings.Contains(remaining, "assign"):
			return "batch_assign"
		default:
			// 仅 "batch" 段本身 → 根据 HTTP 方法推断
			switch method {
			case "DELETE":
				return "batch_delete"
			case "PUT":
				return "batch_status"
			}
		}
	}

	// 回收站
	if strings.Contains(remaining, "deleted") {
		if method == "GET" {
			return "list_deleted"
		}
		if method == "PUT" {
			return "restore"
		}
	}

	if strings.Contains(remaining, "restore") {
		return "restore"
	}

	// 权限绑定/解绑
	switch {
	case strings.Contains(remaining, "permissions/batch"):
		if method == "DELETE" {
			return "batch_unbind_perm"
		}
	case strings.Contains(remaining, "permissions"):
		if method == "PUT" {
			return "bind_perm"
		}
		if method == "DELETE" {
			return "unbind_perm"
		}
		if method == "GET" {
			return "get_perms"
		}
	}

	// 用户-角色分配（rbac/user-roles/{user_id}/roles 或 /users/{id}/roles）
	if strings.Contains(remaining, "roles") {
		if method == "POST" {
			return "assign"
		}
		if method == "GET" {
			if strings.Contains(remaining, "/users") || strings.HasSuffix(remaining, "users") {
				return "get_users"
			}
			return "get_roles"
		}
		if method == "DELETE" {
			// 用 {param} 的数量来判断：2 个 → remove_one，1 个 → remove_all
			paramCount := strings.Count(remaining, "{param}")
			if paramCount >= 2 {
				return "remove_one"
			}
			if paramCount == 1 {
				// DELETE {param}/roles → remove_all（只有用户ID，没有角色ID）
				return "remove_all"
			}
			// 没有参数但有 roles → 也是 remove_all
			return "remove_all"
		}
	}

	// ===== 常规 REST CRUD =====
	hasIDPlaceholder := strings.Contains(remaining, "{param}") || strings.Contains(remaining, "{")

	switch {
	case remaining == "" || remaining == "/":
		switch method {
		case "GET":
			return "list"
		case "POST":
			return "create"
		}

	case hasIDPlaceholder:
		switch {
		case strings.Contains(remaining, "detail"):
			if method == "GET" {
				return "read"
			}
		case strings.Contains(remaining, "update"):
			if method == "PUT" {
				return "update"
			}
		case strings.Contains(remaining, "delete"):
			if method == "DELETE" {
				return "delete"
			}
		case strings.Contains(remaining, "status"):
			if method == "PUT" {
				return "status"
			}
		case strings.Contains(remaining, "permanent"):
			if method == "DELETE" {
				return "permanent_delete"
			}
		}
	}

	// ===== 兜底：按 Method 映射 =====
	switch method {
	case "GET":
		return "list"
	case "POST":
		return "create"
	case "PUT":
		return "update"
	case "DELETE":
		return "delete"
	}

	return ""
}