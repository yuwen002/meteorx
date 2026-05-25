package bootstrap

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// SetupMiddleware 集中配置全局中间件
func SetupMiddleware(r *chi.Mux) {
	// 1. 标准中间件
	r.Use(middleware.RequestID)                 // 为每个请求分配 ID
	r.Use(middleware.RealIP)                    // 获取真实 IP
	r.Use(middleware.Logger)                    // 打印日志
	r.Use(middleware.Recoverer)                 // 宕机恢复
	r.Use(middleware.Timeout(60 * time.Second)) // 设置超时

	// 2. 自定义中间件（比如 CORS）
	r.Use(CorsMiddleware)
}

// CorsMiddleware 处理跨域问题
func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
