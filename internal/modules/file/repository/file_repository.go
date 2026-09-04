package repository

import (
	"context"
	"meteorx/internal/modules/file/model"

	"gorm.io/gorm"
)

// FileRepositoryImpl 文件仓储实现
type FileRepositoryImpl struct {
	db *gorm.DB
}

// NewFileRepository 创建文件仓储实例
func NewFileRepository(db *gorm.DB) FileRepository {
	return &FileRepositoryImpl{db: db}
}

// Create 创建文件记录
func (r *FileRepositoryImpl) Create(ctx context.Context, file *model.File) error {
	return r.db.WithContext(ctx).Create(file).Error
}

// GetByID 根据ID获取文件
func (r *FileRepositoryImpl) GetByID(ctx context.Context, id string) (*model.File, error) {
	var file model.File
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// GetByIDUnscoped 根据ID获取文件（包含已软删除记录）
func (r *FileRepositoryImpl) GetByIDUnscoped(ctx context.Context, id string) (*model.File, error) {
	var file model.File
	err := r.db.WithContext(ctx).Unscoped().Where("id = ?", id).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// GetByMD5 根据MD5获取文件（用于去重）
func (r *FileRepositoryImpl) GetByMD5(ctx context.Context, tenantID, md5 string) (*model.File, error) {
	var file model.File
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND md5 = ? AND deleted_at IS NULL", tenantID, md5).
		First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// ListByTenant 获取租户的文件列表
func (r *FileRepositoryImpl) ListByTenant(ctx context.Context, tenantID string, page, pageSize int) ([]*model.File, int64, error) {
	var files []*model.File
	var total int64

	offset := (page - 1) * pageSize

	query := r.db.WithContext(ctx).Model(&model.File{}).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

// ListByUser 获取用户的文件列表
func (r *FileRepositoryImpl) ListByUser(ctx context.Context, tenantID, userID string, page, pageSize int) ([]*model.File, int64, error) {
	var files []*model.File
	var total int64

	offset := (page - 1) * pageSize

	query := r.db.WithContext(ctx).Model(&model.File{}).
		Where("tenant_id = ? AND user_id = ? AND deleted_at IS NULL", tenantID, userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

// Update 更新文件信息
func (r *FileRepositoryImpl) Update(ctx context.Context, file *model.File) error {
	return r.db.WithContext(ctx).Save(file).Error
}

// Delete 软删除文件
func (r *FileRepositoryImpl) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.File{}).Error
}

// PermanentDelete 永久删除文件
func (r *FileRepositoryImpl) PermanentDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&model.File{}).Error
}

// GetDeletedList 获取已删除文件列表
func (r *FileRepositoryImpl) GetDeletedList(ctx context.Context, tenantID string, page, pageSize int) ([]*model.File, int64, error) {
	var files []*model.File
	var total int64

	offset := (page - 1) * pageSize

	query := r.db.WithContext(ctx).Model(&model.File{}).Where("tenant_id = ? AND deleted_at IS NOT NULL", tenantID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("deleted_at DESC").Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

// Restore 恢复已删除文件
func (r *FileRepositoryImpl) Restore(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Unscoped().Model(&model.File{}).
		Where("id = ?", id).
		Update("deleted_at", nil).Error
}
