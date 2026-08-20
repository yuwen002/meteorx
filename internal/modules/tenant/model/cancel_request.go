package model

import "time"

// 注销申请状态
const (
	CancelRequestStatusPending   = 1 // 待审批
	CancelRequestStatusApproved  = 2 // 已通过（等待执行）
	CancelRequestStatusRejected  = 3 // 已驳回
	CancelRequestStatusCompleted = 4 // 已完成注销
)

// CancelRequest 租户注销申请领域模型
type CancelRequest struct {
	ID           string
	TenantID     string
	TenantName   string
	Reason       string
	Status       int
	ApproverID   string
	ReviewRemark string
	EffectiveAt  *time.Time // 计划执行时间（审批通过后生效）
	AppliedAt    time.Time
	ApprovedAt   *time.Time
	CompletedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
