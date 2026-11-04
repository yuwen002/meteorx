// Package repository 提供任务模块的用户姓名解析实现。
// 任务响应需要展示创建人/负责人姓名，此处基于共享的 users 表做批量查询，
// 避免逐条查询（N+1）并对用户模块保持低耦合（仅读取 id/昵称/用户名）。
package repository

import (
	"context"

	"gorm.io/gorm"
)

// UserDirectory 基于 users 表的用户姓名解析器。
type UserDirectory struct {
	db *gorm.DB
}

// NewUserDirectory 创建用户姓名解析器，供任务服务富化人员展示名。
func NewUserDirectory(db *gorm.DB) *UserDirectory {
	return &UserDirectory{db: db}
}

// userBriefRow 用户展示名查询的轻量结果行。
type userBriefRow struct {
	ID       string
	Nickname string
	Username string
}

// ResolveUserNames 批量解析用户 ID 到展示名，展示名取昵称优先、回退用户名。
// ids 为空时返回空映射；软删除用户默认被 GORM 过滤，未命中的 ID 不出现在结果中。
func (d *UserDirectory) ResolveUserNames(ctx context.Context, ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	var rows []userBriefRow
	if err := d.db.WithContext(ctx).Table("users").
		Select("id", "nickname", "username").
		Where("id IN ?", ids).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	names := make(map[string]string, len(rows))
	for _, r := range rows {
		name := r.Nickname
		if name == "" {
			name = r.Username
		}
		names[r.ID] = name
	}
	return names, nil
}

// IsTenantMember 校验 userID 是否为指定 tenantID 下未删除的有效用户。
// 供任务指派前拦截不存在或跨租户的负责人；Table 直查不绑定模型，需显式过滤软删除。
func (d *UserDirectory) IsTenantMember(ctx context.Context, tenantID, userID string) (bool, error) {
	if tenantID == "" || userID == "" {
		return false, nil
	}
	var cnt int64
	err := d.db.WithContext(ctx).Table("users").
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", userID, tenantID).
		Count(&cnt).Error
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// AssigneeBrief 供前端负责人选择器使用的轻量成员信息。
type AssigneeBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListAssignable 返回指定租户下未删除的存活成员（id + 展示名，昵称优先回退用户名），
// 供 /tasks/assignable 端点给普通成员选择负责人（不依赖需 user:list 权限的 /users）。
func (d *UserDirectory) ListAssignable(ctx context.Context, tenantID string) ([]AssigneeBrief, error) {
	out := make([]AssigneeBrief, 0)
	if tenantID == "" {
		return out, nil
	}
	var rows []userBriefRow
	if err := d.db.WithContext(ctx).Table("users").
		Select("id", "nickname", "username").
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Order("username ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		name := r.Nickname
		if name == "" {
			name = r.Username
		}
		out = append(out, AssigneeBrief{ID: r.ID, Name: name})
	}
	return out, nil
}
