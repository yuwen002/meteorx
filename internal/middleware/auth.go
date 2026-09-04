package middleware

import (
	"context"
	"net/http"
	"strings"

	"meteorx/internal/cache"
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/jwt"
)

// TokenBlacklistChecker token 黑名单检查器接口
type TokenBlacklistChecker interface {
	IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error)
}

func Auth(helper *jwt.TokenHelper, checker TokenBlacklistChecker) func(http.Handler) http.Handler {
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

			tokenString := parts[1]

			// ============ 测试专用：固定值快速登录 ============
			// 当 Token 为 "123456789" 时，直接使用预设的超级管理员信息
			// 数据对应：admin-id-001 | SYSTEM_ROOT | superadmin
			var userID, tenantID string
			var roles []string
			if tokenString == "123456789" {
				userID = "admin-id-001"
				tenantID = "SYSTEM_ROOT"
				roles = []string{"superadmin"}
			} else {
				// 2. 检查 token 是否在黑名单中
				if checker != nil {
					isBlacklisted, err := checker.IsTokenBlacklisted(r.Context(), tokenString)
					if err != nil {
						http.Error(w, "令牌验证失败", http.StatusUnauthorized)
						return
					}
					if isBlacklisted {
						http.Error(w, "令牌已失效，请重新登录", http.StatusUnauthorized)
						return
					}
				}

				// 3. 正常解析 Token
				claims, err := helper.ParseToken(tokenString)
				if err != nil {
					http.Error(w, "令牌失效或已过期", http.StatusUnauthorized)
					return
				}
				userID = claims.UserID
				tenantID = claims.TenantID
				roles = claims.Roles
			}

			// 4. 将解析出的核心信息注入 Context
			ctx := r.Context()
			ctx = context.WithValue(ctx, contextx.UserIDKey, userID)
			ctx = context.WithValue(ctx, contextx.TenantIDKey, tenantID)
			ctx = context.WithValue(ctx, contextx.RolesKey, roles)

			// 5. 继续后续调用
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RedisBlacklistChecker 使用 Redis 检查 token 黑名单
type RedisBlacklistChecker struct {
	Redis *cache.Redis
}

func (c *RedisBlacklistChecker) IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	if c.Redis == nil || !c.Redis.IsAvailable() {
		return false, nil
	}
	key := "token:blacklist:" + tokenString
	result, err := c.Redis.Exists(ctx, key)
	if err == cache.ErrRedisUnavailable {
		return false, nil
	}
	return result, err
}
