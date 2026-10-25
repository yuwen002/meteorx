package repository

import (
	"context"

	"gorm.io/gorm"
)

// userDirectory 基于 users 表实现收件人展开。
// 只读取 id / status 两列，避免把用户模块的完整模型带进本模块。
type userDirectory struct {
	db *gorm.DB
}

// NewUserDirectory 创建收件人目录。
func NewUserDirectory(db *gorm.DB) UserDirectory {
	return &userDirectory{db: db}
}

// userRow 按租户展开用户 ID 的最小投影。
type userRow struct {
	ID string
}

// ListActiveUserIDs 返回租户内启用状态的用户 ID，按创建时间升序取前 limit 条。
func (d *userDirectory) ListActiveUserIDs(ctx context.Context, tenantID string, limit int) ([]string, error) {
	if tenantID == "" || limit <= 0 {
		return nil, nil
	}
	var rows []userRow
	err := d.db.WithContext(ctx).
		Table("users").
		Select("id").
		Where("tenant_id = ? AND status = ? AND deleted_at IS NULL", tenantID, userStatusEnabled).
		Order("created_at ASC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids, nil
}
