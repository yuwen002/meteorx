package contextx

import "context"

// 定义私有类型，防止外部冲突
type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	TenantIDKey contextKey = "tenant_id"
	RolesKey    contextKey = "roles"
)

// SetVars 存入核心身份信息
func SetVars(ctx context.Context, tenantID, userID string, roles []string) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, userID)
	ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	return context.WithValue(ctx, RolesKey, roles)
}

// GetTenantID 获取当前租户ID
func GetTenantID(ctx context.Context) string {
	if val, ok := ctx.Value(TenantIDKey).(string); ok {
		return val
	}
	return ""
}

// GetUserID 获取当前用户ID
func GetUserID(ctx context.Context) string {
	if val, ok := ctx.Value(UserIDKey).(string); ok {
		return val
	}
	return ""
}

// GetRoles 获取当前用户角色列表
func GetRoles(ctx context.Context) []string {
	if val, ok := ctx.Value(RolesKey).([]string); ok {
		return val
	}
	return []string{}
}

// HasRole 检查当前用户是否具有指定角色
func HasRole(ctx context.Context, requiredRole string) bool {
	for _, role := range GetRoles(ctx) {
		if role == requiredRole {
			return true
		}
	}
	return false
}
