package dto

import (
	"meteorx/internal/modules/user/model"
	"time"
)

// UserConverter 用户模型与响应 DTO 之间的转换器（无状态）。
type UserConverter struct{}

// ToResponse 将用户模型转换为响应对象
func (c *UserConverter) ToResponse(user *model.User) *UserResp {
	if user == nil {
		return nil
	}
	return &UserResp{
		ID:            user.ID,
		TenantID:      user.TenantID,
		Username:      user.Username,
		Nickname:      user.Nickname,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Phone:         user.Phone,
		Avatar:        user.Avatar,
		Department:    user.Department,
		Position:      user.Position,
		Remark:        user.Remark,
		Roles:         user.Roles,
		Status:        user.Status,
		IsMaster:      user.IsMaster,
		LastLoginAt:   formatTimePtr(user.LastLoginAt),
		CreatedAt:     user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// formatTimePtr 将可空时间指针格式化为字符串，nil 返回空串。
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// ToResponseList 批量转换（以后用户列表页会用到）
func (c *UserConverter) ToResponseList(users []*model.User) []*UserResp {
	resps := make([]*UserResp, len(users))
	for i, u := range users {
		resps[i] = c.ToResponse(u)
	}
	return resps
}
