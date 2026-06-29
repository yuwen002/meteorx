package middleware

import (
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
//           2) 提取第一个段作为"模块/资源"："users" → 单数化为 "user"
//           3) 分析 Method + 剩余路径，推导动作：
//              - GET  根路径          → list
//              - POST 根路径          → create
//              - GET  含 {id}/detail → read
//              - PUT  含 /update     → update
//              - DELETE 含 /delete   → delete
//              - PUT  含 /status     → status
//              - PUT  含 /permissions → bind_perm
//              - DELETE 含 /permissions → unbind_perm
//              - GET  含 /permissions → get_perms
//              - PUT  含 {id}/roles  → assign
//              - GET  含 /roles      → get_roles
//           4) 组合为: "rbac:{资源}:{动作}" 或 "{资源}:{动作}"
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
			if contextxHasRole(r.Context(), "superadmin") {
				next.ServeHTTP(w, r)
				return
			}

			userID := contextxGetUserID(r.Context())
			if userID == "" {
				responseFail(w, http.StatusForbidden, "权限不足")
				return
			}

			// 1. 从 chi 获取匹配到的路由 pattern
			permCode := derivePermissionCode(r)
			if permCode == "" {
				// 无法自动推导权限码（如未匹配到 chi 路由），直接放行由业务层判断
				next.ServeHTTP(w, r)
				return
			}

			// 2. 查询用户权限并校验
			codes, err := checker.GetUserPermissionCodes(r.Context(), userID)
			if err != nil {
				responseFail(w, http.StatusInternalServerError, "权限检查失败")
				return
			}
			for _, code := range codes {
				if code == permCode {
					next.ServeHTTP(w, r)
					return
				}
			}

			responseFail(w, http.StatusForbidden, "权限不足，缺少权限: "+permCode)
		})
	}
}

// derivePermissionCode 根据 HTTP 请求推导权限码
func derivePermissionCode(r *http.Request) string {
	rc := chi.RouteContext(r.Context())
	if rc == nil || len(rc.RoutePatterns) == 0 {
		return ""
	}

	// 获取最后一个路由 pattern（chi 的嵌套路由会累积多个 pattern）
	pattern := rc.RoutePatterns[len(rc.RoutePatterns)-1]

	// 去掉前缀 /api/v1/
	pattern = strings.TrimPrefix(pattern, "/api/v1/")

	// 分割路径段
	parts := strings.Split(strings.Trim(pattern, "/"), "/")
	if len(parts) == 0 {
		return ""
	}

	// 检测模块前缀：第一段是 "rbac" 说明是 rbac 子模块
	hasRBACPrefix := parts[0] == "rbac"
	// 检测是否是用户模块（位于 /users 下的 user-roles 这类特殊路由）
	hasUserPrefix := false
	if !hasRBACPrefix && (parts[0] == "users" || strings.HasPrefix(parts[0], "user")) {
		hasUserPrefix = true
	}

	// 核心资源名：去掉 "rbac" 前缀后取第一段
	// 例如: rbac/roles/{id}/permissions → 核心资源 = "roles"
	var coreParts []string
	if hasRBACPrefix {
		coreParts = parts[1:]
	} else if hasUserPrefix {
		coreParts = parts[1:] // users/{id}/roles → 核心是 roles
		// 但如果是 /users/ 本身的常规 CRUD，coreParts 为空，此时将 users 作为资源
		if len(coreParts) == 0 {
			coreParts = []string{"users"}
		}
	} else {
		coreParts = parts
	}

	if len(coreParts) == 0 {
		return ""
	}

	// 资源名：第一个段转为单数
	// roles → role, permissions → perm, users → user
	resource := singularize(coreParts[0])

	// 分析剩余路径
	remaining := strings.Join(coreParts[1:], "/")
	method := strings.ToUpper(r.Method)

	// ====== 基于剩余路径和 Method 推导动作 ======
	action := deriveAction(method, remaining)
	if action == "" {
		return ""
	}

	// 组合权限码
	if hasRBACPrefix {
		return "rbac:" + resource + ":" + action
	}
	if hasUserPrefix {
		// /users/{id}/roles 这类在 /users 下的 user-role 分配接口
		if resource == "role" {
			return "rbac:user_role:" + action
		}
	}
	return resource + ":" + action
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

// deriveAction 根据 method + 剩余路径推导动作
func deriveAction(method, remaining string) string {
	// 特殊路径匹配（优先匹配更具体的路径）
	switch {
	case strings.Contains(remaining, "batch/status"):
		return "batch_status"
	case strings.Contains(remaining, "batch/delete"):
		return "batch_delete"
	case strings.Contains(remaining, "batch/permissions"):
		if method == "PUT" {
			return "batch_bind"
		}
		if method == "DELETE" {
			return "batch_unbind"
		}
	case strings.Contains(remaining, "batch/assign"):
		return "batch_assign"
	case strings.Contains(remaining, "deleted"):
		if method == "GET" {
			return "list_deleted"
		}
		if method == "PUT" {
			return "restore"
		}
	case strings.Contains(remaining, "restore"):
		return "restore"
	case strings.Contains(remaining, "status"):
		if strings.Contains(remaining, "batch") {
			return "batch_status"
		}
		return "status"
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
	case strings.Contains(remaining, "roles/batch"):
		if method == "DELETE" {
			return "batch_unassign"
		}
	case strings.Contains(remaining, "roles"):
		if method == "POST" {
			return "assign"
		}
		if method == "GET" {
			// GET /users/{id}/roles → get_roles; GET /rbac/roles/{id}/users → get_users
			if strings.Contains(remaining, "{id}/roles") || strings.Contains(remaining, "/roles") {
				// 检查是否是 role/{id}/users
				if strings.Contains(remaining, "roles/{id}/users") ||
					strings.Contains(remaining, "role/{id}/users") ||
					strings.Contains(remaining, "/users") {
					return "get_users"
				}
				return "get_roles"
			}
			return "get_roles"
		}
		if method == "DELETE" {
			// DELETE /users/{id}/roles/{role_id} → remove_one
			// DELETE /users/{id}/roles → remove_all
			if strings.Count(remaining, "{") >= 2 || strings.Count(remaining, "}") >= 2 {
				return "remove_one"
			}
			// 判断是否以 /{role_id} 这种结尾
			if strings.HasSuffix(remaining, "{role_id}") || strings.Contains(remaining, "{") && strings.Contains(remaining, "}") &&
				!strings.Contains(remaining, "batch") {
				return "remove_one"
			}
			return "remove_all"
		}
	case strings.Contains(remaining, "users"):
		// GET /rbac/user-roles/roles/{id}/users → get_users
		if method == "GET" {
			return "get_users"
		}
	}

	// 常规 REST CRUD：根据剩余路径中是否有 {id} 来判断
	hasID := strings.Contains(remaining, "{") || strings.Contains(remaining, "}") ||
		remaining == "" && false // 占位

	// 如果剩余路径为空，或含其他非详情路径
	switch {
	case remaining == "" || remaining == "/":
		if method == "GET" {
			return "list"
		}
		if method == "POST" {
			return "create"
		}
	case hasID || strings.Contains(remaining, "{id}"):
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
		}
	}

	// 兜底：根据 method 映射
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