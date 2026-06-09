package service

import (
	"context"
	"fmt"
	"meteorx/internal/modules/user/dto"
	"meteorx/internal/modules/user/model"
	"meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	ulpkg "meteorx/pkg/ulid"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// ListByTenant 获取租户下的用户列表
func (s *UserService) ListByTenant(ctx context.Context, tenantID string) ([]*dto.UserResp, error) {
	users, _, err := s.repo.ListByTenant(ctx, tenantID, 1, 0) // page=1, pageSize=0 获取全部
	if err != nil {
		return nil, err
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
	return resp, nil
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
		user.Role = req.Role
	}
	if req.Status != 0 {
		user.Status = req.Status
	}

	if err := s.repo.Update(ctx, user); err != nil {
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
	if req.Status != 0 {
		user.Status = req.Status
	}

	if err := s.repo.Update(ctx, user); err != nil {
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
func (s *UserService) AdminListTenantUsers(ctx context.Context, tenantID string, page, pageSize int) (*dto.UserListResp, error) {
	users, total, err := s.repo.ListByTenant(ctx, tenantID, page, pageSize)
	if err != nil {
		return nil, err
	}

	resp := &dto.UserListResp{
		Items: make([]*dto.UserResp, 0, len(users)),
		Total: total,
	}

	for _, user := range users {
		resp.Items = append(resp.Items, &dto.UserResp{
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

	return resp, nil
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
		user.Role = req.Role
	}
	if req.Status != 0 {
		user.Status = req.Status
	}

	if err := s.repo.Update(ctx, user); err != nil {
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
