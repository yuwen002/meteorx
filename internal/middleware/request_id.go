package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"meteorx/internal/common/response"
)

type requestIDKey struct{}

// GenerateRequestID 生成一个 32 位十六进制随机请求 ID。
func GenerateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// RequestIDMiddleware 为请求分配（或复用）X-Request-ID 并写入上下文与响应头。
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = GenerateRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID 从上下文读取请求 ID，不存在时返回空串。
func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(requestIDKey{}).(string); ok {
		return val
	}
	return ""
}

// GlobalErrorHandler 捕获下游 panic，返回带请求 ID 的统一 500 响应。
func GlobalErrorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())

		defer func() {
			if rec := recover(); rec != nil {
				appErr := &struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				}{
					Code:    "INTERNAL_ERROR",
					Message: "Internal server error",
				}
				_ = rec
				response.WriteJSON(w, http.StatusInternalServerError, map[string]any{
					"code":       appErr.Code,
					"message":    appErr.Message,
					"request_id": requestID,
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}
