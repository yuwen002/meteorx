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
			// 超级管理员直接放行
			if contextx.HasRole(r.Context(), "superadmin") {
				next.ServeHTTP(w, r)
				return
			}

			userID := contextx.GetUserID(r.Context())
			if userID == "" {
				response.Fail(w, http.StatusForbidden, "权限不足")
				return
			}

			// 查询该用户拥有的所有权限（合并多角色）
			codes, err := checker.GetUserPermissionCodes(r.Context(), userID)
			if err != nil {
				response.Fail(w, http.StatusInternalServerError, "权限检查失败")
				return
			}

			// 检查是否包含所需权限
			for _, code := range codes {
				if code == permissionCode {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Fail(w, http.StatusForbidden, "权限不足，缺少权限: "+permissionCode)
		})
	}
}
