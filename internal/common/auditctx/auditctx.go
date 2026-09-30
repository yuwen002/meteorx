// Package auditctx 提供审计上下文工具，用于在请求处理链路中传递审计相关信息。
package auditctx

import (
	"context"
	"time"
)

// Action 描述一次待记录的审计操作，包含主体、资源、请求上下文与结果等信息。
type Action struct {
	Module     string
	Action     string
	Resource   string
	ResourceID string
	UserID     string
	Username   string
	TenantID   string
	ClientIP   string
	IPLocation string
	UserAgent  string
	DeviceInfo string
	Before     any
	After      any
	Result     string
	StatusCode int
	ErrorMsg   string
	Duration   int64
	SessionID  string
	RequestID  string
	TraceID    string
	Referer    string
	RiskLevel  string
	Tags       []string
	CreatedAt  time.Time
}

type contextKey string

const auditKey contextKey = "audit_action"

// WithAction 将审计操作存入上下文，供下游中间件读取。
func WithAction(ctx context.Context, action *Action) context.Context {
	return context.WithValue(ctx, auditKey, action)
}

// GetAction 从上下文读取当前审计操作，不存在时返回 nil。
func GetAction(ctx context.Context) *Action {
	action, _ := ctx.Value(auditKey).(*Action)
	return action
}

// NewAction 创建一个新的审计操作，默认结果为 success 并记录当前时间。
func NewAction(module, action, resource, resourceID string) *Action {
	return &Action{
		Module:     module,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Result:     "success",
		CreatedAt:  time.Now(),
	}
}

// WithUser 链式设置操作用户信息。
func (a *Action) WithUser(userID, username, tenantID string) *Action {
	a.UserID = userID
	a.Username = username
	a.TenantID = tenantID
	return a
}

// WithRequest 链式设置客户端 IP、User-Agent 与响应状态码。
func (a *Action) WithRequest(ip, userAgent string, statusCode int) *Action {
	a.ClientIP = ip
	a.UserAgent = userAgent
	a.StatusCode = statusCode
	return a
}

// WithIPLocation 链式设置 IP 归属地。
func (a *Action) WithIPLocation(location string) *Action {
	a.IPLocation = location
	return a
}

// WithDeviceInfo 链式设置设备信息。
func (a *Action) WithDeviceInfo(info string) *Action {
	a.DeviceInfo = info
	return a
}

// WithSession 链式设置会话 ID。
func (a *Action) WithSession(sessionID string) *Action {
	a.SessionID = sessionID
	return a
}

// WithRequestID 链式设置请求 ID。
func (a *Action) WithRequestID(requestID string) *Action {
	a.RequestID = requestID
	return a
}

// WithTraceID 链式设置链路追踪 ID。
func (a *Action) WithTraceID(traceID string) *Action {
	a.TraceID = traceID
	return a
}

// WithReferer 链式设置来源页 Referer。
func (a *Action) WithReferer(referer string) *Action {
	a.Referer = referer
	return a
}

// WithRiskLevel 链式设置风险等级。
func (a *Action) WithRiskLevel(level string) *Action {
	a.RiskLevel = level
	return a
}

// WithTags 链式设置自定义标签。
func (a *Action) WithTags(tags []string) *Action {
	a.Tags = tags
	return a
}

// WithBefore 链式设置变更前的快照数据。
func (a *Action) WithBefore(before any) *Action {
	a.Before = before
	return a
}

// WithAfter 链式设置变更后的快照数据。
func (a *Action) WithAfter(after any) *Action {
	a.After = after
	return a
}

// Failed 标记操作失败并记录错误消息。
func (a *Action) Failed(err error) *Action {
	a.Result = "failure"
	if err != nil {
		a.ErrorMsg = err.Error()
	}
	return a
}

// Succeeded 标记操作成功。
func (a *Action) Succeeded() *Action {
	a.Result = "success"
	return a
}

// WithDuration 链式设置操作耗时（毫秒）。
func (a *Action) WithDuration(d time.Duration) *Action {
	a.Duration = d.Milliseconds()
	return a
}
