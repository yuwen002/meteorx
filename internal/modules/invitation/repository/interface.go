package repository

import (
	"context"
	"meteorx/internal/modules/invitation/model"
)

type InvitationRepository interface {
	Create(ctx context.Context, inv *model.Invitation) error
	GetByToken(ctx context.Context, token string) (*model.Invitation, error)
	GetByID(ctx context.Context, id string) (*model.Invitation, error)
	ListByTenant(ctx context.Context, tenantID string, page, pageSize int, keyword, status string) ([]*model.Invitation, int64, error)
	UpdateStatus(ctx context.Context, id, status string) error
	Delete(ctx context.Context, id string) error
	CountPendingByTenant(ctx context.Context, tenantID string) (int64, error)
	FindByEmailAndTenant(ctx context.Context, email, tenantID string) (*model.Invitation, error)
}