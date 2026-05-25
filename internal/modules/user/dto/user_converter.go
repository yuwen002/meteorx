package dto

import (
	"meteorx/internal/modules/user/model"
)

type UserConverter struct{}

// ToResponse 将用户模型转换为响应对象
func (c *UserConverter) ToResponse(user *model.User) *UserResp {
	if user == nil {
		return nil
	}
	return &UserResp{
		ID:       user.ID,
		TenantID: user.TenantID,
		Username: user.Username,
		Nickname: user.Nickname,
		Email:    user.Email,
		Role:     user.Role,
		Status:   user.Status,
		// 转换时间格式，前端更友好
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ToResponseList 批量转换（以后用户列表页会用到）
func (c *UserConverter) ToResponseList(users []*model.User) []*UserResp {
	resps := make([]*UserResp, len(users))
	for i, u := range users {
		resps[i] = c.ToResponse(u)
	}
	return resps
}
