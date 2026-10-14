// Package contextx 提供请求上下文存取工具，封装 UserID、TenantID、角色等信息在 context.Context 中的传递。
package contextx

import "context"

type contextKey string

const (
	UserIDKey       contextKey = "user_id"
	TenantIDKey     contextKey = "tenant_id"
	RolesKey        contextKey = "roles"
	AllowedPathsKey contextKey = "allowed_paths"
	ScopedPermsKey  contextKey = "scoped_perms"
	RequestIDKey    contextKey = "request_id"
	TraceIDKey      contextKey = "trace_id"
)

// TokenPrincipal API Token 验证后的身份与授权信息，供认证中间件注入请求上下文。
// Scoped 为 true 时，Permissions 为该令牌被严格收敛后的可用权限码集合
// （即「属主权限 ∩ 令牌 scopes」），下游权限中间件须据此判定并忽略超管旁路；
// Scoped 为 false 表示令牌未配置 scope，沿用属主的完整 RBAC 权限（向后兼容）。
type TokenPrincipal struct {
	UserID       string
	TenantID     string
	AllowedPaths []string
	Scoped       bool
	Permissions  []string
}

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

// GetAllowedPaths 获取 API Token 的可访问路径列表
// 返回 nil 表示不限制（默认权限，或为 JWT 认证）
func GetAllowedPaths(ctx context.Context) []string {
	if val, ok := ctx.Value(AllowedPathsKey).([]string); ok {
		return val
	}
	return nil
}

// SetScopedPermissions 将带 scope 限制的 API Token 收敛后的权限码集合存入上下文。
// 仅在令牌配置了非空 scope 时调用，用于下游权限中间件的严格判定。
func SetScopedPermissions(ctx context.Context, perms []string) context.Context {
	return context.WithValue(ctx, ScopedPermsKey, perms)
}

// GetScopedPermissions 读取 API Token 的 scope 收敛权限集合。
// 第二个返回值表示当前请求是否来自「带 scope 限制的 API Token」（严格模式）；
// 为 false 时集合无意义（JWT 或无 scope 的令牌，沿用属主完整权限）。
func GetScopedPermissions(ctx context.Context) ([]string, bool) {
	if val, ok := ctx.Value(ScopedPermsKey).([]string); ok {
		return val, true
	}
	return nil, false
}
