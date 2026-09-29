// Package repository 实现邀请模块的 GORM 仓储层。
// 提供邀请数据的 CRUD、按令牌/租户/邮箱查询等操作。
package repository

import (
	"context"
	"meteorx/internal/modules/invitation/model"
	"time"

	"gorm.io/gorm"
)

type InvitationPO struct {
	ID         string         `gorm:"primaryKey;size:26;comment:邀请ID"`
	TenantID   string         `gorm:"index;size:26;not null;comment:租户ID"`
	Email      string         `gorm:"size:100;not null;index;comment:被邀请邮箱"`
	Token      string         `gorm:"size:64;uniqueIndex;not null;comment:邀请令牌"`
	RoleIDs    string         `gorm:"type:text;comment:角色ID列表(JSON数组)"`
	Status     string         `gorm:"size:20;default:pending;index;comment:状态"`
	InvitedBy  string         `gorm:"size:26;not null;comment:邀请人ID"`
	ExpiresAt  time.Time      `gorm:"not null;comment:过期时间"`
	AcceptedAt *time.Time     `gorm:"comment:接受时间"`
	CreatedAt  time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
}

func (InvitationPO) TableName() string {
	return "invitations"
}

func (record InvitationPO) toDomain() *model.Invitation {
	inv := &model.Invitation{
		ID:        record.ID,
		TenantID:  record.TenantID,
		Email:     record.Email,
		Token:     record.Token,
		RoleIDs:   record.RoleIDs,
		Status:    record.Status,
		InvitedBy: record.InvitedBy,
		ExpiresAt: record.ExpiresAt,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
	if record.AcceptedAt != nil {
		inv.AcceptedAt = record.AcceptedAt
	}
	return inv
}

type invitationRepository struct {
	db *gorm.DB
}

func NewInvitationRepository(db *gorm.DB) InvitationRepository {
	return &invitationRepository{db: db}
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&InvitationPO{})
}

func (r *invitationRepository) Create(ctx context.Context, inv *model.Invitation) error {
	record := InvitationPO{
		ID:        inv.ID,
		TenantID:  inv.TenantID,
		Email:     inv.Email,
		Token:     inv.Token,
		RoleIDs:   inv.RoleIDs,
		Status:    inv.Status,
		InvitedBy: inv.InvitedBy,
		ExpiresAt: inv.ExpiresAt,
	}
	if inv.AcceptedAt != nil {
		record.AcceptedAt = inv.AcceptedAt
	}
	return r.db.WithContext(ctx).Create(&record).Error
}

func (r *invitationRepository) GetByToken(ctx context.Context, token string) (*model.Invitation, error) {
	var record InvitationPO
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&record).Error
	if err != nil {
		return nil, err
	}
	return record.toDomain(), nil
}

func (r *invitationRepository) GetByID(ctx context.Context, id string) (*model.Invitation, error) {
	var record InvitationPO
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return record.toDomain(), nil
}

func (r *invitationRepository) ListByTenant(ctx context.Context, tenantID string, page, pageSize int, keyword, status string) ([]*model.Invitation, int64, error) {
	var records []InvitationPO
	var total int64

	query := r.db.WithContext(ctx).Model(&InvitationPO{}).Where("tenant_id = ?", tenantID)

	if keyword != "" {
		query = query.Where("email LIKE ?", "%"+keyword+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}

	if err := query.Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, 0, err
	}

	var invs []*model.Invitation
	for _, record := range records {
		invs = append(invs, record.toDomain())
	}
	return invs, total, nil
}

func (r *invitationRepository) UpdateStatus(ctx context.Context, id, status string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if status == model.InvitationStatusAccepted {
		updates["accepted_at"] = time.Now()
	}
	return r.db.WithContext(ctx).Model(&InvitationPO{}).Where("id = ?", id).Updates(updates).Error
}

func (r *invitationRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&InvitationPO{}, "id = ?", id).Error
}

func (r *invitationRepository) CountPendingByTenant(ctx context.Context, tenantID string) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&InvitationPO{}).
		Where("tenant_id = ? AND status = ?", tenantID, model.InvitationStatusPending).
		Count(&total).Error
	return total, err
}

func (r *invitationRepository) FindByEmailAndTenant(ctx context.Context, email, tenantID string) (*model.Invitation, error) {
	var record InvitationPO
	err := r.db.WithContext(ctx).
		Where("email = ? AND tenant_id = ? AND status = ?", email, tenantID, model.InvitationStatusPending).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return record.toDomain(), nil
}