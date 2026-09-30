// Package middleware 提供 HTTP 中间件集合，包括认证、权限校验、审计日志、限流、请求追踪、文件服务等横切关注点。
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"meteorx/internal/cache"
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/jwt"
)

// TokenBlacklistChecker Token 黑名单检查接口
type TokenBlacklistChecker interface {
	IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error)
}

// APITokenValidator API Token 验证接口，用于长期令牌认证
type APITokenValidator interface {
	Validate(ctx context.Context, tokenString string) (userID, tenantID string, allowedPaths []string, err error)
	GetUserRoles(ctx context.Context, userID string) ([]string, error)
}

// Auth 认证中间件，支持 JWT Token 和 API Token 双模式验证
// 优先解析 JWT，若失败且 token 以 "mxat_" 开头则尝试 API Token 验证
func Auth(helper *jwt.TokenHelper, checker TokenBlacklistChecker, appMode string, allowTestBypass bool, apiTokenValidator APITokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			// 非 release 模式下允许测试绕过
			testBypassEnabled := allowTestBypass && !strings.EqualFold(appMode, "release")

			var userID, tenantID string
			var roles []string
			if testBypassEnabled && tokenString == "123456789" {
				// 测试绕过模式：使用固定测试 token
				userID = "admin-id-001"
				tenantID = "SYSTEM_ROOT"
				roles = []string{"superadmin"}
			} else {
				// 检查 token 是否在黑名单中
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

				// 尝试解析 JWT Token
				claims, err := helper.ParseToken(tokenString)
				if err != nil {
					// JWT 解析失败，尝试 API Token 验证（仅对 mxat_ 前缀的 token）
					if apiTokenValidator != nil && strings.HasPrefix(tokenString, "mxat_") {
						uid, tid, allowedPaths, verr := apiTokenValidator.Validate(r.Context(), tokenString)
						if verr != nil {
							http.Error(w, "令牌失效或已过期", http.StatusUnauthorized)
							return
						}
						// 校验路径权限：若令牌配置了 allowedPaths，则请求路径必须匹配其中之一
						if len(allowedPaths) > 0 && !matchAllowedPath(r.URL.Path, allowedPaths) {
							http.Error(w, "该令牌无权访问此路径", http.StatusForbidden)
							return
						}
						tokenRoles, rerr := apiTokenValidator.GetUserRoles(r.Context(), uid)
						if rerr != nil {
							http.Error(w, "令牌验证失败", http.StatusUnauthorized)
							return
						}
						userID = uid
						tenantID = tid
						roles = tokenRoles
						ctx := r.Context()
						ctx = context.WithValue(ctx, contextx.AllowedPathsKey, allowedPaths)
						r = r.WithContext(ctx)
					} else {
						http.Error(w, "令牌失效或已过期", http.StatusUnauthorized)
						return
					}
				} else {
					// JWT 解析成功，从 claims 中提取用户信息
					userID = claims.UserID
					tenantID = claims.TenantID
					roles = claims.Roles
				}
			}

			// 将用户信息注入请求上下文
			ctx := r.Context()
			ctx = context.WithValue(ctx, contextx.UserIDKey, userID)
			ctx = context.WithValue(ctx, contextx.TenantIDKey, tenantID)
			ctx = context.WithValue(ctx, contextx.RolesKey, roles)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RedisBlacklistChecker 使用 Redis 检查 token 黑名单
type RedisBlacklistChecker struct {
	Redis *cache.Redis
}

// IsTokenBlacklisted 检查 token 是否在 Redis 黑名单中
// Redis 不可用时返回 false，确保不因基础设施故障阻止合法请求
func (c *RedisBlacklistChecker) IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	if c.Redis == nil || !c.Redis.IsAvailable() {
		return false, nil
	}
	key := "token:blacklist:" + tokenString
	result, err := c.Redis.Exists(ctx, key)
	// Redis 不可用时视为未黑名单，放行请求
	if errors.Is(err, cache.ErrRedisUnavailable) {
		return false, nil
	}
	return result, err
}

// matchAllowedPath 检查请求路径是否匹配允许的路径前缀列表
// 路径前缀匹配：如 /users 会匹配 /users、/users/、/users/abc 等
func matchAllowedPath(requestPath string, allowedPaths []string) bool {
	for _, prefix := range allowedPaths {
		if strings.HasPrefix(requestPath, prefix) {
			return true
		}
	}
	return false
}
