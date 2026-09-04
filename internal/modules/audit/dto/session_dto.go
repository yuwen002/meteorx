package dto

// SessionAnalysisResp 会话分析响应
type SessionAnalysisResp struct {
	SessionID string           `json:"session_id"`
	UserID    string           `json:"user_id"`
	Username  string           `json:"username"`
	TotalOps  int64            `json:"total_ops"`
	Duration  int64            `json:"duration_ms"`
	StartTime string           `json:"start_time"`
	EndTime   string           `json:"end_time"`
	Logs      []*AuditLogResp  `json:"logs"`
	RiskStats map[string]int64 `json:"risk_stats"`
}

// SessionSummaryResp 会话摘要列表
type SessionSummaryResp struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	TotalOps  int64  `json:"total_ops"`
	Duration  int64  `json:"duration_ms"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	MaxRisk   string `json:"max_risk"`
}
