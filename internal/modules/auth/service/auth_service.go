package service

import (
	"context"
	"errors"
	"meteorx/internal/common/jwt"
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/user/model"
	"meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	"time"
)

type AuthService struct {
	userRepo    repository.UserRepository
	tokenHelper *jwt.TokenHelper
}

func NewAuthService(ur repository.UserRepository, th *jwt.TokenHelper) *AuthService {
	return &AuthService{
		userRepo:    ur,
		tokenHelper: th,
	}
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

// Login 处理用户登录
func (s *AuthService) Login(ctx context.Context, req dto.LoginReq) (*model.User, string, error) {
	// 1. 查找用户
	user, err := s.userRepo.GetByUsername(ctx, req.TenantID, req.Username)
	if err != nil {
		// 为了安全，通常不直接说“用户不存在”，而是说“账号或密码错误”
		return nil, "", errors.New("账号或密码错误")
	}

	// 2. 验证密码 (调用你 pkg/crypto 下的工具)
	if !crypto.CheckPassword(req.Password, user.Password) {
		return nil, "", errors.New("账号或密码错误")
	}

	// 3. 验证状态
	if user.Status == 0 {
		return nil, "", errors.New("账号已被禁用")
	}

	// 4. 生成 JWT Token
	token, err := s.tokenHelper.GenerateToken(user.ID, user.TenantID, user.Role)
	if err != nil {
		return nil, "", errors.New("生成令牌失败")
	}

	return user, token, nil
}
