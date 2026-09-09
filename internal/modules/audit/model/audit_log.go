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

// RiskLevel 风险等级
const (
	RiskLow      = "low"      // 低风险
	RiskMedium   = "medium"   // 中风险
	RiskHigh     = "high"     // 高风险
	RiskCritical = "critical" // 严重风险
)

// SessionSummary 会话摘要
type SessionSummary struct {
	SessionID string
	UserID    string
	Username  string
	TotalOps  int64
	StartTime time.Time
	EndTime   time.Time
	Duration  int64
}

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
	IPLocation   string    // IP地理位置
	UserAgent    string    // 用户代理
	DeviceInfo   string    // 设备信息
	Duration     int64     // 请求耗时（毫秒）
	SessionID    string    // 会话ID
	RequestID    string    // 请求ID
	TraceID      string    // 链路追踪ID
	Referer      string    // 来源页面
	RiskLevel    string    // 风险等级（low/medium/high/critical）
	Tags         string    // 标签（JSON数组）
	CreatedAt    time.Time // 创建时间
}

type AuditLogStats struct {
	TotalCount  int64            `json:"total_count"`
	TodayCount  int64            `json:"today_count"`
	ActionStats map[string]int64 `json:"action_stats"`
	ModuleStats map[string]int64 `json:"module_stats"`
	ResultStats map[string]int64 `json:"result_stats"`
}

type AuditTrendPoint struct {
	Date    string `json:"date"`
	Count   int64  `json:"count"`
	Success int64  `json:"success"`
	Failure int64  `json:"failure"`
}

type AuditDashboardData struct {
	TotalCount  int64             `json:"total_count"`
	TodayCount  int64             `json:"today_count"`
	ActionStats map[string]int64  `json:"action_stats"`
	ModuleStats map[string]int64  `json:"module_stats"`
	ResultStats map[string]int64  `json:"result_stats"`
	Trend       []AuditTrendPoint `json:"trend"`
	TopModules  []ModuleCount     `json:"top_modules"`
}

type ModuleCount struct {
	Module string `json:"module"`
	Count  int64  `json:"count"`
}

type AuditLogQuery struct {
	UserID    string `form:"user_id"`
	Username  string `form:"username"`
	TenantID  string `form:"tenant_id"`
	Module    string `form:"module"`
	Action    string `form:"action"`
	Resource  string `form:"resource"`
	Result    string `form:"result"`
	Keyword   string `form:"keyword"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type TrendQuery struct {
	TenantID  string `form:"tenant_id"`
	Days      int    `form:"days"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
}

// UserActivityStat 用户活跃度统计
type UserActivityStat struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Count    int64  `json:"count"`
	Failures int64  `json:"failures"`
}

// AnomalyLog 异常日志记录
type AnomalyLog struct {
	UserID        string    `json:"user_id"`
	Username      string    `json:"username"`
	AnomalyType   string    `json:"anomaly_type"` // high_failure_rate, geo_anomaly, brute_force
	FailureCount  int64     `json:"failure_count"`
	TotalCount    int64     `json:"total_count"`
	WindowMinutes int       `json:"window_minutes"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	RiskLevel     string    `json:"risk_level"`
	Details       string    `json:"details"`
}

// AnomalyType 异常类型常量
const (
	AnomalyTypeHighFailure = "high_failure_rate" // 高频失败
	AnomalyTypeGeoAnomaly  = "geo_anomaly"       // 异地登录异常
	AnomalyTypeBruteForce  = "brute_force"        // 暴力破解尝试
)