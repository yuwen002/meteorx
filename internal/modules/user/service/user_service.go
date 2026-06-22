package service

import (
	"context"
	"fmt"
	rbacModel "meteorx/internal/modules/rbac/model"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	tenantRepository "meteorx/internal/modules/tenant/repository"
	"meteorx/internal/modules/user/dto"
	"meteorx/internal/modules/user/model"
	"meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	ulpkg "meteorx/pkg/ulid"
	"time"
)

type UserService struct {
	repo       repository.UserRepository
	tenantRepo tenantRepository.TenantRepository
	roleRepo   rbacRepo.RoleRepository
}

func NewUserService(repo repository.UserRepository, tenantRepo tenantRepository.TenantRepository, roleRepo rbacRepo.RoleRepository) *UserService {
	return &UserService{repo: repo, tenantRepo: tenantRepo, roleRepo: roleRepo}
}

// validateRole 校验角色编码是否存在于 roles 表中，且作用域匹配、状态启用
func (s *UserService) validateRole(ctx context.Context, roleCode string, scope string) error {
	role, err := s.roleRepo.GetByCode(ctx, "", roleCode)
	if err != nil {
		return fmt.Errorf("角色 '%s' 不存在", roleCode)
	}
	if role.Status == 0 {
		return fmt.Errorf("角色 '%s' 已禁用", roleCode)
	}
	// 校验作用域：角色的 scope 必须匹配当前操作上下文，或为 all
	if role.Scope != rbacModel.RoleScopeAll && role.Scope != scope {
		return fmt.Errorf("角色 '%s' 不允许在当前上下文分配", roleCode)
	}
	return nil
}

// ListByTenant 获取租户下的用户列表（支持分页和关键字搜索）
func (s *UserService) ListByTenant(ctx context.Context, tenantID string, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	users, total, err := s.repo.ListByTenant(ctx, tenantID, page, pageSize, keyword)
	if err != nil {
		return nil, 0, err
	}
	var resp []*dto.UserResp
	for _, user := range users {
		resp = append(resp, &dto.UserResp{
			ID:        user.ID,
			TenantID:  user.TenantID,
			Username:  user.Username,
			Nickname:  user.Nickname,
			Email:     user.Email,
			Role:      user.Role,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return resp, total, nil
}

// GetByID 获取用户详情
func (s *UserService) GetByID(ctx context.Context, userID string) (*dto.UserResp, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &dto.UserResp{
		ID:        user.ID,
		TenantID:  user.TenantID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// Create 创建新用户
func (s *UserService) Create(ctx context.Context, tenantID string, req dto.CreateUserReq) (*dto.UserResp, error) {
	// 检查用户名是否已存在
	exists, err := s.repo.UsernameExists(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("用户名已被使用")
	}

	// 校验角色是否存在且启用，且作用域为 tenant
	if err := s.validateRole(ctx, req.Role, rbacModel.RoleScopeTenant); err != nil {
		return nil, err
	}

	// 密码加密
	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := &model.User{
		ID:       ulpkg.Generate(),
		TenantID: tenantID,
		Username: req.Username,
		Password: hashedPassword,
		Nickname: req.Nickname,
		Email:    req.Email,
		Role:     req.Role,
		Status:   1,
		IsMaster: false,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return &dto.UserResp{
		ID:        user.ID,
		TenantID:  user.TenantID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// Update 更新用户信息
func (s *UserService) Update(ctx context.Context, userID string, req dto.UpdateUserReq) (*dto.UserResp, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Role != "" {
		// 校验角色是否存在且启用，且作用域为 tenant
		if err := s.validateRole(ctx, req.Role, rbacModel.RoleScopeTenant); err != nil {
			return nil, err
		}
		user.Role = req.Role
	}
	if req.Status != nil {
		user.Status = *req.Status
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	return &dto.UserResp{
		ID:        user.ID,
		TenantID:  user.TenantID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: now,
	}, nil
}

// Delete 删除用户
func (s *UserService) Delete(ctx context.Context, userID string) error {
	return s.repo.Delete(ctx, userID)
}

// BelongsToTenant 检查用户是否属于指定租户
func (s *UserService) BelongsToTenant(ctx context.Context, userID, tenantID string) bool {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return false
	}
	return user.TenantID == tenantID
}

// ============ 系统管理员相关方法 ============

// ListMasterAdmins 获取所有系统管理员列表（支持分页和关键词搜索）
func (s *UserService) ListMasterAdmins(ctx context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	users, total, err := s.repo.ListMasterAdmins(ctx, page, pageSize, keyword)
	if err != nil {
		return nil, 0, err
	}

	var resp []*dto.UserResp
	for _, user := range users {
		resp = append(resp, &dto.UserResp{
			ID:        user.ID,
			TenantID:  user.TenantID,
			Username:  user.Username,
			Nickname:  user.Nickname,
			Email:     user.Email,
			Role:      user.Role,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return resp, total, nil
}

// GetMasterAdmin 获取系统管理员详情
func (s *UserService) GetMasterAdmin(ctx context.Context, userID string) (*dto.UserResp, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.IsMaster {
		return nil, fmt.Errorf("用户不是系统管理员")
	}
	return &dto.UserResp{
		ID:        user.ID,
		TenantID:  user.TenantID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// CreateMasterAdmin 创建系统管理员
func (s *UserService) CreateMasterAdmin(ctx context.Context, req dto.CreateMasterAdminReq) (*dto.UserResp, error) {
	// 检查用户名是否已存在
	exists, err := s.repo.UsernameExists(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("用户名已被使用")
	}

	// 校验 superadmin 角色是否存在且启用
	if err := s.validateRole(ctx, "superadmin", rbacModel.RoleScopeSystem); err != nil {
		return nil, fmt.Errorf("系统管理员角色未配置，请先在 roles 表中初始化 superadmin 角色")
	}

	// 密码加密
	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 创建系统管理员（自动设置角色为 superadmin，租户为 SYSTEM_ROOT）
	user := &model.User{
		ID:       ulpkg.Generate(),
		TenantID: "SYSTEM_ROOT",
		Username: req.Username,
		Password: hashedPassword,
		Nickname: req.Nickname,
		Email:    req.Email,
		Role:     "superadmin",
		Status:   1,
		IsMaster: true,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	// 重新查询用户以获取正确的时间戳
	createdUser, err := s.repo.GetByID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &dto.UserResp{
		ID:        createdUser.ID,
		TenantID:  createdUser.TenantID,
		Username:  createdUser.Username,
		Nickname:  createdUser.Nickname,
		Email:     createdUser.Email,
		Role:      createdUser.Role,
		Status:    createdUser.Status,
		CreatedAt: createdUser.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: createdUser.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// UpdateMasterAdmin 更新系统管理员信息
func (s *UserService) UpdateMasterAdmin(ctx context.Context, userID string, req dto.UpdateUserReq) (*dto.UserResp, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.IsMaster {
		return nil, fmt.Errorf("用户不是系统管理员")
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Status != nil {
		user.Status = *req.Status
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	return &dto.UserResp{
		ID:        user.ID,
		TenantID:  user.TenantID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: now,
	}, nil
}

// DeleteMasterAdmin 删除系统管理员
func (s *UserService) DeleteMasterAdmin(ctx context.Context, userID string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.IsMaster {
		return fmt.Errorf("用户不是系统管理员")
	}
	return s.repo.Delete(ctx, userID)
}

// AdminCreateTenantUser 系统管理员为指定租户创建用户
func (s *UserService) AdminCreateTenantUser(ctx context.Context, req dto.AdminCreateTenantUserReq) (*dto.UserResp, error) {
	// 检查用户名是否已存在
	exists, err := s.repo.UsernameExists(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("用户名已被使用")
	}

	// 校验角色是否存在且启用，且作用域为 tenant（管理员为租户创建用户，只能分配租户级角色）
	if err := s.validateRole(ctx, req.Role, rbacModel.RoleScopeTenant); err != nil {
		return nil, err
	}

	// 密码加密
	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 创建用户（关联到指定租户）
	user := &model.User{
		ID:       ulpkg.Generate(),
		TenantID: req.TenantID, // 使用指定的租户ID
		Username: req.Username,
		Password: string(hashedPassword),
		Nickname: req.Nickname,
		Email:    req.Email,
		Role:     req.Role,
		Status:   1,
		IsMaster: false,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return &dto.UserResp{
		ID:        user.ID,
		TenantID:  user.TenantID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// AdminListTenantUsers 系统管理员获取指定租户的用户列表
func (s *UserService) AdminListTenantUsers(ctx context.Context, tenantID string, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	users, total, err := s.repo.ListByTenant(ctx, tenantID, page, pageSize, keyword)
	if err != nil {
		return nil, 0, err
	}

	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}

	var resp []*dto.UserResp
	for _, user := range users {
		resp = append(resp, &dto.UserResp{
			ID:         user.ID,
			TenantID:   user.TenantID,
			TenantName: tenant.Name,
			Username:   user.Username,
			Nickname:   user.Nickname,
			Email:      user.Email,
			Role:       user.Role,
			Status:     user.Status,
			CreatedAt:  user.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:  user.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return resp, total, nil
}

func (s *UserService) AdminListAllTenantUsers(ctx context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	users, total, err := s.repo.ListAllTenantUsers(ctx, page, pageSize, keyword)
	if err != nil {
		return nil, 0, err
	}

	tenantNames := make(map[string]string)
	for _, user := range users {
		if _, ok := tenantNames[user.TenantID]; !ok {
			tenant, err := s.tenantRepo.GetByID(ctx, user.TenantID)
			if err == nil {
				tenantNames[user.TenantID] = tenant.Name
			} else {
				tenantNames[user.TenantID] = ""
			}
		}
	}

	var resp []*dto.UserResp
	for _, user := range users {
		resp = append(resp, &dto.UserResp{
			ID:         user.ID,
			TenantID:   user.TenantID,
			TenantName: tenantNames[user.TenantID],
			Username:   user.Username,
			Nickname:   user.Nickname,
			Email:      user.Email,
			Role:       user.Role,
			Status:     user.Status,
			CreatedAt:  user.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:  user.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return resp, total, nil
}

// AdminUpdateTenantUser 系统管理员更新指定租户的用户
func (s *UserService) AdminUpdateTenantUser(ctx context.Context, tenantID, userID string, req dto.UpdateUserReq) (*dto.UserResp, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.TenantID != tenantID {
		return nil, fmt.Errorf("用户不属于指定租户")
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Role != "" {
		// 校验角色是否存在且启用，且作用域为 tenant
		if err := s.validateRole(ctx, req.Role, rbacModel.RoleScopeTenant); err != nil {
			return nil, err
		}
		user.Role = req.Role
	}
	if req.Status != nil {
		user.Status = *req.Status
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	tenant, err := s.tenantRepo.GetByID(ctx, user.TenantID)
	var tenantName string
	if err == nil {
		tenantName = tenant.Name
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	return &dto.UserResp{
		ID:         user.ID,
		TenantID:   user.TenantID,
		TenantName: tenantName,
		Username:   user.Username,
		Nickname:   user.Nickname,
		Email:      user.Email,
		Role:       user.Role,
		Status:     user.Status,
		CreatedAt:  user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  now,
	}, nil
}

// AdminDeleteTenantUser 系统管理员删除指定租户的用户
func (s *UserService) AdminDeleteTenantUser(ctx context.Context, tenantID, userID string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.TenantID != tenantID {
		return fmt.Errorf("用户不属于指定租户")
	}

	return s.repo.Delete(ctx, userID)
}

// ChangePassword 修改用户密码
func (s *UserService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// 验证原密码
	if !crypto.CheckPassword(oldPassword, user.Password) {
		return fmt.Errorf("原密码错误")
	}

	// 加密新密码
	hashedPassword, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	return s.repo.Update(ctx, user)
}
