package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"meteorx/internal/cache"
	"meteorx/internal/common/jwt"
	"meteorx/internal/modules/auth/dto"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	"meteorx/internal/modules/user/model"
	"meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	"meteorx/pkg/uuid"
)

const (
	tokenBlacklistPrefix = "token:blacklist:"
)

type AuthService struct {
	userRepo           repository.UserRepository
	roleRepo           rbacRepo.RoleRepository
	userRoleRepo       rbacRepo.UserRoleRepository
	rolePermissionRepo rbacRepo.RolePermissionRepository
	tokenHelper        *jwt.TokenHelper
	redis              *cache.Redis
}

func NewAuthService(ur repository.UserRepository, rr rbacRepo.RoleRepository, urr rbacRepo.UserRoleRepository, rpr rbacRepo.RolePermissionRepository, th *jwt.TokenHelper, redis *cache.Redis) *AuthService {
	return &AuthService{
		userRepo:           ur,
		roleRepo:           rr,
		userRoleRepo:       urr,
		rolePermissionRepo: rpr,
		tokenHelper:        th,
		redis:              redis,
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

func (s *AuthService) Login(ctx context.Context, req dto.LoginReq) (*model.User, []string, []string, string, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.TenantID, req.Username)
	if err != nil {
		return nil, nil, nil, "", errors.New("account or password is incorrect")
	}

	if !crypto.CheckPassword(req.Password, user.Password) {
		return nil, nil, nil, "", errors.New("account or password is incorrect")
	}

	if user.Status == 0 {
		return nil, nil, nil, "", errors.New("account is disabled")
	}

	// 查询用户关联的角色编码列表
	roleCodes, err := s.userRoleRepo.GetRoleCodesByUserID(ctx, user.ID)
	if err != nil {
		return nil, nil, nil, "", errors.New("failed to query user roles")
	}

	// 如果是系统管理员且没有角色，兜底返回 superadmin
	if user.IsMaster && len(roleCodes) == 0 {
		roleCodes = []string{"superadmin"}
	}

	user.Roles = roleCodes

	// 查询用户关联的所有权限码（通过角色 -> 权限关联表）
	permissionCodes := make([]string, 0)
	if user.IsMaster {
		// 系统管理员兜底：返回空数组，前端通过 is_master 标志判断拥有所有权限
	} else {
		roleIDs, err := s.userRoleRepo.GetRoleIDsByUserID(ctx, user.ID)
		if err == nil {
			seen := make(map[string]bool)
			for _, roleID := range roleIDs {
				codes, err := s.rolePermissionRepo.GetPermissionCodesByRoleID(ctx, roleID)
				if err != nil {
					continue
				}
				for _, c := range codes {
					if !seen[c] {
						seen[c] = true
						permissionCodes = append(permissionCodes, c)
					}
				}
			}
		}
	}

	token, err := s.tokenHelper.GenerateToken(user.ID, user.TenantID, roleCodes)
	if err != nil {
		return nil, nil, nil, "", errors.New("failed to generate token")
	}

	return user, roleCodes, permissionCodes, token, nil
}

// Logout 用户登出，将 token 加入黑名单
func (s *AuthService) Logout(ctx context.Context, tokenString string) error {
	if s.redis == nil {
		return errors.New("redis not initialized")
	}

	// 解析 token 获取过期时间
	claims, err := s.tokenHelper.ParseToken(tokenString)
	if err != nil {
		return errors.New("invalid token")
	}

	// 计算 token 剩余有效期
	now := time.Now()
	var expiration time.Duration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.After(now) {
		expiration = claims.ExpiresAt.Time.Sub(now)
	} else {
		// token 已过期，无需加入黑名单
		return nil
	}

	// 将 token 加入黑名单，有效期与 token 剩余有效期相同
	key := fmt.Sprintf("%s%s", tokenBlacklistPrefix, tokenString)
	return s.redis.Set(ctx, key, "1", expiration)
}

// IsTokenBlacklisted 检查 token 是否在黑名单中
func (s *AuthService) IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	if s.redis == nil {
		return false, nil
	}

	key := fmt.Sprintf("%s%s", tokenBlacklistPrefix, tokenString)
	return s.redis.Exists(ctx, key)
}
