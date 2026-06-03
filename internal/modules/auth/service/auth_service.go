package service

import (
	"context"
	"errors"
	"time"

	"meteorx/internal/common/jwt"
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/user/model"
	"meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	"meteorx/pkg/uuid"
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

func (s *AuthService) Register(ctx context.Context, req dto.RegisterUserReq) (*model.User, error) {
	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &model.User{
		ID:        uuid.Generate(),
		TenantID:  req.TenantID,
		Username:  req.Username,
		Password:  hashedPassword,
		Nickname:  req.Nickname,
		Role:      "admin",
		Status:    1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginReq) (*model.User, string, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.TenantID, req.Username)
	if err != nil {
		return nil, "", errors.New("account or password is incorrect")
	}

	if !crypto.CheckPassword(req.Password, user.Password) {
		return nil, "", errors.New("account or password is incorrect")
	}

	if user.Status == 0 {
		return nil, "", errors.New("account is disabled")
	}

	token, err := s.tokenHelper.GenerateToken(user.ID, user.TenantID, user.Role)
	if err != nil {
		return nil, "", errors.New("failed to generate token")
	}

	return user, token, nil
}
