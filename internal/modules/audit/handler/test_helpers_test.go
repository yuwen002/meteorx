package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/common/contextx"
)

// doReq 发送携带指定角色上下文的请求
func doReq(t *testing.T, h http.Handler, method, path, body string, roles []string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req = req.WithContext(contextx.SetVars(req.Context(), "tenant-1", "user-1", roles))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}
