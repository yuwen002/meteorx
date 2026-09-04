package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"meteorx/internal/common/response"
)

type requestIDKey struct{}

func GenerateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

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

func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(requestIDKey{}).(string); ok {
		return val
	}
	return ""
}

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
