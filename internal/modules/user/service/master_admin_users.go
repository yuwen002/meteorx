package service

import (
	"context"
	"fmt"

	rbacModel "meteorx/internal/modules/rbac/model"
	"meteorx/internal/modules/user/dto"
	"meteorx/internal/modules/user/model"
	"meteorx/pkg/crypto"
	"meteorx/pkg/idgen"
)

func (s *UserService) ListMasterAdmins(ctx context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	users, total, err := s.repo.ListMasterAdmins(ctx, page, pageSize, keyword)
	if err != nil {
		return nil, 0, err
	}
	resp, err := s.buildUserRespList(ctx, users)
	if err != nil {
		return nil, 0, err
	}
	return resp, total, nil
}

func (s *UserService) GetMasterAdmin(ctx context.Context, userID string) (*dto.UserResp, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.IsMaster {
		return nil, ErrUserNotSystemAdmin
	}
	return s.buildUserResp(ctx, user)
}

func (s *UserService) CreateMasterAdmin(ctx context.Context, req dto.CreateMasterAdminReq) (*dto.UserResp, error) {
	exists, err := s.repo.UsernameExists(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameExists
	}

	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:       idgen.New(),
		TenantID: "SYSTEM_ROOT",
		Username: req.Username,
		Password: hashedPassword,
		Nickname: req.Nickname,
		Email:    req.Email,
		Status:   1,
		IsMaster: true,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	// 确定要分配的角色
	var roleID string
	if req.RoleID != "" {
		// 校验指定的角色是否为系统级别角色（scope 为 system 或 all）
		if err := s.validateSystemAdminRoleID(ctx, req.RoleID); err != nil {
			return nil, err
		}
		roleID = req.RoleID
	} else {
		// 未指定角色，默认分配 superadmin
		superadminRole, err := s.roleRepo.GetByCode(ctx, "", "superadmin")
		if err != nil {
			return nil, fmt.Errorf("系统管理员角色未配置，请先在 roles 表中初始化 superadmin 角色")
		}
		if superadminRole.Status == 0 {
			return nil, fmt.Errorf("系统管理员角色已禁用")
		}
		roleID = superadminRole.ID
	}

	// 分配角色（单个）
	if err := s.userRoleRepo.AssignRoles(ctx, user.ID, []string{roleID}); err != nil {
		return nil, err
	}

	return s.buildUserResp(ctx, user)
}

func (s *UserService) UpdateMasterAdmin(ctx context.Context, userID string, req dto.UpdateMasterAdminReq) (*dto.UserResp, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.IsMaster {
		return nil, ErrUserNotSystemAdmin
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

	// 如果指定了角色ID，则更新角色
	if req.RoleID != "" {
		// 校验指定的角色是否为系统级别角色（scope 为 system 或 all）
		if err := s.validateSystemAdminRoleID(ctx, req.RoleID); err != nil {
			return nil, err
		}
		if err := s.userRoleRepo.AssignRoles(ctx, user.ID, []string{req.RoleID}); err != nil {
			return nil, err
		}
	}

	return s.buildUserResp(ctx, user)
}

func (s *UserService) DeleteMasterAdmin(ctx context.Context, userID string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.IsMaster {
		return ErrUserNotSystemAdmin
	}

	// 检查是否为系统保护用户（初始管理员，不允许删除）
	if isProtectedUser(userID) {
		return fmt.Errorf("系统初始管理员 [%s] 不允许删除", user.Username)
	}

	// 删除用户前先解除所有角色绑定
	if err := s.userRoleRepo.DeleteByUserID(ctx, userID); err != nil {
		return fmt.Errorf("解除用户角色绑定失败: %w", err)
	}

	return s.repo.Delete(ctx, userID)
}

func (s *UserService) UpdateMasterAdminStatus(ctx context.Context, userID string, status int) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.IsMaster {
		return ErrUserNotSystemAdmin
	}
	return s.repo.UpdateStatus(ctx, userID, status)
}

func (s *UserService) ListDeletedMasterAdmins(ctx context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	users, total, err := s.repo.FindDeletedMasterAdmins(ctx, page, pageSize, keyword)
	if err != nil {
		return nil, 0, err
	}
	resp, err := s.buildUserRespList(ctx, users)
	if err != nil {
		return nil, 0, err
	}
	return resp, total, nil
}

func (s *UserService) RestoreMasterAdmin(ctx context.Context, userID string) error {
	return s.repo.RestoreMasterAdmin(ctx, userID)
}

func (s *UserService) PermanentDeleteMasterAdmin(ctx context.Context, userID string) error {
	// 检查是否为系统保护用户（初始管理员，不允许删除）
	if isProtectedUser(userID) {
		return fmt.Errorf("系统初始管理员不允许永久删除")
	}
	return s.repo.PermanentDeleteMasterAdmin(ctx, userID)
}

func (s *UserService) BatchUpdateMasterAdminStatus(ctx context.Context, ids []string, status int) (int64, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("用户ID列表不能为空")
	}
	return s.repo.BatchUpdateStatus(ctx, ids, status)
}

func (s *UserService) BatchDeleteMasterAdmins(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("用户ID列表不能为空")
	}

	// 检查每个用户
	for _, id := range ids {
		user, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return 0, err
		}
		if !user.IsMaster {
			return 0, fmt.Errorf("用户 [%s] 不是系统管理员", user.Username)
		}

		// 检查是否为系统保护用户（初始管理员，不允许删除）
		if isProtectedUser(id) {
			return 0, fmt.Errorf("系统初始管理员 [%s] 不允许删除", user.Username)
		}

		// 解除用户所有角色绑定
		if err := s.userRoleRepo.DeleteByUserID(ctx, id); err != nil {
			return 0, fmt.Errorf("解除用户 [%s] 角色绑定失败: %w", user.Username, err)
		}
	}

	return s.repo.BatchDelete(ctx, ids)
}

// ============ 辅助函数 ============

// isProtectedUser 检查用户是否为系统保护用户（初始管理员，不允许删除）
// 保护用户ID列表，这些用户是系统初始化的关键用户，不允许删除
var protectedUserIDs = map[string]bool{
	"admin-id-000001": true, // 初始系统管理员
}

func isProtectedUser(userID string) bool {
	return protectedUserIDs[userID]
}

// ============ 系统管理员管理租户用户 ============

func (s *UserService) AdminCreateTenantUser(ctx context.Context, req dto.AdminCreateTenantUserReq) (*dto.UserResp, error) {
	exists, err := s.repo.UsernameExists(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameExists
	}

	// 校验套餐配额
	if err := s.checkUserQuota(ctx, req.TenantID); err != nil {
		return nil, err
	}

	// 校验角色
	if err := s.validateRoleIDs(ctx, req.RoleIDs, rbacModel.RoleScopeTenant); err != nil {
		return nil, err
	}

	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:       idgen.New(),
		TenantID: req.TenantID,
		Username: req.Username,
		Password: hashedPassword,
		Nickname: req.Nickname,
		Email:    req.Email,
		Status:   1,
		IsMaster: false,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	if err := s.userRoleRepo.AssignRoles(ctx, user.ID, req.RoleIDs); err != nil {
		return nil, err
	}

	return s.buildUserResp(ctx, user)
}

func (s *UserService) AdminListTenantUsers(ctx context.Context, tenantID string, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	users, total, err := s.repo.ListByTenant(ctx, tenantID, page, pageSize, keyword, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := s.buildUserRespList(ctx, users)
	if err != nil {
		return nil, 0, err
	}
	return resp, total, nil
}

func (s *UserService) AdminListAllTenantUsers(ctx context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	users, total, err := s.repo.ListAllTenantUsers(ctx, page, pageSize, keyword)
	if err != nil {
		return nil, 0, err
	}
	resp, err := s.buildUserRespList(ctx, users)
	if err != nil {
		return nil, 0, err
	}
	return resp, total, nil
}

func (s *UserService) AdminUpdateTenantUser(ctx context.Context, tenantID, userID string, req dto.UpdateUserReq) (*dto.UserResp, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.TenantID != tenantID {
		return nil, ErrUserNotInTenant
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

	if len(req.RoleIDs) > 0 {
		if err := s.validateRoleIDs(ctx, req.RoleIDs, rbacModel.RoleScopeTenant); err != nil {
			return nil, err
		}
		if err := s.userRoleRepo.AssignRoles(ctx, user.ID, req.RoleIDs); err != nil {
			return nil, err
		}
	}

	return s.buildUserResp(ctx, user)
}

func (s *UserService) AdminDeleteTenantUser(ctx context.Context, tenantID, userID string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.TenantID != tenantID {
		return ErrUserNotInTenant
	}

	// 删除用户前先解除所有角色绑定
	if err := s.userRoleRepo.DeleteByUserID(ctx, userID); err != nil {
		return fmt.Errorf("解除用户角色绑定失败: %w", err)
	}

	return s.repo.Delete(ctx, userID)
}

// AdminResetTenantUserPassword 系统管理员重置指定租户用户的密码
func (s *UserService) AdminResetTenantUserPassword(ctx context.Context, tenantID, userID, newPassword string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.TenantID != tenantID {
		return ErrUserNotInTenant
	}

	hashedPassword, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	return s.repo.Update(ctx, user)
}

func (s *UserService) AdminListDeletedTenantUsers(ctx context.Context, tenantID string, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	users, total, err := s.repo.FindDeletedTenantUsers(ctx, tenantID, page, pageSize, keyword)
	if err != nil {
		return nil, 0, err
	}
	resp, err := s.buildUserRespList(ctx, users)
	if err != nil {
		return nil, 0, err
	}
	return resp, total, nil
}

func (s *UserService) AdminListAllDeletedTenantUsers(ctx context.Context, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	users, total, err := s.repo.FindAllDeletedTenantUsers(ctx, page, pageSize, keyword)
	if err != nil {
		return nil, 0, err
	}
	resp, err := s.buildUserRespList(ctx, users)
	if err != nil {
		return nil, 0, err
	}
	return resp, total, nil
}

func (s *UserService) AdminRestoreTenantUser(ctx context.Context, tenantID, userID string) error {
	return s.repo.RestoreTenantUser(ctx, tenantID, userID)
}

func (s *UserService) AdminPermanentDeleteTenantUser(ctx context.Context, tenantID, userID string) error {
	return s.repo.PermanentDeleteTenantUser(ctx, tenantID, userID)
}

// ==================== 普通租户管理员回收站功能 ====================

// ListDeletedUsers 获取当前租户的已删除用户列表（回收站）
func (s *UserService) ListDeletedUsers(ctx context.Context, tenantID string, page, pageSize int, keyword string) ([]*dto.UserResp, int64, error) {
	users, total, err := s.repo.FindDeletedTenantUsers(ctx, tenantID, page, pageSize, keyword)
	if err != nil {
		return nil, 0, err
	}
	resp, err := s.buildUserRespList(ctx, users)
	if err != nil {
		return nil, 0, err
	}
	return resp, total, nil
}

// RestoreUser 恢复已删除的用户
func (s *UserService) RestoreUser(ctx context.Context, tenantID, userID string) error {
	return s.repo.RestoreTenantUser(ctx, tenantID, userID)
}

// PermanentDeleteUser 永久删除用户（物理删除）
func (s *UserService) PermanentDeleteUser(ctx context.Context, tenantID, userID string) error {
	return s.repo.PermanentDeleteTenantUser(ctx, tenantID, userID)
}

func (s *UserService) AdminUpdateTenantUserStatus(ctx context.Context, tenantID, userID string, status int) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.TenantID != tenantID {
		return ErrUserNotInTenant
	}

	return s.repo.UpdateStatus(ctx, userID, status)
}

func (s *UserService) AdminBatchUpdateTenantUserStatus(ctx context.Context, tenantID string, ids []string, status int) (int64, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("用户ID列表不能为空")
	}
	return s.repo.BatchUpdateTenantUserStatus(ctx, tenantID, ids, status)
}

func (s *UserService) AdminBatchDeleteTenantUsers(ctx context.Context, tenantID string, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("用户ID列表不能为空")
	}

	// 检查每个用户是否属于指定租户，并解除角色绑定
	for _, id := range ids {
		user, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return 0, err
		}
		if user.TenantID != tenantID {
			return 0, fmt.Errorf("用户 [%s] 不属于指定租户", user.Username)
		}

		// 解除用户所有角色绑定
		if err := s.userRoleRepo.DeleteByUserID(ctx, id); err != nil {
			return 0, fmt.Errorf("解除用户 [%s] 角色绑定失败: %w", user.Username, err)
		}
	}

	return s.repo.BatchDeleteTenantUsers(ctx, tenantID, ids)
}

// ============ 密码管理 ============
