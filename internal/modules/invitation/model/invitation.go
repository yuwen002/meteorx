// Package model 定义邀请模块的领域模型。
package model

import "time"

// 邀请状态常量
const (
	// InvitationStatusPending 待处理，等待被邀请人接受
	InvitationStatusPending = "pending"
	// InvitationStatusAccepted 已接受，被邀请人已注册并加入租户
	InvitationStatusAccepted = "accepted"
	// InvitationStatusCancelled 已取消，邀请人主动取消
	InvitationStatusCancelled = "cancelled"
	// InvitationStatusExpired 已过期，超过有效期未接受
	InvitationStatusExpired = "expired"
)

// Invitation 租户成员邀请领域模型。
// 记录管理员邀请外部用户加入租户的完整生命周期。
type Invitation struct {
	ID          string     // 邀请唯一标识
	TenantID    string     // 目标租户 ID
	Email       string     // 被邀请人邮箱
	Token       string     // 邀请令牌（UUID），用于邮件链接和接受验证
	RoleIDs     string     // 分配的角色 ID 列表（JSON 数组字符串）
	Status      string     // 当前状态：pending / accepted / cancelled / expired
	InvitedBy   string     // 邀请人用户 ID
	ExpiresAt   time.Time  // 过期时间（创建后 7 天）
	AcceptedAt  *time.Time // 接受时间（仅 accepted 状态有值）
	CreatedAt   time.Time  // 创建时间
	UpdatedAt   time.Time  // 更新时间
}