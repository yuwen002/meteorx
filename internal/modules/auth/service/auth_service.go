package service

import (
	"context"
	"errors"
	"time"

	"meteorx/internal/common/jwt"
	"meteorx/internal/modules/auth/dto"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	"meteorx/internal/modules/user/model"
	"meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	"meteorx/pkg/uuid"
)

type AuthService struct {
	userRepo     repository.UserRepository
	roleRepo     rbacRepo.RoleRepository
	userRoleRepo rbacRepo.UserRoleRepository
	tokenHelper  *jwt.TokenHelper
}

func NewAuthService(ur repository.UserRepository, rr rbacRepo.RoleRepository, urr rbacRepo.UserRoleRepository, th *jwt.TokenHelper) *AuthService {
	return &AuthService{
		userRepo:     ur,
		roleRepo:     rr,
		userRoleRepo: urr,
		tokenHelper:  th,
	}
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterUserReq) (*model.User, error) {
	// 默认角色为 tenant_admin，校验其是否存在且启用
	defaultRole := "tenant_admin"
	role, err := s.roleRepo.GetByCode(ctx, "", defaultRole)
	if err != nil {
		return nil, errors.New("默认注册角色未配置，请联系管理员初始化角色")
	}
	if role.Status == 0 {
		return nil, errors.New("默认注册角色已禁用，请联系管理员")
	}

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
		Roles:     []string{defaultRole},
		Status:    1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// 创建用户后分配角色
	if err := s.userRoleRepo.AssignRoles(ctx, user.ID, []string{role.ID}); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginReq) (*model.User, []string, string, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.TenantID, req.Username)
	if err != nil {
		return nil, nil, "", errors.New("account or password is incorrect")
	}

	if !crypto.CheckPassword(req.Password, user.Password) {
		return nil, nil, "", errors.New("account or password is incorrect")
	}

	if user.Status == 0 {
		return nil, nil, "", errors.New("account is disabled")
	}

	// 查询用户关联的角色编码列表
	roleCodes, err := s.userRoleRepo.GetRoleCodesByUserID(ctx, user.ID)
	if err != nil {
		return nil, nil, "", errors.New("failed to query user roles")
	}

	// 如果是系统管理员且没有角色，兜底返回 superadmin
	if user.IsMaster && len(roleCodes) == 0 {
		roleCodes = []string{"superadmin"}
	}

	user.Roles = roleCodes

	token, err := s.tokenHelper.GenerateToken(user.ID, user.TenantID, roleCodes)
	if err != nil {
		return nil, nil, "", errors.New("failed to generate token")
	}

	return user, roleCodes, token, nil
}