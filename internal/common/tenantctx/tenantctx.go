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

// Scope 返回一个已绑定当前租户边界的 gorm.DB。
// 平台超级管理员（IsMaster）不追加过滤，可跨租户操作；
// 普通租户强制追加 WHERE tenant_id = currentTenant，使该 DB 上的所有查询/更新/删除天然隔离。
// Repository 层应统一通过 Scope 获取 DB，而不是依赖开发者手动加过滤条件。
func Scope(ctx context.Context, db *gorm.DB, tenantColumn string) *gorm.DB {
	info, err := From(ctx)
	if err != nil {
		// 无法确定租户上下文时返回一个恒为 false 的查询，杜绝越权访问
		return db.Where("1 = 0")
	}
	if info.IsMaster {
		return db
	}
	return db.Where(tenantColumn+" = ?", info.ID)
}

// TenantID 返回当前上下文中的租户 ID（不校验是否存在）。
// 用于写入时填充子表的 tenant_id 列，保证与 Scope 读路径一致。
func TenantID(ctx context.Context) string {
	info, err := From(ctx)
	if err != nil {
		return ""
	}
	return info.ID
}
