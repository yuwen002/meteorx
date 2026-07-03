package service

import (
	"context"
	"errors"
	"fmt"
	rbacModel "meteorx/internal/modules/rbac/model"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	tenantRepository "meteorx/internal/modules/tenant/repository"
	"meteorx/internal/modules/user/dto"
	"meteorx/internal/modules/user/model"
	"meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	ulpkg "meteorx/pkg/ulid"
	"strconv"
)

type UserService struct {
	repo         repository.UserRepository
	tenantRepo   tenantRepository.TenantRepository
	roleRepo     rbacRepo.RoleRepository
	userRoleRepo rbacRepo.UserRoleRepository
}

func NewUserService(
	repo repository.UserRepository,
	tenantRepo tenantRepository.TenantRepository,
	roleRepo rbacRepo.RoleRepository,
	userRoleRepo rbacRepo.UserRoleRepository,
) *UserService {
	return &UserService{
		repo:         repo,
		tenantRepo:   tenantRepo,
		roleRepo:     roleRepo,
		userRoleRepo: userRoleRepo,
	}
}

// validateRoleIDs 校验角色ID列表是否存在且启用，且 scope 匹配
func (s *UserService) validateRoleIDs(ctx context.Context, roleIDs []string, scope string) error {
	for _, roleID := range roleIDs {
		role, err := s.roleRepo.GetByID(ctx, roleID)
		if err != nil {
			return fmt.Errorf("角色 '%s' 不存在", roleID)
		}
		if role.Status == 0 {
			return fmt.Errorf("角色 '%s' 已禁用", role.Code)
		}
		if role.Scope != rbacModel.RoleScopeAll && role.Scope != scope {
			return fmt.Errorf("角色 '%s' 不允许在当前上下文分配", role.Code)
		}
	}
	return nil
}

// validateSystemAdminRoleID 校验角色ID是否为系统级别角色（scope 为 system 或 all）
func (s *UserService) validateSystemAdminRoleID(ctx context.Context, roleID string) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("角色 '%s' 不存在", roleID)
	}
	if role.Status == 0 {
		return fmt.Errorf("角色 '%s' 已禁用", role.Code)
	}
	// 必须是系统级别角色：scope 为 system 或 all
	if role.Scope != rbacModel.RoleScopeSystem && role.Scope != rbacModel.RoleScopeAll {
		return fmt.Errorf("角色 '%s' 不是系统级别角色，无法分配给系统管理员", role.Code)
	}
	return nil
}

// buildUserResp 构建用户响应（含角色信息）
func (s *UserService) buildUserResp(ctx context.Context, user *model.User) (*dto.UserResp, error) {
	tenant, _ := s.tenantRepo.GetByID(ctx, user.TenantID)
	tenantName := ""
	if tenant != nil {
		tenantName = tenant.Name
	}

	// 查询角色信息
	roleIDs, err := s.userRoleRepo.GetRoleIDsByUserID(ctx, user.ID)
	if err != nil {
		roleIDs = []string{}
	}
	roleCodes, err := s.userRoleRepo.GetRoleCodesByUserID(ctx, user.ID)
	if err != nil {
		roleCodes = []string{}
	}

	resp := &dto.UserResp{
		ID:         user.ID,
		TenantID:   user.TenantID,
		TenantName: tenantName,
		Username:   user.Username,
		Nickname:   user.Nickname,
		Email:      user.Email,
		Roles:      roleCodes,
		RoleIDs:    roleIDs,
		Status:     user.Status,
		CreatedAt:  user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:  user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if user.DeletedAt != nil {
		resp.DeletedAt = user.DeletedAt.Format("2006-01-02 15:04:05")
	}
	return resp, nil
}

// buildUserRespList 批量构建用户响应
func (s *UserService) buildUserRespList(ctx context.Context, users []*model.User) ([]*dto.UserResp, error) {
	if len(users) == 0 {
		return []*dto.UserResp{}, nil
	}

	// 批量查询用户-角色关联
	userIDs := make([]string, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	userRoleIDsMap, err := s.userRoleRepo.BatchGetRoleIDsByUserIDs(ctx, userIDs)
	if err != nil {
		userRoleIDsMap = make(map[string][]string)
	}

	// 构建角色ID -> 角色Code映射
	allRoleIDs := []string{}
	for _, ids := range userRoleIDsMap {
		allRoleIDs = append(allRoleIDs, ids...)
	}

	// 查询租户名称
	tenantNames := make(map[string]string)
	for _, u := range users {
		if _, ok := tenantNames[u.TenantID]; ok {
			continue
		}
		tenant, _ := s.tenantRepo.GetByID(ctx, u.TenantID)
		if tenant != nil {
			tenantNames[u.TenantID] = tenant.Name
		}
	}

	// 查询角色code（简化处理：逐个查询）
	roleIDToCode := make(map[string]string)
	for _, roleID := range allRoleIDs {
		if _, ok := roleIDToCode[roleID]; ok {
			continue
		}
		role, err := s.roleRepo.GetByID(ctx, roleID)
		if err == nil {
			roleIDToCode[roleID] = role.Code
		}
	}

	var respList []*dto.UserResp
	for _, user := range users {
		roleIDs := userRoleIDsMap[user.ID]
		roleCodes := make([]string, 0, len(roleIDs))
		for _, rid := range roleIDs {
			if code, ok := roleIDToCode[rid]; ok {
				roleCodes = append(roleCodes, code)
			}
		}

		resp := &dto.UserResp{
			ID:         user.ID,
			TenantID:   user.TenantID,
			TenantName: tenantNames[user.TenantID],
			Username:   user.Username,
			Nickname:   user.Nickname,
			Email:      user.Email,
			Roles:      roleCodes,
			RoleIDs:    roleIDs,
			Status:     user.Status,
			CreatedAt:  user.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:  user.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		if user.DeletedAt != nil {
			resp.DeletedAt = user.DeletedAt.Format("2006-01-02 15:04:05")
		}
		respList = append(respList, resp)
	}
	return respList, nil
}

// ============ 租户用户管理 ============

func (s *UserService) ListByTenant(ctx context.Context, tenantID string, page, pageSize int, keyword string, status *int) ([]*dto.UserResp, int64, error) {
	users, total, err := s.repo.ListByTenant(ctx, tenantID, page, pageSize, keyword, status)
	if err != nil {
		return nil, 0, err
	}
	resp, err := s.buildUserRespList(ctx, users)
	if err != nil {
		return nil, 0, err
	}
	return resp, total, nil
}

func (s *UserService) GetByID(ctx context.Context, userID string) (*dto.UserResp, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.buildUserResp(ctx, user)
}

func (s *UserService) Create(ctx context.Context, tenantID string, req dto.CreateUserReq) (*dto.UserResp, error) {
	exists, err := s.repo.UsernameExists(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("用户名已被使用")
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
		ID:       ulpkg.Generate(),
		TenantID: tenantID,
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

	// 分配角色
	if err := s.userRoleRepo.AssignRoles(ctx, user.ID, req.RoleIDs); err != nil {
		return nil, err
	}

	return s.buildUserResp(ctx, user)
}

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
	if req.Status != nil {
		user.Status = *req.Status
	}

	// 先更新用户基本信息
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	// 如果指定了角色ID，则更新角色
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

// AssignRoles 为用户分配角色（覆盖式更新）
func (s *UserService) AssignRoles(ctx context.Context, userID string, roleIDs []string) error {
	if len(roleIDs) == 0 {
		return errors.New("角色ID列表不能为空")
	}
	// 校验角色
	if err := s.validateRoleIDs(ctx, roleIDs, rbacModel.RoleScopeTenant); err != nil {
		return err
	}
	return s.userRoleRepo.AssignRoles(ctx, userID, roleIDs)
}

// GetUserRoles 获取用户的角色列表
func (s *UserService) GetUserRoles(ctx context.Context, userID string) ([]string, []string, error) {
	roleIDs, err := s.userRoleRepo.GetRoleIDsByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	roleCodes, err := s.userRoleRepo.GetRoleCodesByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	return roleIDs, roleCodes, nil
}

func (s *UserService) Delete(ctx context.Context, userID string) error {
	// 检查关联：是否有角色绑定
	roleCount, err := s.userRoleRepo.CountByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if roleCount > 0 {
		return errors.New("该用户已绑定 " + strconv.FormatInt(roleCount, 10) + " 个角色，请先解除用户角色绑定后再删除")
	}
	return s.repo.Delete(ctx, userID)
}

func (s *UserService) BelongsToTenant(ctx context.Context, userID, tenantID string) bool {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return false
	}
	return user.TenantID == tenantID
}

// ============ 系统管理员相关 ============

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
		return nil, fmt.Errorf("用户不是系统管理员")
	}
	return s.buildUserResp(ctx, user)
}

func (s *UserService) CreateMasterAdmin(ctx context.Context, req dto.CreateMasterAdminReq) (*dto.UserResp, error) {
	exists, err := s.repo.UsernameExists(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("用户名已被使用")
	}

	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:       ulpkg.Generate(),
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
		return fmt.Errorf("用户不是系统管理员")
	}

	// 检查是否为系统保护用户（初始管理员，不允许删除）
	if isProtectedUser(userID) {
		return fmt.Errorf("系统初始管理员 [%s] 不允许删除", user.Username)
	}

	// 检查关联：是否有角色绑定
	roleCount, err := s.userRoleRepo.CountByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if roleCount > 0 {
		return errors.New("该用户已绑定 " + strconv.FormatInt(roleCount, 10) + " 个角色，请先解除用户角色绑定后再删除")
	}

	return s.repo.Delete(ctx, userID)
}

func (s *UserService) UpdateMasterAdminStatus(ctx context.Context, userID string, status int) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.IsMaster {
		return fmt.Errorf("用户不是系统管理员")
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

	// 检查每个用户的关联
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

		roleCount, err := s.userRoleRepo.CountByUserID(ctx, id)
		if err != nil {
			return 0, err
		}
		if roleCount > 0 {
			return 0, errors.New("用户 [" + user.Username + "] 已绑定 " + strconv.FormatInt(roleCount, 10) + " 个角色，请先解除用户角色绑定后再删除")
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
		return nil, fmt.Errorf("用户名已被使用")
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
		ID:       ulpkg.Generate(),
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
		return nil, fmt.Errorf("用户不属于指定租户")
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
		return fmt.Errorf("用户不属于指定租户")
	}

	// 检查关联：是否有角色绑定
	roleCount, err := s.userRoleRepo.CountByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if roleCount > 0 {
		return errors.New("该用户已绑定 " + strconv.FormatInt(roleCount, 10) + " 个角色，请先解除用户角色绑定后再删除")
	}

	return s.repo.Delete(ctx, userID)
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

func (s *UserService) AdminUpdateTenantUserStatus(ctx context.Context, tenantID, userID string, status int) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.TenantID != tenantID {
		return fmt.Errorf("用户不属于指定租户")
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

	// 检查每个用户的关联
	for _, id := range ids {
		user, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return 0, err
		}
		if user.TenantID != tenantID {
			return 0, fmt.Errorf("用户 [%s] 不属于指定租户", user.Username)
		}

		roleCount, err := s.userRoleRepo.CountByUserID(ctx, id)
		if err != nil {
			return 0, err
		}
		if roleCount > 0 {
			return 0, errors.New("用户 [" + user.Username + "] 已绑定 " + strconv.FormatInt(roleCount, 10) + " 个角色，请先解除用户角色绑定后再删除")
		}
	}

	return s.repo.BatchDeleteTenantUsers(ctx, tenantID, ids)
}

// ============ 密码管理 ============

func (s *UserService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !crypto.CheckPassword(oldPassword, user.Password) {
		return fmt.Errorf("原密码错误")
	}

	hashedPassword, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	return s.repo.Update(ctx, user)
}
// CountByTenant 统计指定租户下的用户总数
func (s *UserService) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	return s.repo.CountByTenant(ctx, tenantID)
}

// CountAllUsers 统计所有用户总数（跨租户）
func (s *UserService) CountAllUsers(ctx context.Context) (int64, error) {
	return s.repo.CountAllUsers(ctx)
}