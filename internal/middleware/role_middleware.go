package middleware

import (
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"net/http"
)

// RequireRole 验证当前用户是否具有指定角色
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := contextx.GetRole(r.Context())
			if role != requiredRole {
				response.Fail(w, http.StatusForbidden, "权限不足")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin 快捷方法：验证是否为租户管理员
func RequireAdmin() func(http.Handler) http.Handler {
	return RequireRole("admin")
}