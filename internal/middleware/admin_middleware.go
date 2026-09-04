package middleware

import (
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response" // 统一用你的 response 包
	"net/http"
)

// RequiresMasterAdmin 专门拦截非全平台超级管理员的请求
func RequiresMasterAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !contextx.HasRole(r.Context(), "superadmin") {
				response.Fail(w, http.StatusForbidden, "权限不足，仅限平台超级管理员访问")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
