package bootstrap

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"meteorx/internal/config"
	"meteorx/internal/middleware"
)

// SetupMiddleware 集中配置全局中间件
func SetupMiddleware(r *chi.Mux, allowedOrigins []string) {
	r.Use(middleware.RequestIDMiddleware)
	r.Use(middleware.GlobalErrorHandler)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	r.Use(CorsMiddleware(allowedOrigins))
}

// BuildAllowedOrigins 汇总 CORS 跨域白名单：client.base_url + client.allowed_origins，
// 统一去除尾斜杠并去重，忽略空值
func BuildAllowedOrigins(clientCfg config.ClientConfig) []string {
	seen := make(map[string]struct{})
	collect := func(o string) {
		o = strings.TrimSuffix(strings.TrimSpace(o), "/")
		if o != "" {
			if _, ok := seen[o]; !ok {
				seen[o] = struct{}{}
			}
		}
	}
	collect(clientCfg.BaseURL)
	for _, o := range clientCfg.AllowedOrigins {
		collect(o)
	}

	origins := make([]string, 0, len(seen))
	for o := range seen {
		origins = append(origins, o)
	}
	return origins
}

// CorsMiddleware 处理跨域问题：仅对白名单内的 Origin 回显允许头，
// 未授权来源的预检请求直接拒绝（403），避免任意站点借浏览器发起跨域调用
func CorsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// 跨域请求必须命中白名单；同源请求无 Origin 头则无需处理
			allowOrigin := ""
			if origin != "" {
				for _, o := range allowedOrigins {
					if strings.EqualFold(strings.TrimSuffix(o, "/"), origin) {
						allowOrigin = origin
						break
					}
				}
			}

			// 预检请求（CORS preflight）
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "86400")
				if allowOrigin == "" {
					// 未授权来源：不返回任何 CORS 允许头
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
				w.Header().Add("Vary", "Origin")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// 普通请求：仅白名单来源回显允许头（浏览器会拦截其余跨域响应）
			if allowOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
				w.Header().Add("Vary", "Origin")
			}

			next.ServeHTTP(w, r)
		})
	}
}
