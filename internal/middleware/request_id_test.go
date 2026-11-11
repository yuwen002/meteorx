package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRequestID_LengthAndCharset(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := GenerateRequestID()
		assert.Len(t, id, 32, "request id should be 32 hex chars")
		for _, c := range id {
			assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
				"id should only contain lowercase hex, got %q", string(c))
		}
	}
}

func TestGenerateRequestID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := GenerateRequestID()
		require.False(t, seen[id], "duplicate request id: %s", id)
		seen[id] = true
	}
}

func TestRequestIDMiddleware_HeaderReuse(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := GetRequestID(r.Context())
		w.Header().Set("X-Request-ID-Out", got)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "custom-request-id-12345678901234")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "custom-request-id-12345678901234", rec.Header().Get("X-Request-ID"))
	assert.Equal(t, "custom-request-id-12345678901234", rec.Header().Get("X-Request-ID-Out"))
}

func TestRequestIDMiddleware_GeneratedWhenMissing(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := GetRequestID(r.Context())
		w.Header().Set("X-Request-ID-Out", got)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	out := rec.Header().Get("X-Request-ID-Out")
	assert.NotEmpty(t, out)
	assert.Len(t, out, 32)
	assert.Equal(t, out, rec.Header().Get("X-Request-ID"))
}

func TestGetRequestID_EmptyWhenMissing(t *testing.T) {
	assert.Empty(t, GetRequestID(context.Background()))
}

func TestGlobalErrorHandler_PanicRecoveryReturns500(t *testing.T) {
	panicHandler := GlobalErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	panicHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "INTERNAL_ERROR", body["code"])
	assert.Equal(t, "Internal server error", body["message"])
}

func TestGlobalErrorHandler_PanicRecoveryIncludesRequestID(t *testing.T) {
	inner := GlobalErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	chain := RequestIDMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "abcdef1234567890abcdef1234567890")
	rec := httptest.NewRecorder()

	chain.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "abcdef1234567890abcdef1234567890", body["request_id"])
}

func TestGlobalErrorHandler_NoPanicPassThrough(t *testing.T) {
	handler := GlobalErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		handler.ServeHTTP(rec, req)
	})
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestGlobalErrorHandler_PanicValueTypes(t *testing.T) {
	testCases := []struct {
		name string
		fn   func()
	}{
		{"string panic", func() { panic("error string") }},
		{"int panic", func() { panic(42) }},
		{"error panic", func() { panic(struct{}{}) }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := GlobalErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tc.fn()
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()

			assert.NotPanics(t, func() {
				handler.ServeHTTP(rec, req)
			})
			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.True(t, strings.HasPrefix(rec.Body.String(), "{"))
		})
	}
}