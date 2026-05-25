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

			// 2. 解析 Token
			claims, err := helper.ParseToken(parts[1])
			if err != nil {
				http.Error(w, "令牌失效或已过期", http.StatusUnauthorized)
				return
			}

			// 3. 将解析出的核心信息注入 Context
			ctx := r.Context()
			ctx = context.WithValue(ctx, contextx.UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, contextx.TenantIDKey, claims.TenantID)
			ctx = context.WithValue(ctx, contextx.RoleKey, claims.Role)

			// 4. 继续后续调用
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
