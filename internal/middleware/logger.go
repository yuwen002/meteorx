package middleware

import (
	"fmt"
	"net/http"
	"time"

	"meteorx/pkg/logger"

	"github.com/google/uuid"
)

// ResponseWriter 包装 http.ResponseWriter 以捕获状态码
type ResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

// NewResponseWriter 创建 ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		StatusCode:     http.StatusOK,
	}
}

// WriteHeader 捕获状态码
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logger 结构化日志中间件
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 生成 Request ID
		requestID := uuid.New().String()
		ctx := logger.WithRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)

		// 包装 ResponseWriter
		rw := NewResponseWriter(w)

		// 处理请求
		next.ServeHTTP(rw, r)

		// 计算耗时
		duration := time.Since(start)

		// 记录结构化日志
		logger.Info("HTTP Request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", rw.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)

		// 设置响应头（方便客户端追踪）
		w.Header().Set("X-Request-ID", requestID)
	})
}

// Recovery 恢复 panic 并记录错误日志
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := logger.GetRequestID(r.Context())

				logger.Error("Panic recovered",
					"request_id", requestID,
					"error", fmt.Sprintf("%v", err),
					"method", r.Method,
					"path", r.URL.Path,
				)

				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}