// Package apperrors 定义应用级错误类型和错误码，支持 HTTP 状态码映射与错误链包装。
package apperrors

import (
	"fmt"
	"net/http"
)

// AppError 应用级错误，包含业务错误码、消息、HTTP 状态码及可选详情与被包裹原始错误。
type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	StatusCode int       `json:"-"`
	RequestID  string    `json:"request_id,omitempty"`
	Details    any       `json:"details,omitempty"`
	err        error     `json:"-"`
}

// Error 实现 error 接口，优先返回消息，其次返回错误码。
func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return string(e.Code)
}

// Unwrap 返回被包裹的原始错误，支持 errors.Is/As 链式判断。
func (e *AppError) Unwrap() error {
	return e.err
}

// New 根据错误码与消息创建 AppError，HTTP 状态码由错误码自动映射。
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: MapToHTTPStatus(code),
	}
}

// NewWithStatus 创建 AppError 并显式指定 HTTP 状态码。
func NewWithStatus(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Wrap 包裹原始错误，保留错误链以便向上定位。
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: MapToHTTPStatus(code),
		err:        err,
	}
}

// WithRequestID 链式设置请求 ID。
func (e *AppError) WithRequestID(id string) *AppError {
	e.RequestID = id
	return e
}

// WithDetails 链式设置附加详情数据。
func (e *AppError) WithDetails(details any) *AppError {
	e.Details = details
	return e
}

// ErrBadRequest 创建 400 参数错误。
func ErrBadRequest(msg string) *AppError {
	return New(ErrInvalidParam, msg)
}

// ErrNotFound 创建 404 资源不存在错误。
func ErrNotFound(msg string) *AppError {
	return New(ErrResourceNotFound, msg)
}

// ErrForbidden 创建 403 权限不足错误。
func ErrForbidden(msg string) *AppError {
	return New(ErrPermissionDenied, msg)
}

// ErrUnauthorized 创建 401 未认证/会话过期错误。
func ErrUnauthorized(msg string) *AppError {
	return NewWithStatus(ErrSessionExpired, msg, http.StatusUnauthorized)
}

// NewConflict 创建 409 资源冲突错误。
func NewConflict(msg string) *AppError {
	return New(ErrConflict, msg)
}

// NewInternal 创建 500 内部错误。
func NewInternal(msg string) *AppError {
	return New(ErrInternal, msg)
}

// IsAppError 判断错误是否为 AppError，是则返回具体实例。
func IsAppError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

// FromError 将任意 error 归一化为 AppError，非 AppError 时包装为内部错误。
func FromError(err error) *AppError {
	if appErr, ok := IsAppError(err); ok {
		return appErr
	}
	return &AppError{
		Code:       ErrInternal,
		Message:    err.Error(),
		StatusCode: http.StatusInternalServerError,
		err:        err,
	}
}

// Newf 按格式化参数创建 AppError。
func Newf(code ErrorCode, format string, args ...any) *AppError {
	return New(code, fmt.Sprintf(format, args...))
}
