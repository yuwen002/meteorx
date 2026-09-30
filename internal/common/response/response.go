// Package response 提供统一的 HTTP JSON 响应封装，包括成功、失败、分页等常用格式。
package response

import (
	"encoding/json"
	"net/http"

	"meteorx/internal/pkg/apperrors"
)

// Response 统一成功响应体，可选携带数据、分页与请求 ID。
type Response struct {
	Data       any         `json:"data,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
	RequestID  string      `json:"request_id,omitempty"`
}

// ErrorResponse 统一错误响应体，包含业务错误码、消息与可选详情。
type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	Details   any    `json:"details,omitempty"`
}

// Pagination 分页元信息（当前页、每页条数、总条数）。
type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// WriteJSON 以指定 HTTP 状态码将 v 序列化为 JSON 写入响应。
func WriteJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

// Success 返回 200 成功响应，携带数据。
func Success(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, Response{Data: data})
}

// SuccessWithPagination 返回 200 成功响应，并附带分页信息。
func SuccessWithPagination(w http.ResponseWriter, data any, page, pageSize int, total int64) {
	WriteJSON(w, http.StatusOK, Response{
		Data:       data,
		Pagination: &Pagination{Page: page, PageSize: pageSize, Total: total},
	})
}

// SuccessNoContent 返回 204 无内容响应。
func SuccessNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Fail 以指定 HTTP 状态码与消息返回失败响应。
func Fail(w http.ResponseWriter, httpStatus int, message string) {
	FailWithStatus(w, httpStatus, message)
}

// FailError 将 error 归一化为应用错误后返回失败响应。
func FailError(w http.ResponseWriter, err error) {
	appErr := apperrors.FromError(err)
	resp := ErrorResponse{
		Code:    string(appErr.Code),
		Message: appErr.Message,
		Details: appErr.Details,
	}
	WriteJSON(w, appErr.StatusCode, resp)
}

// JSON 返回自定义状态码的 JSON 响应。
func JSON(w http.ResponseWriter, httpStatus int, code int, message string, data any) {
	WriteJSON(w, httpStatus, Response{
		Data: data,
	})
}

// FailWithData 以指定状态码返回携带数据的失败响应。
func FailWithData(w http.ResponseWriter, httpStatus int, message string, data any) {
	WriteJSON(w, httpStatus, Response{
		Data: data,
	})
}

// FailWithRequestID 返回失败响应并携带请求 ID，便于链路排查。
func FailWithRequestID(w http.ResponseWriter, err error, requestID string) {
	appErr := apperrors.FromError(err)
	appErr.RequestID = requestID
	resp := ErrorResponse{
		Code:      string(appErr.Code),
		Message:   appErr.Message,
		RequestID: requestID,
		Details:   appErr.Details,
	}
	WriteJSON(w, appErr.StatusCode, resp)
}

// BadRequest 返回 400 请求参数错误。
func BadRequest(w http.ResponseWriter, message string) {
	FailError(w, apperrors.ErrBadRequest(message))
}

// NotFound 返回 404 资源不存在。
func NotFound(w http.ResponseWriter, message string) {
	FailError(w, apperrors.ErrNotFound(message))
}

// Forbidden 返回 403 无权访问。
func Forbidden(w http.ResponseWriter, message string) {
	FailError(w, apperrors.ErrForbidden(message))
}

// Unauthorized 返回 401 未认证。
func Unauthorized(w http.ResponseWriter, message string) {
	FailError(w, apperrors.ErrUnauthorized(message))
}

// InternalError 返回 500 服务器内部错误。
func InternalError(w http.ResponseWriter, message string) {
	FailError(w, apperrors.NewInternal(message))
}

// FailWithStatus 根据 HTTP 状态码映射业务错误码后返回失败响应。
func FailWithStatus(w http.ResponseWriter, httpStatus int, message string) {
	code := apperrors.ErrInternal
	switch httpStatus {
	case http.StatusBadRequest:
		code = apperrors.ErrInvalidParam
	case http.StatusUnauthorized:
		code = apperrors.ErrSessionExpired
	case http.StatusForbidden:
		code = apperrors.ErrPermissionDenied
	case http.StatusNotFound:
		code = apperrors.ErrResourceNotFound
	case http.StatusConflict:
		code = apperrors.ErrConflict
	default:
		code = apperrors.ErrInternal
	}
	FailError(w, apperrors.NewWithStatus(code, message, httpStatus))
}
