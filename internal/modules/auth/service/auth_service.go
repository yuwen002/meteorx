package service

import (
	"context"
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/user/model"
	"meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	"time"
)

type AuthService struct {
	userRepo repository.UserRepository
}

func NewAuthService(ur repository.UserRepository) *AuthService {
	return &AuthService{userRepo: ur}
}

// Register 处理用户注册流程
func (s *AuthService) Register(ctx context.Context, req dto.RegisterUserReq) (*model.User, error) {
	// 1. 密码哈希处理（安全第一）
	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 2. 构造 User 领域模型
	// 这里的 ID 建议之后统一用 UUID 生成器
	user := &model.User{
		ID:        "u_" + time.Now().Format("050405"),
		TenantID:  req.TenantID,
		Username:  req.Username,
		Password:  hashedPassword,
		Nickname:  req.Nickname,
		Role:      "admin", // 默认第一个注册的是管理员，或者根据业务定
		Status:    1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 3. 调用 User 仓储层存入数据库
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
