package middleware

import (
	"context"
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"net/http"
)

type explicitPermKey struct{}

// WithExplicitPermission 将显式权限码存入请求上下文。
func WithExplicitPermission(next http.Handler, permissionCode string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), explicitPermKey{}, permissionCode)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetExplicitPermission 从上下文读取显式权限码，不存在时返回空串。
func GetExplicitPermission(ctx context.Context) string {
	code, _ := ctx.Value(explicitPermKey{}).(string)
	return code
}

// RequireExplicitPermission 返回校验显式权限码的中间件；超管直放行，未命中时回退到 defaultCode。
func RequireExplicitPermission(checker PermissionChecker, defaultCode string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if contextx.HasRole(r.Context(), "superadmin") {
				next.ServeHTTP(w, r)
				return
			}

			userID := contextx.GetUserID(r.Context())
			if userID == "" {
				response.Fail(w, http.StatusForbidden, "权限不足")
				return
			}

			explicitCode := GetExplicitPermission(r.Context())
			requiredCode := explicitCode
			if requiredCode == "" {
				requiredCode = defaultCode
			}

			if requiredCode == "" {
				next.ServeHTTP(w, r)
				return
			}

			codes, err := checker.GetUserPermissionCodes(r.Context(), userID)
			if err != nil {
				response.Fail(w, http.StatusInternalServerError, "权限检查失败")
				return
			}

			for _, code := range codes {
				if code == requiredCode {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Fail(w, http.StatusForbidden, "权限不足，缺少权限: "+requiredCode)
		})
	}
}
