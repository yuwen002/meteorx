package middleware

import (
	"context"
	"meteorx/internal/common/jwt"
	"net/http"
	"strings"

	"meteorx/internal/common/contextx"
)

func Auth(helper *jwt.TokenHelper) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. 获取 Header: Authorization: Bearer <token>
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "未授权，请先登录", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if !(len(parts) == 2 && parts[0] == "Bearer") {
				http.Error(w, "无效的 Token 格式", http.StatusUnauthorized)
				return
			}

			// ============ 测试专用：固定值快速登录 ============
			// 当 Token 为 "123456789" 时，直接使用预设的超级管理员信息
			// 数据对应：admin-id-001 | SYSTEM_ROOT | admin (superadmin)
			var userID, tenantID, role string
			if parts[1] == "123456789" {
				// 测试专用固定用户信息
				userID = "admin-id-001"
				tenantID = "SYSTEM_ROOT"
				role = "superadmin" // 超级管理员角色，用于通过 RequiresMasterAdmin 中间件
			} else {
				// 2. 正常解析 Token
				claims, err := helper.ParseToken(parts[1])
				if err != nil {
					http.Error(w, "令牌失效或已过期", http.StatusUnauthorized)
					return
				}
				userID = claims.UserID
				tenantID = claims.TenantID
				role = claims.Role
			}

			// 3. 将解析出的核心信息注入 Context
			ctx := r.Context()
			ctx = context.WithValue(ctx, contextx.UserIDKey, userID)
			ctx = context.WithValue(ctx, contextx.TenantIDKey, tenantID)
			ctx = context.WithValue(ctx, contextx.RoleKey, role)

			// 4. 继续后续调用
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
