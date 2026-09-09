package middleware

import (
	"net/http"
	"strings"
	"time"

	"meteorx/pkg/logger"
)

// AuditLogger 审计日志中间件
// 记录关键操作（创建、更新、删除）到日志文件
func AuditLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter
		rw := NewResponseWriter(w)

		// 处理请求
		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// 只记录写操作
		if isWriteOperation(r.Method) {
			userID := logger.GetUserID(r.Context())
			tenantID := logger.GetTenantID(r.Context())
			requestID := logger.GetRequestID(r.Context())

			logger.Ctx(r.Context()).Info("Audit Log",
				"operation", r.Method,
				"path", r.URL.Path,
				"user_id", userID,
				"tenant_id", tenantID,
				"status", rw.StatusCode,
				"duration_ms", duration.Milliseconds(),
			)
		}
	})
}

// isWriteOperation 判断是否为写操作
func isWriteOperation(method string) bool {
	method = strings.ToUpper(method)
	return method == http.MethodPost ||
		method == http.MethodPut ||
		method == http.MethodPatch ||
		method == http.MethodDelete
}