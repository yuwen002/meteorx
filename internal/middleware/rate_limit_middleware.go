package middleware

import (
	"net/http"

	"meteorx/internal/common/response"
	"meteorx/pkg/security"
)

// RateLimitMiddleware 接口限流中间件
func RateLimitMiddleware(limiter *security.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			identifier := security.GetClientIP(r)

			allowed, err := limiter.Allow(ctx, identifier)
			if err != nil {
				response.Fail(w, http.StatusInternalServerError, "rate limit check failed")
				return
			}

			if !allowed {
				response.Fail(w, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitWithUserMiddleware 基于用户ID的限流中间件（用于登录等敏感接口）
func RateLimitWithUserMiddleware(limiter *security.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 优先使用用户名作为标识（用于登录接口）
			identifier := r.PostFormValue("username")
			if identifier == "" {
				// 否则使用 IP
				identifier = security.GetClientIP(r)
			}

			allowed, err := limiter.Allow(ctx, identifier)
			if err != nil {
				response.Fail(w, http.StatusInternalServerError, "rate limit check failed")
				return
			}

			if !allowed {
				response.Fail(w, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
