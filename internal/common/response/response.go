package response

import (
	"encoding/json"
	"net/http"

	"meteorx/internal/pkg/apperrors"
)

type Response struct {
	Data       any         `json:"data,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
	RequestID  string      `json:"request_id,omitempty"`
}

type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	Details   any    `json:"details,omitempty"`
}

type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

func WriteJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

func Success(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, Response{Data: data})
}

func SuccessWithPagination(w http.ResponseWriter, data any, page, pageSize int, total int64) {
	WriteJSON(w, http.StatusOK, Response{
		Data:       data,
		Pagination: &Pagination{Page: page, PageSize: pageSize, Total: total},
	})
}

func SuccessNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Fail(w http.ResponseWriter, httpStatus int, message string) {
	FailWithStatus(w, httpStatus, message)
}

func FailError(w http.ResponseWriter, err error) {
	appErr := apperrors.FromError(err)
	resp := ErrorResponse{
		Code:    string(appErr.Code),
		Message: appErr.Message,
		Details: appErr.Details,
	}
	WriteJSON(w, appErr.StatusCode, resp)
}

func JSON(w http.ResponseWriter, httpStatus int, code int, message string, data any) {
	WriteJSON(w, httpStatus, Response{
		Data: data,
	})
}

func FailWithData(w http.ResponseWriter, httpStatus int, message string, data any) {
	WriteJSON(w, httpStatus, Response{
		Data: data,
	})
}

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

func BadRequest(w http.ResponseWriter, message string) {
	FailError(w, apperrors.ErrBadRequest(message))
}

func NotFound(w http.ResponseWriter, message string) {
	FailError(w, apperrors.ErrNotFound(message))
}

func Forbidden(w http.ResponseWriter, message string) {
	FailError(w, apperrors.ErrForbidden(message))
}

func Unauthorized(w http.ResponseWriter, message string) {
	FailError(w, apperrors.ErrUnauthorized(message))
}

func InternalError(w http.ResponseWriter, message string) {
	FailError(w, apperrors.NewInternal(message))
}

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
