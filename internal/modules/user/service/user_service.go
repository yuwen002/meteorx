package service

import (
	"context"
	"errors"
	"fmt"
	planRepo "meteorx/internal/modules/plan/repository"
	planService "meteorx/internal/modules/plan/service"
	rbacModel "meteorx/internal/modules/rbac/model"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	tenantRepository "meteorx/internal/modules/tenant/repository"
	"meteorx/internal/modules/user/dto"
	"meteorx/internal/modules/user/model"
	"meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	"meteorx/pkg/idgen"
)

// 业务层哨兵错误：Handler 层通过 errors.Is 精确判定错误类型，
// 避免使用错误消息字符串比对（消息变更即失效）。
var (
	ErrUsernameExists     = errors.New("用户名已被使用")
	ErrUserNotSystemAdmin = errors.New("用户不是系统管理员")
	ErrUserNotInTenant    = errors.New("用户不属于指定租户")
	ErrWrongOldPassword   = errors.New("原密码错误")
)

type UserService struct {
	repo          repository.UserRepository
	tenantRepo    tenantRepository.TenantRepository
	roleRepo      rbacRepo.RoleRepository
	userRoleRepo  rbacRepo.UserRoleRepository
	quotaVerifier planRepo.QuotaVerifier
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

// SetQuotaVerifier 注入配额校验器（由 bootstrap 组装，避免循环依赖）
func (s *UserService) SetQuotaVerifier(v planRepo.QuotaVerifier) {
	s.quotaVerifier = v
}

// checkUserQuota 校验创建用户时是否超出套餐配额
func (s *UserService) checkUserQuota(ctx context.Context, tenantID string) error {
	if s.quotaVerifier == nil {
		return nil // 未配置配额校验则跳过
	}
	overLimit, users, limit, err := s.quotaVerifier.CheckUserLimit(ctx, tenantID)
	if err != nil {
		if errors.Is(err, planService.ErrPlanExpired) {
			return errors.New("套餐已到期，请联系平台管理员续费")
		}
		if errors.Is(err, planService.ErrUserLimitExceeded) {
			return fmt.Errorf("已达到套餐用户数上限（%d/%d），请联系平台管理员升级套餐", users, limit)
		}
		return err
	}
	if overLimit {
		return fmt.Errorf("已达到套餐用户数上限（%d/%d），请联系平台管理员升级套餐", users, limit)
	}
	return nil
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
		IsMaster:   user.IsMaster,
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

	// 查询角色信息（简化处理：逐个查询）
	roleIDToCode := make(map[string]string)
	roleIDToName := make(map[string]string)
	for _, roleID := range allRoleIDs {
		if _, ok := roleIDToCode[roleID]; ok {
			continue
		}
		role, err := s.roleRepo.GetByID(ctx, roleID)
		if err == nil {
			roleIDToCode[roleID] = role.Code
			roleIDToName[roleID] = role.Name
		}
	}

	var respList []*dto.UserResp
	for _, user := range users {
		roleIDs := userRoleIDsMap[user.ID]
		roleCodes := make([]string, 0, len(roleIDs))
		roleList := make([]dto.UserRoleInfo, 0, len(roleIDs))
		for _, rid := range roleIDs {
			code, hasCode := roleIDToCode[rid]
			name, hasName := roleIDToName[rid]
			if hasCode {
				roleCodes = append(roleCodes, code)
			}
			if hasCode && hasName {
				roleList = append(roleList, dto.UserRoleInfo{
					ID:   rid,
					Name: name,
					Code: code,
				})
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
			RoleList:   roleList,
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
		return nil, ErrUsernameExists
	}

	// 校验套餐配额
	if err := s.checkUserQuota(ctx, tenantID); err != nil {
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
	// 删除用户前先解除所有角色绑定
	if err := s.userRoleRepo.DeleteByUserID(ctx, userID); err != nil {
		return fmt.Errorf("解除用户角色绑定失败: %w", err)
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
