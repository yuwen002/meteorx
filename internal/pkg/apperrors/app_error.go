package apperrors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	StatusCode int       `json:"-"`
	RequestID  string    `json:"request_id,omitempty"`
	Details    any       `json:"details,omitempty"`
	err        error     `json:"-"`
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return string(e.Code)
}

func (e *AppError) Unwrap() error {
	return e.err
}

func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: MapToHTTPStatus(code),
	}
}

func NewWithStatus(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: MapToHTTPStatus(code),
		err:        err,
	}
}

func (e *AppError) WithRequestID(id string) *AppError {
	e.RequestID = id
	return e
}

func (e *AppError) WithDetails(details any) *AppError {
	e.Details = details
	return e
}

func ErrBadRequest(msg string) *AppError {
	return New(ErrInvalidParam, msg)
}

func ErrNotFound(msg string) *AppError {
	return New(ErrResourceNotFound, msg)
}

func ErrForbidden(msg string) *AppError {
	return New(ErrPermissionDenied, msg)
}

func ErrUnauthorized(msg string) *AppError {
	return NewWithStatus(ErrSessionExpired, msg, http.StatusUnauthorized)
}

func NewConflict(msg string) *AppError {
	return New(ErrConflict, msg)
}

func NewInternal(msg string) *AppError {
	return New(ErrInternal, msg)
}

func IsAppError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

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

func Newf(code ErrorCode, format string, args ...any) *AppError {
	return New(code, fmt.Sprintf(format, args...))
}