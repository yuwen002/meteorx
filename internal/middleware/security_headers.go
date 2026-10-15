package middleware

import (
	"net/http"
)

// SecurityHeaders 为所有响应注入基础安全头，作为反向代理之外的纵深防御层：
// 即便有请求绕过 nginx 直连应用端口（如本地/容器网络内访问 :8081），也能保证
// nosniff、点击劫持防护与引用策略生效。
//
// 关于 HSTS：严格遵循「谁终结 TLS 谁负责 HSTS」。仅当本进程直接持有 TLS 连接
// （r.TLS != nil）时才下发；若前置存在反向代理（请求带 X-Forwarded-Proto），TLS 由
// 边缘层终结，HSTS 应交由该层（nginx）统一下发——否则响应会出现重复的
// Strict-Transport-Security 头，且各层 max-age 策略不一致时语义更难维护。
//
// 注意：内容安全策略（CSP）属于文档级策略，由前端 nginx 统一下发，此处不重复设置，
// 以免给 JSON 接口附加无意义的头。
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// 仅在本进程直接终结 TLS 时下发 HSTS；被反代挡在前面时交给边缘层，避免重复头。
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}
