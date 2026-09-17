package contextx

import "context"

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	TenantIDKey  contextKey = "tenant_id"
	RolesKey     contextKey = "roles"
	RequestIDKey contextKey = "request_id"
	TraceIDKey   contextKey = "trace_id"
)

// SetVars 存入核心身份信息
func SetVars(ctx context.Context, tenantID, userID string, roles []string) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, userID)
	ctx = context.WithValue(ctx, TenantIDKey, tenantID)
	return context.WithValue(ctx, RolesKey, roles)
}

// SetRequestID 存入请求ID（在middleware入口生成）
func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID 获取请求ID
func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(RequestIDKey).(string); ok {
		return val
	}
	return ""
}

// SetTraceID 存入链路追踪ID
func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID 获取链路追踪ID
func GetTraceID(ctx context.Context) string {
	if val, ok := ctx.Value(TraceIDKey).(string); ok {
		return val
	}
	return ""
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