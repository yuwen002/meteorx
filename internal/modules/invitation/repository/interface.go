// Package repository 定义邀请模块的仓储接口和实现。
package repository

import (
	"context"
	"time"

	"meteorx/internal/modules/invitation/model"
)

// InvitationRepository 邀请仓储接口，定义邀请数据的持久化操作。
type InvitationRepository interface {
	// Create 创建邀请记录
	Create(ctx context.Context, inv *model.Invitation) error
	// GetByToken 通过邀请令牌查询邀请（用于接受邀请时验证令牌）
	GetByToken(ctx context.Context, token string) (*model.Invitation, error)
	// GetByID 通过 ID 查询邀请
	GetByID(ctx context.Context, id string) (*model.Invitation, error)
	// ListByTenant 按租户分页查询邀请列表，支持关键词和状态筛选
	ListByTenant(ctx context.Context, tenantID string, page, pageSize int, keyword, status string) ([]*model.Invitation, int64, error)
	// UpdateStatus 更新邀请状态
	UpdateStatus(ctx context.Context, id, status string) error
	// Delete 物理删除邀请记录
	Delete(ctx context.Context, id string) error
	// CountPendingByTenant 统计租户内待处理邀请数量
	CountPendingByTenant(ctx context.Context, tenantID string) (int64, error)
	// FindByEmailAndTenant 按邮箱和租户查询邀请（用于重复邀请校验）
	FindByEmailAndTenant(ctx context.Context, email, tenantID string) (*model.Invitation, error)
	// ExpireOutdated 将所有超过有效期仍为 pending 的邀请置为 expired，返回受影响行数
	ExpireOutdated(ctx context.Context, now time.Time) (int64, error)
}