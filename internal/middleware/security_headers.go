package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeaders 为所有响应注入基础安全头，作为反向代理之外的纵深防御层：
// 即便有请求绕过 nginx 直连应用端口（如本地/容器网络内访问 :8081），也能保证
// nosniff、点击劫持防护与引用策略生效。Strict-Transport-Security 仅在识别到
// HTTPS（TLS 直连或 X-Forwarded-Proto=https）时才下发，避免在纯 HTTP 部署下无效输出。
//
// 注意：内容安全策略（CSP）属于文档级策略，由前端 nginx 统一下发，此处不重复设置，
// 以免给 JSON 接口附加无意义的头。
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// HSTS 仅在安全上下文中生效：优先信任反代透传的 X-Forwarded-Proto，
		// 其次判断 Go 侧是否直接终结了 TLS。
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}
