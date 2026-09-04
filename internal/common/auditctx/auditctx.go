package auditctx

import (
	"context"
	"time"
)

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

func WithAction(ctx context.Context, action *Action) context.Context {
	return context.WithValue(ctx, auditKey, action)
}

func GetAction(ctx context.Context) *Action {
	action, _ := ctx.Value(auditKey).(*Action)
	return action
}

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

func (a *Action) WithUser(userID, username, tenantID string) *Action {
	a.UserID = userID
	a.Username = username
	a.TenantID = tenantID
	return a
}

func (a *Action) WithRequest(ip, userAgent string, statusCode int) *Action {
	a.ClientIP = ip
	a.UserAgent = userAgent
	a.StatusCode = statusCode
	return a
}

func (a *Action) WithIPLocation(location string) *Action {
	a.IPLocation = location
	return a
}

func (a *Action) WithDeviceInfo(info string) *Action {
	a.DeviceInfo = info
	return a
}

func (a *Action) WithSession(sessionID string) *Action {
	a.SessionID = sessionID
	return a
}

func (a *Action) WithRequestID(requestID string) *Action {
	a.RequestID = requestID
	return a
}

func (a *Action) WithTraceID(traceID string) *Action {
	a.TraceID = traceID
	return a
}

func (a *Action) WithReferer(referer string) *Action {
	a.Referer = referer
	return a
}

func (a *Action) WithRiskLevel(level string) *Action {
	a.RiskLevel = level
	return a
}

func (a *Action) WithTags(tags []string) *Action {
	a.Tags = tags
	return a
}

func (a *Action) WithBefore(before any) *Action {
	a.Before = before
	return a
}

func (a *Action) WithAfter(after any) *Action {
	a.After = after
	return a
}

func (a *Action) Failed(err error) *Action {
	a.Result = "failure"
	if err != nil {
		a.ErrorMsg = err.Error()
	}
	return a
}

func (a *Action) Succeeded() *Action {
	a.Result = "success"
	return a
}

func (a *Action) WithDuration(d time.Duration) *Action {
	a.Duration = d.Milliseconds()
	return a
}
