// Package repository 提供数据导出任务的数据访问层。
package repository

import (
	"context"
	"errors"
	"time"

	"meteorx/internal/modules/export/model"

	"gorm.io/gorm"
)

// ExportJobPO 导出任务的 GORM 持久化模型
type ExportJobPO struct {
	ID          string     `gorm:"primaryKey;size:26;comment:'任务ID'"`
	TenantID    string     `gorm:"index;size:26;comment:'租户ID'"`
	UserID      string     `gorm:"index;size:26;comment:'创建用户ID'"`
	Resource    string     `gorm:"size:32;index;comment:'资源类型：user/task/audit_log/wiki_document'"`
	FiltersJSON string     `gorm:"type:text;comment:'筛选条件 JSON'"`
	Status      string     `gorm:"size:16;index;comment:'状态：pending/running/completed/failed'"`
	FilePath    string     `gorm:"size:512;comment:'生成的 CSV 文件路径'"`
	Rows        int64      `gorm:"comment:'导出总行数'"`
	ErrorMsg    string     `gorm:"size:1000;comment:'失败原因'"`
	CreatedAt   time.Time  `gorm:"autoCreateTime;index;comment:'创建时间'"`
	StartedAt   *time.Time `gorm:"comment:'开始执行时间'"`
	CompletedAt *time.Time `gorm:"comment:'完成时间'"`
	ExpiresAt   time.Time  `gorm:"index;comment:'文件过期时间'"`
	DeletedAt   gorm.DeletedAt
}

// TableName 表名
func (ExportJobPO) TableName() string { return "export_jobs" }

func (po ExportJobPO) toDomain() *model.ExportJob {
	return &model.ExportJob{
		ID:          po.ID,
		TenantID:    po.TenantID,
		UserID:      po.UserID,
		Resource:    po.Resource,
		FiltersJSON: po.FiltersJSON,
		Status:      po.Status,
		FilePath:    po.FilePath,
		Rows:        po.Rows,
		ErrorMsg:    po.ErrorMsg,
		CreatedAt:   po.CreatedAt,
		StartedAt:   po.StartedAt,
		CompletedAt: po.CompletedAt,
		ExpiresAt:   po.ExpiresAt,
	}
}

func fromDomain(m *model.ExportJob) *ExportJobPO {
	return &ExportJobPO{
		ID:          m.ID,
		TenantID:    m.TenantID,
		UserID:      m.UserID,
		Resource:    m.Resource,
		FiltersJSON: m.FiltersJSON,
		Status:      m.Status,
		FilePath:    m.FilePath,
		Rows:        m.Rows,
		ErrorMsg:    m.ErrorMsg,
		CreatedAt:   m.CreatedAt,
		StartedAt:   m.StartedAt,
		CompletedAt: m.CompletedAt,
		ExpiresAt:   m.ExpiresAt,
	}
}

// ExportRepository 导出任务仓储接口
type ExportRepository interface {
	Create(ctx context.Context, job *model.ExportJob) error
	GetByID(ctx context.Context, id string) (*model.ExportJob, error)
	ListByUser(ctx context.Context, tenantID, userID string, page, pageSize int, status, resource string) ([]*model.ExportJob, int64, error)
	// ListPending 拉取待执行任务（worker 消费），按创建时间升序
	ListPending(ctx context.Context, limit int) ([]*model.ExportJob, error)
	// MarkRunning 将任务置为 running 并写入开始时间
	MarkRunning(ctx context.Context, id string, startedAt time.Time) error
	// MarkCompleted 更新为 completed，附带文件路径与行数
	MarkCompleted(ctx context.Context, id, filePath string, rows int64, completedAt time.Time) error
	// MarkFailed 更新为 failed 并记录错误消息
	MarkFailed(ctx context.Context, id, errMsg string, completedAt time.Time) error
	// ListExpired 查询已过期待清理的任务
	ListExpired(ctx context.Context, now time.Time, limit int) ([]*model.ExportJob, error)
	Delete(ctx context.Context, id string) error
}

type exportRepository struct {
	db *gorm.DB
}

// NewExportRepository 创建导出任务仓储
func NewExportRepository(db *gorm.DB) ExportRepository {
	return &exportRepository{db: db}
}

func (r *exportRepository) Create(ctx context.Context, job *model.ExportJob) error {
	return r.db.WithContext(ctx).Create(fromDomain(job)).Error
}

func (r *exportRepository) GetByID(ctx context.Context, id string) (*model.ExportJob, error) {
	var po ExportJobPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExportJobNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *exportRepository) ListByUser(ctx context.Context, tenantID, userID string, page, pageSize int, status, resource string) ([]*model.ExportJob, int64, error) {
	var pos []*ExportJobPO
	var total int64

	query := r.db.WithContext(ctx).Model(&ExportJobPO{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*model.ExportJob, len(pos))
	for i, po := range pos {
		items[i] = po.toDomain()
	}
	return items, total, nil
}

func (r *exportRepository) ListPending(ctx context.Context, limit int) ([]*model.ExportJob, error) {
	var pos []*ExportJobPO
	if err := r.db.WithContext(ctx).
		Where("status = ?", model.ExportStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&pos).Error; err != nil {
		return nil, err
	}
	items := make([]*model.ExportJob, len(pos))
	for i, po := range pos {
		items[i] = po.toDomain()
	}
	return items, nil
}

func (r *exportRepository) MarkRunning(ctx context.Context, id string, startedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&ExportJobPO{}).
		Where("id = ? AND status = ?", id, model.ExportStatusPending).
		Updates(map[string]any{
			"status":     model.ExportStatusRunning,
			"started_at": startedAt,
		}).Error
}

func (r *exportRepository) MarkCompleted(ctx context.Context, id, filePath string, rows int64, completedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&ExportJobPO{}).
		Where("id = ? AND status = ?", id, model.ExportStatusRunning).
		Updates(map[string]any{
			"status":       model.ExportStatusCompleted,
			"file_path":    filePath,
			"rows":         rows,
			"completed_at": completedAt,
		}).Error
}

func (r *exportRepository) MarkFailed(ctx context.Context, id, errMsg string, completedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&ExportJobPO{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       model.ExportStatusFailed,
			"error_msg":    errMsg,
			"completed_at": completedAt,
		}).Error
}

func (r *exportRepository) ListExpired(ctx context.Context, now time.Time, limit int) ([]*model.ExportJob, error) {
	var pos []*ExportJobPO
	if err := r.db.WithContext(ctx).
		Where("expires_at < ?", now).
		Order("expires_at ASC").
		Limit(limit).
		Find(&pos).Error; err != nil {
		return nil, err
	}
	items := make([]*model.ExportJob, len(pos))
	for i, po := range pos {
		items[i] = po.toDomain()
	}
	return items, nil
}

func (r *exportRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&ExportJobPO{}, "id = ?", id).Error
}

// ErrExportJobNotFound 导出任务不存在
var ErrExportJobNotFound = errors.New("export job not found")

// AutoMigrate 迁移导出任务表
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&ExportJobPO{})
}
