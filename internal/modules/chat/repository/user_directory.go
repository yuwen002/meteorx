// Package repository 提供聊天模块的用户姓名解析与租户成员校验实现。
// 会话/消息响应需展示参与者姓名，此处基于共享 users 表批量查询，对用户模块保持低耦合。
package repository

import (
	"context"

	"gorm.io/gorm"
)

// UserDirectory 基于 users 表的用户姓名解析器与租户成员校验器。
type UserDirectory struct {
	db *gorm.DB
}

// NewUserDirectory 创建用户目录实例。
func NewUserDirectory(db *gorm.DB) *UserDirectory {
	return &UserDirectory{db: db}
}

// userBriefRow 用户展示名查询的轻量结果行。
type userBriefRow struct {
	ID       string
	Nickname string
	Username string
}

// ResolveUserNames 批量解析用户 ID 到展示名（昵称优先、回退用户名）；空输入返回空映射。
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
// 供发起私聊前拦截不存在或跨租户的参与者；Table 直查需显式过滤软删除。
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
