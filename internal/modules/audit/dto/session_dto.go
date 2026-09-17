package dto

// SessionSummaryResp 会话摘要列表（前端会话表格使用）
type SessionSummaryResp struct {
	SessionID     string `json:"session_id"`
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	TotalRequests int64  `json:"total_requests"`
	SuccessCount  int64  `json:"success_count"`
	FailureCount  int64  `json:"failure_count"`
	AvgDuration   int64  `json:"avg_duration"`
	FirstRequest  string `json:"first_request"`
	LastRequest   string `json:"last_request"`
	MaxRisk       string `json:"max_risk"`
}

// SessionAnalysisResp 会话详情响应（前端会话详情弹窗使用）
type SessionAnalysisResp struct {
	SessionID       string           `json:"session_id"`
	UserID          string           `json:"user_id"`
	Username        string           `json:"username"`
	TotalRequests   int64            `json:"total_requests"`
	SuccessCount    int64            `json:"success_count"`
	FailureCount    int64            `json:"failure_count"`
	AvgDuration     int64            `json:"avg_duration"`
	FirstRequest    string           `json:"first_request"`
	LastRequest     string           `json:"last_request"`
	DurationMinutes int64            `json:"duration_minutes"`
	Logs            []*AuditLogResp  `json:"logs"`
	RiskStats       map[string]int64 `json:"risk_stats"`
}