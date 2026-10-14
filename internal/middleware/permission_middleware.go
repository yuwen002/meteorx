package middleware

import (
	"context"
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"net/http"
)

// PermissionChecker 权限检查器接口
type PermissionChecker interface {
	GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error)
}

// RequirePermission 验证当前用户是否具有指定权限
func RequirePermission(checker PermissionChecker, permissionCode string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := contextx.GetUserID(r.Context())
			granted, err := permissionGranted(r.Context(), checker, userID, permissionCode)
			if err != nil {
				response.Fail(w, http.StatusInternalServerError, "权限检查失败")
				return
			}
			if granted {
				next.ServeHTTP(w, r)
				return
			}
			if userID == "" {
				response.Fail(w, http.StatusForbidden, "权限不足")
				return
			}
			response.Fail(w, http.StatusForbidden, "权限不足，缺少权限: "+permissionCode)
		})
	}
}
