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
	UserAgent  string
	Before     any
	After      any
	Result     string
	StatusCode int
	ErrorMsg   string
	Duration   int64
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