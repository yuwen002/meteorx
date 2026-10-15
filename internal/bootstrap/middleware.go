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

// SetupMiddleware 集中配置全局中间件。
// trustedProxies 为可信反向代理 CIDR 列表，用于从 X-Forwarded-For 链解析真实客户端 IP：
// 非空时右起逐段跳过命中可信段的地址，首个不可信地址即客户端；
// 为空时取 XFF 最右一条（仅适用于本服务前只有一层 nginx 反代的部署，
// 因为 nginx 总会把真实对端追加到最右）。解析结果写入请求 context，
// 由 security.GetClientIP / chimiddleware.GetClientIP 读取，不修改 RemoteAddr。
func SetupMiddleware(r *chi.Mux, allowedOrigins []string, trustedProxies []string) {
	r.Use(middleware.GlobalErrorHandler)
	// 安全响应头纵深防御：紧跟错误处理之后，确保异常/恢复路径的响应同样携带安全头
	r.Use(middleware.SecurityHeaders)
	// 替代已弃用的 chimiddleware.RealIP：后者无条件信任最左 XFF/True-Client-IP/X-Real-IP，
	// 可被客户端伪造审计来源 IP（GHSA-3fxj-6jh8-hvhx）
	r.Use(chimiddleware.ClientIPFromXFF(trustedProxies...))
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
