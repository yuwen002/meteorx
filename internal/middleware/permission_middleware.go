package middleware

import (
	"context"
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"net/http"
)

// PermissionChecker 权限检查器接口
type PermissionChecker interface {
	GetRolePermissionCodes(ctx context.Context, roleID string) ([]string, error)
}

// RequirePermission 验证当前用户是否具有指定权限
func RequirePermission(checker PermissionChecker, permissionCode string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := contextx.GetRole(r.Context())
			if role == "" {
				response.Fail(w, http.StatusForbidden, "权限不足")
				return
			}

			// 超级管理员直接放行
			if role == "superadmin" {
				next.ServeHTTP(w, r)
				return
			}

			// 查询该角色拥有的权限
			codes, err := checker.GetRolePermissionCodes(r.Context(), role)
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
