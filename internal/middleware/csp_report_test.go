package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ================= CSPReport =================

func TestCSPReport_ValidReport_ReturnsNoContent(t *testing.T) {
	body := `{"csp-report":{"document-uri":"http://localhost/","blocked-uri":"http://evil.example/x.js","violated-directive":"script-src 'self'","effective-directive":"script-src","status-code":200}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/csp-report", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/csp-report")
	rr := httptest.NewRecorder()

	CSPReport().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
}

func TestCSPReport_MalformedJSON_StillNoContent(t *testing.T) {
	// 非法 JSON 必须静默吞掉并回 204，绝不把错误回吐给浏览器
	req := httptest.NewRequest(http.MethodPost, "/api/v1/csp-report", strings.NewReader("{not-json"))
	rr := httptest.NewRecorder()

	CSPReport().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
}

func TestCSPReport_EmptyPayload_StillNoContent(t *testing.T) {
	// 无 blocked-uri 与 violated-directive 视为无效上报，仍回 204
	req := httptest.NewRequest(http.MethodPost, "/api/v1/csp-report", strings.NewReader(`{"csp-report":{}}`))
	rr := httptest.NewRecorder()

	CSPReport().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
}

func TestCSPReport_OversizedBody_CappedAndNoContent(t *testing.T) {
	// 超大 body 应被 LimitReader 截断处理，不 panic、仍回 204
	huge := `{"csp-report":{"blocked-uri":"` + strings.Repeat("a", cspReportMaxBody+1024) + `"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/csp-report", strings.NewReader(huge))
	rr := httptest.NewRecorder()

	CSPReport().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
}

func TestCSPReport_NilBody_NoContent(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/csp-report", nil)
	req.Body = nil
	rr := httptest.NewRecorder()

	CSPReport().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
}

func TestCSPPreview(t *testing.T) {
	if got := cspPreview("short", 10); got != "short" {
		t.Fatalf("short 不应被截断，实际 %q", got)
	}
	if got := cspPreview("abcdefghij", 4); got != "abcd..." {
		t.Fatalf("截断结果 = %q, want abcd...", got)
	}
}
