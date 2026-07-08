package model

import "time"

// ActionType 操作类型常量
const (
	ActionTypeCreate = "create" // 创建
	ActionTypeUpdate = "update" // 更新
	ActionTypeDelete = "delete" // 删除
	ActionTypeQuery  = "query"  // 查询
	ActionTypeLogin  = "login"  // 登录
	ActionTypeLogout = "logout" // 登出
	ActionTypeOther  = "other"  // 其他
)

// ResultStatus 操作结果状态
const (
	ResultSuccess = "success" // 成功
	ResultFailure = "failure" // 失败
)

// AuditLog 审计日志领域模型
type AuditLog struct {
	ID           string    // 日志ID
	UserID       string    // 操作用户ID
	Username     string    // 操作用户名（冗余，便于查询）
	TenantID     string    // 所属租户ID
	Module       string    // 操作模块（如：user, tenant, rbac, auth）
	Action       string    // 操作类型（create/update/delete/query/login/logout/other）
	Resource     string    // 操作资源（如：/api/v1/users）
	ResourceID   string    // 被操作资源ID（如：用户ID）
	Method       string    // HTTP 方法（GET/POST/PUT/DELETE）
	Path         string    // 请求路径
	RequestBody  string    // 请求参数（JSON，敏感信息脱敏）
	ResponseBody string    // 响应结果摘要（JSON，敏感信息脱敏）
	StatusCode   int       // HTTP 状态码
	Result       string    // 操作结果（success/failure）
	ErrorMessage string    // 错误信息
	ClientIP     string    // 客户端IP
	UserAgent    string    // 用户代理
	Duration     int64     // 请求耗时（毫秒）
	CreatedAt    time.Time // 创建时间
}