package repository

import (
	"context"
	"meteorx/internal/modules/file/model"
)

// FileRepository 文件仓储接口
type FileRepository interface {
	// Create 创建文件记录
	Create(ctx context.Context, file *model.File) error
	
	// GetByID 根据ID获取文件（不含已软删除）
	GetByID(ctx context.Context, id string) (*model.File, error)

	// GetByIDUnscoped 根据ID获取文件（包含已软删除记录，用于恢复/永久删除场景）
	GetByIDUnscoped(ctx context.Context, id string) (*model.File, error)
	
	// GetByMD5 根据MD5获取文件（用于去重）
	GetByMD5(ctx context.Context, tenantID, md5 string) (*model.File, error)
	
	// ListByTenant 获取租户的文件列表
	ListByTenant(ctx context.Context, tenantID string, page, pageSize int) ([]*model.File, int64, error)
	
	// ListByUser 获取用户的文件列表
	ListByUser(ctx context.Context, tenantID, userID string, page, pageSize int) ([]*model.File, int64, error)
	
	// Update 更新文件信息
	Update(ctx context.Context, file *model.File) error
	
	// Delete 软删除文件
	Delete(ctx context.Context, id string) error
	
	// PermanentDelete 永久删除文件
	PermanentDelete(ctx context.Context, id string) error
	
	// GetDeletedList 获取已删除文件列表
	GetDeletedList(ctx context.Context, tenantID string, page, pageSize int) ([]*model.File, int64, error)
	
	// Restore 恢复已删除文件
	Restore(ctx context.Context, id string) error
}