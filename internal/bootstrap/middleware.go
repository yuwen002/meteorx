package bootstrap

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"meteorx/internal/middleware"
)

// SetupMiddleware 集中配置全局中间件
func SetupMiddleware(r *chi.Mux) {
	r.Use(middleware.RequestIDMiddleware)
	r.Use(middleware.GlobalErrorHandler)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Timeout(60 * time.Second))

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