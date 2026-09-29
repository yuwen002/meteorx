package model

import "time"

const (
	InvitationStatusPending  = "pending"
	InvitationStatusAccepted = "accepted"
	InvitationStatusCancelled = "cancelled"
	InvitationStatusExpired  = "expired"
)

type Invitation struct {
	ID          string
	TenantID    string
	Email       string
	Token       string
	RoleIDs     string
	Status      string
	InvitedBy   string
	ExpiresAt   time.Time
	AcceptedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}