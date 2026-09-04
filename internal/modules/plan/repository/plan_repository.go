// Package repository 套餐与订阅数据访问层
// 基于 GORM 实现 PlanRepository 与 SubscriptionRepository 接口
package repository

import (
	"context"
	"time"

	"meteorx/internal/modules/plan/model"

	"gorm.io/gorm"
)

// 确保实现接口
var (
	_ PlanRepository         = (*planRepository)(nil)
	_ SubscriptionRepository = (*subscriptionRepository)(nil)
)

// PlanPO 套餐持久化模型
type PlanPO struct {
	ID          string         `gorm:"primaryKey;size:26;comment:'套餐唯一标识'"`
	Name        *string        `gorm:"size:100;comment:'套餐名称'"`
	Code        string         `gorm:"size:50;uniqueIndex;comment:'套餐编码，全局唯一'"`
	Description *string        `gorm:"size:255;comment:'套餐描述'"`
	UserLimit   int            `gorm:"comment:'用户数上限，-1表示不限'"`
	Price       float64        `gorm:"type:decimal(10,2);comment:'月费价格'"`
	Status      int            `gorm:"index;comment:'状态：1-启用 0-停用'"`
	CreatedAt   time.Time      `gorm:"comment:'创建时间'"`
	UpdatedAt   time.Time      `gorm:"comment:'更新时间'"`
	DeletedAt   gorm.DeletedAt `gorm:"index;comment:'软删除时间'"`
}

func (PlanPO) TableName() string { return "tenant_plans" }

// SubscriptionPO 租户订阅持久化模型
type SubscriptionPO struct {
	ID        string         `gorm:"primaryKey;size:26;comment:'订阅唯一标识'"`
	TenantID  string         `gorm:"size:26;index;comment:'租户ID'"`
	PlanID    string         `gorm:"size:26;index;comment:'套餐ID'"`
	Status    int            `gorm:"index;comment:'状态：1-生效 2-到期 3-取消'"`
	StartedAt time.Time      `gorm:"comment:'生效开始时间'"`
	ExpiresAt *time.Time     `gorm:"comment:'到期时间，NULL表示长期有效'"`
	CreatedAt time.Time      `gorm:"comment:'创建时间'"`
	UpdatedAt time.Time      `gorm:"comment:'更新时间'"`
	DeletedAt gorm.DeletedAt `gorm:"index;comment:'软删除时间'"`
}

func (SubscriptionPO) TableName() string { return "tenant_subscriptions" }

// AutoMigrate 套餐模块数据库迁移
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&PlanPO{}, &SubscriptionPO{})
}

// ---------- 工具函数 ----------

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (p PlanPO) toDomain() *model.Plan {
	plan := &model.Plan{
		ID:          p.ID,
		Name:        strVal(p.Name),
		Code:        p.Code,
		Description: strVal(p.Description),
		UserLimit:   p.UserLimit,
		Price:       p.Price,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
	if p.DeletedAt.Valid {
		plan.DeletedAt = &p.DeletedAt.Time
	}
	return plan
}

func (s SubscriptionPO) toDomain() *model.TenantSubscription {
	sub := &model.TenantSubscription{
		ID:        s.ID,
		TenantID:  s.TenantID,
		PlanID:    s.PlanID,
		Status:    s.Status,
		StartedAt: s.StartedAt,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
	if s.ExpiresAt != nil {
		exp := *s.ExpiresAt
		sub.ExpiresAt = &exp
	}
	return sub
}

// ---------- PlanRepository 实现 ----------

type planRepository struct {
	db *gorm.DB
}

func NewPlanRepository(db *gorm.DB) PlanRepository {
	return &planRepository{db: db}
}

func (r *planRepository) Create(ctx context.Context, plan *model.Plan) error {
	po := &PlanPO{
		ID:          plan.ID,
		Name:        strPtr(plan.Name),
		Code:        plan.Code,
		Description: strPtr(plan.Description),
		UserLimit:   plan.UserLimit,
		Price:       plan.Price,
		Status:      plan.Status,
	}
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	plan.CreatedAt = po.CreatedAt
	plan.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *planRepository) GetByID(ctx context.Context, id string) (*model.Plan, error) {
	var po PlanPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *planRepository) GetByCode(ctx context.Context, code string) (*model.Plan, error) {
	var po PlanPO
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&po).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *planRepository) Update(ctx context.Context, id string, plan *model.Plan) error {
	updates := map[string]interface{}{}
	if plan.Name != "" {
		updates["name"] = plan.Name
	}
	if plan.Code != "" {
		updates["code"] = plan.Code
	}
	if plan.Description != "" {
		updates["description"] = plan.Description
	}
	if plan.UserLimit >= -1 {
		updates["user_limit"] = plan.UserLimit
	}
	if plan.Price >= 0 {
		updates["price"] = plan.Price
	}
	if plan.Status >= 0 {
		updates["status"] = plan.Status
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&PlanPO{}).Where("id = ?", id).Updates(updates).Error
}

func (r *planRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&PlanPO{}, "id = ?", id).Error
}

func (r *planRepository) FindPage(ctx context.Context, page, pageSize int, keyword string, status *int) ([]*model.Plan, int64, error) {
	var pos []*PlanPO
	var total int64

	query := r.db.WithContext(ctx).Model(&PlanPO{})
	if keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	var plans []*model.Plan
	for _, po := range pos {
		plans = append(plans, po.toDomain())
	}
	return plans, total, nil
}

func (r *planRepository) ListAllEnabled(ctx context.Context) ([]*model.Plan, error) {
	var pos []*PlanPO
	if err := r.db.WithContext(ctx).Where("status = ?", model.StatusEnabled).Order("created_at ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	var plans []*model.Plan
	for _, po := range pos {
		plans = append(plans, po.toDomain())
	}
	return plans, nil
}

func (r *planRepository) CountByID(ctx context.Context, ids []string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&PlanPO{}).Where("id IN ?", ids).Count(&count).Error
	return count, err
}

// ---------- SubscriptionRepository 实现 ----------

type subscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) GetActiveByTenant(ctx context.Context, tenantID string) (*model.TenantSubscription, error) {
	var po SubscriptionPO
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = ?", tenantID, model.SubscriptionActive).
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

func (r *subscriptionRepository) Create(ctx context.Context, sub *model.TenantSubscription) error {
	po := &SubscriptionPO{
		ID:        sub.ID,
		TenantID:  sub.TenantID,
		PlanID:    sub.PlanID,
		Status:    sub.Status,
		StartedAt: sub.StartedAt,
	}
	if sub.ExpiresAt != nil {
		exp := *sub.ExpiresAt
		po.ExpiresAt = &exp
	}
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	sub.CreatedAt = po.CreatedAt
	sub.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *subscriptionRepository) UpdateStatus(ctx context.Context, id string, status int) error {
	return r.db.WithContext(ctx).Model(&SubscriptionPO{}).Where("id = ?", id).Update("status", status).Error
}

func (r *subscriptionRepository) FindExpiredActive(ctx context.Context) ([]*model.TenantSubscription, error) {
	var pos []*SubscriptionPO
	err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?", model.SubscriptionActive, time.Now()).
		Find(&pos).Error
	if err != nil {
		return nil, err
	}
	var subs []*model.TenantSubscription
	for _, po := range pos {
		subs = append(subs, po.toDomain())
	}
	return subs, nil
}

func (r *subscriptionRepository) CountByPlan(ctx context.Context, planID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&SubscriptionPO{}).
		Where("plan_id = ? AND status = ?", planID, model.SubscriptionActive).
		Count(&count).Error
	return count, err
}

func (r *subscriptionRepository) ListActiveByPlans(ctx context.Context, planIDs []string) (map[string]int64, error) {
	result := make(map[string]int64)
	if len(planIDs) == 0 {
		return result, nil
	}
	type row struct {
		PlanID string
		Count  int64
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&SubscriptionPO{}).
		Select("plan_id, COUNT(*) as count").
		Where("plan_id IN ? AND status = ?", planIDs, model.SubscriptionActive).
		Group("plan_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		result[r.PlanID] = r.Count
	}
	return result, nil
}

func (r *subscriptionRepository) ListActiveByTenants(ctx context.Context, tenantIDs []string) ([]*model.TenantSubscription, error) {
	if len(tenantIDs) == 0 {
		return nil, nil
	}
	var pos []*SubscriptionPO
	err := r.db.WithContext(ctx).
		Where("tenant_id IN ? AND status = ?", tenantIDs, model.SubscriptionActive).
		Find(&pos).Error
	if err != nil {
		return nil, err
	}
	var subs []*model.TenantSubscription
	for _, po := range pos {
		subs = append(subs, po.toDomain())
	}
	return subs, nil
}
