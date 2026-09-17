package apperrors

import (
	"net/http"
)

type ErrorCode string

const (
	// Auth errors (1xxxx)
	ErrAuthInvalidCode     ErrorCode = "AUTH_INVALID_CODE"
	ErrAuthLocked          ErrorCode = "AUTH_LOCKED"
	ErrSessionNotFound     ErrorCode = "SESSION_NOT_FOUND"
	ErrSessionExpired      ErrorCode = "SESSION_EXPIRED"
	ErrPasswordMismatch    ErrorCode = "PASSWORD_MISMATCH"
	ErrPasswordTooWeak     ErrorCode = "PASSWORD_TOO_WEAK"
	ErrPasswordAlreadyUsed ErrorCode = "PASSWORD_ALREADY_USED"
	ErrResetTokenInvalid   ErrorCode = "RESET_TOKEN_INVALID"
	ErrResetTokenExpired   ErrorCode = "RESET_TOKEN_EXPIRED"

	// Tenant errors (2xxxx)
	ErrTenantNotFound    ErrorCode = "TENANT_NOT_FOUND"
	ErrTenantDisabled    ErrorCode = "TENANT_DISABLED"
	ErrTenantLimit       ErrorCode = "TENANT_LIMIT_EXCEEDED"
	ErrCrossTenantAccess ErrorCode = "CROSS_TENANT_ACCESS_DENIED"

	// RBAC errors (3xxxx)
	ErrPermissionDenied   ErrorCode = "PERMISSION_DENIED"
	ErrRoleNotFound       ErrorCode = "ROLE_NOT_FOUND"
	ErrRoleAlreadyExists  ErrorCode = "ROLE_ALREADY_EXISTS"
	ErrPermissionNotFound ErrorCode = "PERMISSION_NOT_FOUND"

	// Wiki errors (4xxxx)
	ErrWikiSpaceNotFound   ErrorCode = "WIKI_SPACE_NOT_FOUND"
	ErrWikiNodeNotFound    ErrorCode = "WIKI_NODE_NOT_FOUND"
	ErrDocumentNotFound    ErrorCode = "WIKI_DOCUMENT_NOT_FOUND"
	ErrRevisionNotFound    ErrorCode = "WIKI_REVISION_NOT_FOUND"
	ErrWikiSpaceExist      ErrorCode = "WIKI_SPACE_ALREADY_EXISTS"
	ErrWikiNodeTypeInvalid ErrorCode = "WIKI_NODE_TYPE_INVALID"

	// Notification errors (5xxxx)
	ErrAnnouncementNotFound       ErrorCode = "ANNOUNCEMENT_NOT_FOUND"
	ErrAnnouncementAlreadyPublished ErrorCode = "ANNOUNCEMENT_ALREADY_PUBLISHED"
	ErrAnnouncementAccessDenied   ErrorCode = "ANNOUNCEMENT_ACCESS_DENIED"

	// Cancel request errors (6xxxx)
	ErrCancelRequestNotFound  ErrorCode = "CANCEL_REQUEST_NOT_FOUND"
	ErrCancelRequestDuplicate ErrorCode = "CANCEL_REQUEST_DUPLICATE"
	ErrCancelRequestNotPending ErrorCode = "CANCEL_REQUEST_NOT_PENDING"

	// Dashboard errors (7xxxx)
	ErrDashboardStatsUnavailable ErrorCode = "DASHBOARD_STATS_UNAVAILABLE"

	// Generic errors (9xxxx)
	ErrInvalidParam     ErrorCode = "INVALID_PARAM"
	ErrResourceNotFound ErrorCode = "RESOURCE_NOT_FOUND"
	ErrConflict         ErrorCode = "CONFLICT"
	ErrInternal         ErrorCode = "INTERNAL_ERROR"
	ErrRateLimited      ErrorCode = "RATE_LIMITED"
)

func MapToHTTPStatus(code ErrorCode) int {
	switch code {
	case ErrSessionExpired:
		return http.StatusUnauthorized
	case ErrAuthInvalidCode, ErrSessionNotFound:
		return http.StatusUnauthorized
	case ErrPermissionDenied, ErrCrossTenantAccess, ErrAnnouncementAccessDenied:
		return http.StatusForbidden
	case ErrResourceNotFound, ErrTenantNotFound, ErrRoleNotFound, ErrPermissionNotFound,
		ErrWikiSpaceNotFound, ErrWikiNodeNotFound, ErrDocumentNotFound, ErrRevisionNotFound,
		ErrAnnouncementNotFound, ErrCancelRequestNotFound:
		return http.StatusNotFound
	case ErrPasswordTooWeak, ErrPasswordAlreadyUsed, ErrInvalidParam, ErrResetTokenInvalid, ErrResetTokenExpired:
		return http.StatusBadRequest
	case ErrPasswordMismatch, ErrTenantDisabled, ErrCancelRequestNotPending, ErrCancelRequestDuplicate:
		return http.StatusBadRequest
	case ErrConflict, ErrWikiSpaceExist, ErrAnnouncementAlreadyPublished:
		return http.StatusConflict
	case ErrRateLimited:
		return http.StatusTooManyRequests
	case ErrTenantLimit:
		return http.StatusPaymentRequired
	default:
		return http.StatusInternalServerError
	}
}