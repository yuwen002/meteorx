package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/common/response"
	"meteorx/internal/pkg/apperrors"
)

func decode(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
}

func TestWriteJSON_ContentTypeAndStatus(t *testing.T) {
	w := httptest.NewRecorder()

	response.WriteJSON(w, http.StatusCreated, map[string]string{"ok": "1"})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	if got := w.Body.String(); got != "{\"ok\":\"1\"}\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	response.Success(w, struct{ Name string `json:"name"` }{Name: "Acme"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var out struct {
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	decode(t, w, &out)
	if out.Data.Name != "Acme" {
		t.Fatalf("data = %+v", out.Data)
	}
}

func TestSuccess_NilData_OmitsData(t *testing.T) {
	w := httptest.NewRecorder()
	response.Success(w, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var out map[string]json.RawMessage
	decode(t, w, &out)
	if _, ok := out["data"]; ok {
		t.Fatalf("nil data 不应出现在响应中: %s", w.Body.String())
	}
}

func TestSuccessWithPagination(t *testing.T) {
	w := httptest.NewRecorder()
	response.SuccessWithPagination(w, []string{"a"}, 2, 10, 21)

	var out struct {
		Data []string `json:"data"`
		Pagination struct {
			Page     int   `json:"page"`
			PageSize int   `json:"page_size"`
			Total    int64 `json:"total"`
		} `json:"pagination"`
	}
	decode(t, w, &out)
	if out.Pagination.Page != 2 || out.Pagination.PageSize != 10 || out.Pagination.Total != 21 {
		t.Fatalf("pagination = %+v", out.Pagination)
	}
	if len(out.Data) != 1 || out.Data[0] != "a" {
		t.Fatalf("data = %+v", out.Data)
	}
}

func TestSuccessNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	response.SuccessNoContent(w)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d", w.Code)
	}
}

// Fail 状态码与 code 映射表
func TestFail_StatusToCodeMapping(t *testing.T) {
	cases := []struct {
		httpStatus int
		code       apperrors.ErrorCode
	}{
		{http.StatusBadRequest, apperrors.ErrInvalidParam},
		{http.StatusUnauthorized, apperrors.ErrSessionExpired},
		{http.StatusForbidden, apperrors.ErrPermissionDenied},
		{http.StatusNotFound, apperrors.ErrResourceNotFound},
		{http.StatusConflict, apperrors.ErrConflict},
		{http.StatusTooManyRequests, apperrors.ErrInternal}, // 429 未显式映射 → 默认 INTERNAL_ERROR
		{http.StatusInternalServerError, apperrors.ErrInternal},
		{http.StatusServiceUnavailable, apperrors.ErrInternal},
	}

	for _, c := range cases {
		w := httptest.NewRecorder()
		response.Fail(w, c.httpStatus, "出错了")

		if w.Code != c.httpStatus {
			t.Fatalf("status %d: got %d", c.httpStatus, w.Code)
		}
		var out struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		decode(t, w, &out)
		if out.Code != string(c.code) {
			t.Fatalf("status %d: code = %q, want %q", c.httpStatus, out.Code, c.code)
		}
		if out.Message != "出错了" {
			t.Fatalf("status %d: message = %q", c.httpStatus, out.Message)
		}
	}
}

func TestFail_BadRequest_MessageAndCode(t *testing.T) {
	w := httptest.NewRecorder()
	response.Fail(w, http.StatusBadRequest, "用户名不合法")

	var out struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	decode(t, w, &out)
	if out.Code != "INVALID_PARAM" || out.Message != "用户名不合法" {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestFailError_PassesStructuredFields(t *testing.T) {
	w := httptest.NewRecorder()
	appErr := apperrors.NewWithStatus(apperrors.ErrConflict, "域名冲突", http.StatusConflict)
	appErr.Details = map[string]any{"domain": "acme"}

	response.FailError(w, appErr)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d", w.Code)
	}
	var out struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	}
	decode(t, w, &out)
	if out.Code != string(apperrors.ErrConflict) || out.Message != "域名冲突" {
		t.Fatalf("body = %s", w.Body.String())
	}
	if _, ok := out.Details["domain"]; !ok {
		t.Fatalf("details = %v", out.Details)
	}
}

func TestFailError_FromPlainError_DefaultsInternal(t *testing.T) {
	w := httptest.NewRecorder()
	response.FailError(w, http.ErrBodyNotAllowed)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestFailWithRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	response.FailWithRequestID(w, apperrors.ErrForbidden("无权访问"), "req-123")

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d", w.Code)
	}
	var out struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	}
	decode(t, w, &out)
	if out.Code != "PERMISSION_DENIED" || out.RequestID != "req-123" {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestConvenienceErrorHelpers(t *testing.T) {
	cases := []struct {
		name     string
		call     func(*httptest.ResponseRecorder)
		status   int
		code     string
		message  string
	}{
		{"BadRequest", func(w *httptest.ResponseRecorder) { response.BadRequest(w, "参数错误") }, http.StatusBadRequest, "INVALID_PARAM", "参数错误"},
		{"NotFound", func(w *httptest.ResponseRecorder) { response.NotFound(w, "找不到") }, http.StatusNotFound, "RESOURCE_NOT_FOUND", "找不到"},
		{"Forbidden", func(w *httptest.ResponseRecorder) { response.Forbidden(w, "没权限") }, http.StatusForbidden, "PERMISSION_DENIED", "没权限"},
		{"Unauthorized", func(w *httptest.ResponseRecorder) { response.Unauthorized(w, "未登录") }, http.StatusUnauthorized, "SESSION_EXPIRED", "未登录"},
		{"InternalError", func(w *httptest.ResponseRecorder) { response.InternalError(w, "内部错误") }, http.StatusInternalServerError, "INTERNAL_ERROR", "内部错误"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c.call(w)
			if w.Code != c.status {
				t.Fatalf("status = %d", w.Code)
			}
			var out struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			decode(t, w, &out)
			if out.Code != c.code || out.Message != c.message {
				t.Fatalf("body = %s", w.Body.String())
			}
		})
	}
}
