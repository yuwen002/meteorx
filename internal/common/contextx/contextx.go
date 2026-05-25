package contextx

import "context"

// 定义私有类型，防止外部冲突
type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	TenantIDKey contextKey = "tenant_id"
	RoleKey     contextKey = "role"
)

// SetVars 存入核心身份信息
func SetVars(ctx context.Context, tenantID, userID, role string) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, userID)
	ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	return context.WithValue(ctx, RoleKey, role)
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
