package apperrors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvenienceConstructors(t *testing.T) {
	cases := []struct {
		name       string
		got        *AppError
		wantCode   ErrorCode
		wantStatus int
	}{
		{"ErrBadRequest", ErrBadRequest("参数错误"), ErrInvalidParam, http.StatusBadRequest},
		{"ErrNotFound", ErrNotFound("找不到"), ErrResourceNotFound, http.StatusNotFound},
		{"ErrForbidden", ErrForbidden("无权限"), ErrPermissionDenied, http.StatusForbidden},
		{"ErrUnauthorized", ErrUnauthorized("未登录"), ErrSessionExpired, http.StatusUnauthorized},
		{"NewConflict", NewConflict("冲突"), ErrConflict, http.StatusConflict},
		{"NewInternal", NewInternal("内部错误"), ErrInternal, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantCode, tc.got.Code)
			assert.Equal(t, tc.wantStatus, tc.got.StatusCode)
		})
	}
}

func TestAppError_ErrorStringPrefersMessage(t *testing.T) {
	e := New(ErrInternal, "boom")
	assert.Equal(t, "boom", e.Error())
	empty := New(ErrInternal, "")
	assert.Equal(t, string(ErrInternal), empty.Error())
}

func TestWrap_UnwrapsCause(t *testing.T) {
	cause := errors.New("db down")
	wrapped := Wrap(cause, ErrInternal, "查询失败")
	assert.Equal(t, http.StatusInternalServerError, wrapped.StatusCode)
	assert.True(t, errors.Is(wrapped, cause))
	assert.True(t, errors.Is(wrapped, wrapped))
}

func TestChaining_WithRequestIDAndDetails(t *testing.T) {
	e := ErrNotFound("不存在").
		WithRequestID("req-123").
		WithDetails(map[string]any{"id": "x"})
	assert.Equal(t, "req-123", e.RequestID)
	assert.Equal(t, map[string]any{"id": "x"}, e.Details)
	assert.Equal(t, "不存在", e.Error())
}

func TestNewWithStatus_OverridesMapping(t *testing.T) {
	e := NewWithStatus(ErrInvalidParam, "特殊", http.StatusUnprocessableEntity)
	assert.Equal(t, http.StatusUnprocessableEntity, e.StatusCode)
}

func TestIsAppError(t *testing.T) {
	_, ok := IsAppError(nil)
	assert.False(t, ok)

	_, ok = IsAppError(errors.New("plain"))
	assert.False(t, ok)

	appErr := ErrBadRequest("x")
	got, ok := IsAppError(appErr)
	assert.True(t, ok)
	assert.Same(t, appErr, got)
}

func TestFromError(t *testing.T) {
	// 已是 AppError 原样返回
	original := ErrForbidden("禁止")
	assert.Same(t, original, FromError(original))

	// 普通错误包装为 500
	converted := FromError(errors.New("oops"))
	assert.Equal(t, ErrInternal, converted.Code)
	assert.Equal(t, http.StatusInternalServerError, converted.StatusCode)
	assert.Equal(t, "oops", converted.Error())
}

func TestNewf_FormatsMessage(t *testing.T) {
	e := Newf(ErrInvalidParam, "非法参数: %s", "name")
	assert.Equal(t, "非法参数: name", e.Message)
}

func TestMapToHTTPStatus(t *testing.T) {
	cases := []struct {
		code ErrorCode
		want int
	}{
		{ErrSessionExpired, http.StatusUnauthorized},
		{ErrSessionNotFound, http.StatusUnauthorized},
		{ErrPermissionDenied, http.StatusForbidden},
		{ErrCrossTenantAccess, http.StatusForbidden},
		{ErrResourceNotFound, http.StatusNotFound},
		{ErrDocumentNotFound, http.StatusNotFound},
		{ErrPasswordTooWeak, http.StatusBadRequest},
		{ErrInvalidParam, http.StatusBadRequest},
		{ErrConflict, http.StatusConflict},
		{ErrWikiSpaceExist, http.StatusConflict},
		{ErrRateLimited, http.StatusTooManyRequests},
		{ErrTenantLimit, http.StatusPaymentRequired}, // 402：商用超量
		{ErrorCode("UNKNOWN_CODE"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, MapToHTTPStatus(tc.code), "code=%s", tc.code)
	}
}
