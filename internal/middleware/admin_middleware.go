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
			ctx := r.Context()

			// 1. 从 contextx 中取出 Role 或者 IsMaster 标记
			role, _ := ctx.Value(contextx.RoleKey).(string)

			// 假设你未来在 contextx 扩展了 IsMasterKey，可以这样取：
			// isMaster, _ := ctx.Value(contextx.IsMasterKey).(bool)

			// 2. 鉴权防线：如果不是超级管理员（比如角色不是 superadmin，或者 isMaster 不为 true）
			// 这里根据你业务中定义“上帝视角”的规则来定，我们先以 role == "superadmin" 为例
			if role != "superadmin" {
				response.Fail(w, http.StatusForbidden, "权限不足，仅限平台超级管理员访问")
				return
			}

			// 3. 校验通过，放行
			next.ServeHTTP(w, r)
		})
	}
}
