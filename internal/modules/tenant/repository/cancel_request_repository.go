package repository

import (
	"context"
	"time"

	"meteorx/internal/modules/tenant/model"

	"gorm.io/gorm"
)

// CancelRequestPO 注销申请的 GORM 模型
type CancelRequestPO struct {
	ID           string         `gorm:"primaryKey;size:26;comment:'申请ID'"`
	TenantID     string         `gorm:"index;size:26;comment:'租户ID'"`
	TenantName   string         `gorm:"size:100;comment:'租户名称（冗余展示）'"`
	Reason       string         `gorm:"size:1000;comment:'注销原因'"`
	Status       int            `gorm:"index;comment:'状态：1待审批 2已通过 3已驳回 4已完成'"`
	ApproverID   string         `gorm:"size:26;comment:'审批人ID'"`
	ReviewRemark string         `gorm:"size:500;comment:'审批备注'"`
	EffectiveAt  *time.Time     `gorm:"index;comment:'计划执行时间'"`
	AppliedAt    time.Time      `gorm:"comment:'申请时间'"`
	ApprovedAt   *time.Time     `gorm:"comment:'审批时间'"`
	CompletedAt  *time.Time     `gorm:"comment:'完成时间'"`
	CreatedAt    time.Time      `gorm:"autoCreateTime;comment:'创建时间'"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime;comment:'更新时间'"`
	DeletedAt    gorm.DeletedAt `gorm:"index;comment:'软删除时间'"`
}

// TableName 表名
func (CancelRequestPO) TableName() string {
	return "cancel_requests"
}

// toDomain 转换为领域模型
func (po CancelRequestPO) toDomain() *model.CancelRequest {
	return &model.CancelRequest{
		ID:           po.ID,
		TenantID:     po.TenantID,
		TenantName:   po.TenantName,
		Reason:       po.Reason,
		Status:       po.Status,
		ApproverID:   po.ApproverID,
		ReviewRemark: po.ReviewRemark,
		EffectiveAt:  po.EffectiveAt,
		AppliedAt:    po.AppliedAt,
		ApprovedAt:   po.ApprovedAt,
		CompletedAt:  po.CompletedAt,
		CreatedAt:    po.CreatedAt,
		UpdatedAt:    po.UpdatedAt,
	}
}

// cancelRequestFromDomain 从领域模型转换
func cancelRequestFromDomain(c *model.CancelRequest) *CancelRequestPO {
	return &CancelRequestPO{
		ID:           c.ID,
		TenantID:     c.TenantID,
		TenantName:   c.TenantName,
		Reason:       c.Reason,
		Status:       c.Status,
		ApproverID:   c.ApproverID,
		ReviewRemark: c.ReviewRemark,
		EffectiveAt:  c.EffectiveAt,
		AppliedAt:    c.AppliedAt,
		ApprovedAt:   c.ApprovedAt,
		CompletedAt:  c.CompletedAt,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

// CreateCancelRequest 创建注销申请
func (r *tenantRepository) CreateCancelRequest(ctx context.Context, c *model.CancelRequest) error {
	po := cancelRequestFromDomain(c)
	return r.db.WithContext(ctx).Create(po).Error
}

// GetCancelRequestByID 根据ID查询注销申请
func (r *tenantRepository) GetCancelRequestByID(ctx context.Context, id string) (*model.CancelRequest, error) {
	var po CancelRequestPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

// GetPendingCancelRequestByTenant 查询租户未处理的注销申请
// 若租户已有待审批或已通过的申请，不允许重复申请
func (r *tenantRepository) GetPendingCancelRequestByTenant(ctx context.Context, tenantID string) (*model.CancelRequest, error) {
	var po CancelRequestPO
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status IN ?", tenantID, []int{
			model.CancelRequestStatusPending,
			model.CancelRequestStatusApproved,
		}).
		Order("created_at DESC").
		First(&po).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return po.toDomain(), nil
}

// UpdateCancelRequest 更新注销申请
func (r *tenantRepository) UpdateCancelRequest(ctx context.Context, c *model.CancelRequest) error {
	po := cancelRequestFromDomain(c)
	return r.db.WithContext(ctx).Model(&CancelRequestPO{}).
		Where("id = ?", c.ID).
		Select("status", "approver_id", "review_remark", "effective_at", "approved_at", "completed_at", "updated_at").
		Updates(po).Error
}

// FindCancelRequests 分页查询注销申请
func (r *tenantRepository) FindCancelRequests(ctx context.Context, page, pageSize int, status int, keyword string) ([]*model.CancelRequest, int64, error) {
	var pos []*CancelRequestPO
	var total int64

	query := r.db.WithContext(ctx).Model(&CancelRequestPO{})
	if status > 0 {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		query = query.Where("tenant_name LIKE ? OR tenant_id = ?", "%"+keyword+"%", keyword)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*model.CancelRequest, len(pos))
	for i, po := range pos {
		items[i] = po.toDomain()
	}
	return items, total, nil
}

// FindApprovedDueCancelRequests 查询所有已到期可执行的注销申请
func (r *tenantRepository) FindApprovedDueCancelRequests(ctx context.Context, now time.Time) ([]*model.CancelRequest, error) {
	var pos []*CancelRequestPO
	err := r.db.WithContext(ctx).
		Where("status = ? AND effective_at IS NOT NULL AND effective_at <= ?",
			model.CancelRequestStatusApproved, now).
		Find(&pos).Error
	if err != nil {
		return nil, err
	}

	items := make([]*model.CancelRequest, len(pos))
	for i, po := range pos {
		items[i] = po.toDomain()
	}
	return items, nil
}

// CancelRequestsAutoMigrate 迁移注销申请表
func CancelRequestsAutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&CancelRequestPO{})
}
