package tenantctx

import (
	"context"
	"errors"

	"meteorx/internal/common/contextx"

	"gorm.io/gorm"
)

var (
	ErrTenantNotFound     = errors.New("tenant context not found")
	ErrTenantAccessDenied = errors.New("cross-tenant access denied")
)

type TenantInfo struct {
	ID       string
	UserID   string
	IsMaster bool
	Roles    []string
}

func From(ctx context.Context) (*TenantInfo, error) {
	tenantID := contextx.GetTenantID(ctx)
	if tenantID == "" {
		return nil, ErrTenantNotFound
	}

	userID := contextx.GetUserID(ctx)
	roles := contextx.GetRoles(ctx)

	isMaster := tenantID == contextx.SystemTenantID

	return &TenantInfo{
		ID:       tenantID,
		UserID:   userID,
		IsMaster: isMaster,
		Roles:    roles,
	}, nil
}

func RequireTenant(ctx context.Context) (*TenantInfo, error) {
	info, err := From(ctx)
	if err != nil {
		return nil, err
	}
	if info.ID == "" {
		return nil, ErrTenantNotFound
	}
	return info, nil
}

func CanAccessTenant(ctx context.Context, targetTenantID string) bool {
	info, err := From(ctx)
	if err != nil {
		return false
	}
	if info.IsMaster {
		return true
	}
	return info.ID == targetTenantID
}

func FilterQuery(ctx context.Context, query *gorm.DB, column string) *gorm.DB {
	info, err := From(ctx)
	if err != nil {
		return query.Where("1 = 0")
	}
	if info.IsMaster {
		return query
	}
	return query.Where(column+" = ?", info.ID)
}