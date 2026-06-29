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
//           2) 提取第一个段作为"模块/资源"："users" → 单数化为 "user"
//           3) 分析 Method + 剩余路径，推导动作
//           4) 组合为: "rbac:{资源}:{动作}" 或 "{资源}:{动作}"
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

	// 获取最后一个路由 pattern（chi 的嵌套路由会累积多个 pattern）
	pattern := rc.RoutePatterns[len(rc.RoutePatterns)-1]

	// 去掉前缀 /api/v1/
	pattern = strings.TrimPrefix(pattern, "/api/v1/")

	// 分割路径段
	parts := strings.Split(strings.Trim(pattern, "/"), "/")
	if len(parts) == 0 {
		return ""
	}

	// ====== Step 1: 判断模块前缀 ======
	hasRBACPrefix := parts[0] == "rbac"

	// 核心路径段（去掉 rbac 前缀后剩余）
	var coreParts []string
	if hasRBACPrefix {
		coreParts = parts[1:]
	} else {
		coreParts = parts
	}
	if len(coreParts) == 0 {
		return ""
	}

	// ====== Step 2: 从核心路径中提取资源名 ======
	resource := singularize(coreParts[0])

	// ====== Step 3: 分析剩余路径段推导 action ======
	remaining := strings.Join(coreParts[1:], "/")
	method := strings.ToUpper(r.Method)

	action := deriveAction(method, remaining)
	if action == "" {
		return ""
	}

	// ====== Step 4: 组合权限码 ======
	if hasRBACPrefix {
		return "rbac:" + resource + ":" + action
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
	// ===== 特殊路径：精确匹配（优先级最高）=====
	switch {
	// 批量操作
	case strings.Contains(remaining, "batch"):
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
		}

	// 回收站
	case strings.Contains(remaining, "deleted"):
		if method == "GET" {
			return "list_deleted"
		}
		if method == "PUT" {
			return "restore"
		}
	case strings.Contains(remaining, "restore"):
		return "restore"

	// 权限绑定/解绑
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

	// 用户-角色分配（rbac/user-roles/{user_id}/roles 或 /users/{id}/roles）
	case strings.Contains(remaining, "roles"):
		if method == "POST" {
			return "assign"
		}
		if method == "GET" {
			// 区分：GET /roles/{id}/users → get_users
			//       GET /users/{id}/roles → get_roles
			if strings.Contains(remaining, "/users") || strings.HasSuffix(remaining, "users") {
				return "get_users"
			}
			return "get_roles"
		}
		if method == "DELETE" {
			// DELETE /{user_id}/roles/{role_id} → remove_one
			// DELETE /{user_id}/roles → remove_all
			openBraces := strings.Count(remaining, "{")
			if openBraces >= 2 {
				return "remove_one"
			}
			if openBraces == 1 && strings.Contains(remaining, "role") {
				return "remove_one"
			}
			return "remove_all"
		}
	}

	// ===== 常规 REST CRUD =====
	hasIDPlaceholder := strings.Contains(remaining, "{")

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