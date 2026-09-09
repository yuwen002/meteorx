package service

import (
	"context"
	"errors"
	"meteorx/internal/common/contextx"
	"meteorx/internal/middleware"
	"meteorx/internal/modules/rbac/dto"
	"meteorx/internal/modules/rbac/model"
	"meteorx/internal/modules/rbac/repository"
	"meteorx/pkg/idgen"
	"meteorx/pkg/logger"
	"strconv"
	"time"
)

type RBACService struct {
	roleRepo           repository.RoleRepository
	permissionRepo     repository.PermissionRepository
	rolePermissionRepo repository.RolePermissionRepository
	userRoleRepo       repository.UserRoleRepository
	auditLogger        *middleware.PermissionAuditLogger
	permCache          *middleware.PermissionCache
}

func NewRBACService(
	rr repository.RoleRepository,
	pr repository.PermissionRepository,
	rpr repository.RolePermissionRepository,
	urr repository.UserRoleRepository,
	auditLogger *middleware.PermissionAuditLogger,
	permCache *middleware.PermissionCache,
) *RBACService {
	return &RBACService{
		roleRepo:           rr,
		permissionRepo:     pr,
		rolePermissionRepo: rpr,
		userRoleRepo:       urr,
		auditLogger:        auditLogger,
		permCache:          permCache,
	}
}

// --- Role ---

func (s *RBACService) CreateRole(ctx context.Context, req dto.CreateRoleReq) (*model.Role, error) {
	// 如果未指定 tenant_id，默认为系统级角色
	if req.TenantID == "" {
		req.TenantID = contextx.SystemTenantID
	}

	// 检查 code 是否已存在
	existing, _ := s.roleRepo.GetByCode(ctx, req.TenantID, req.Code)
	if existing != nil {
		return nil, errors.New("角色编码已存在")
	}

	// Status 默认启用
	status := req.Status
	if status == 0 {
		status = model.RoleStatusEnabled
	}

	// Scope 默认 tenant
	scope := req.Scope
	if scope == "" {
		scope = model.RoleScopeTenant
	}

	role := &model.Role{
		ID:          idgen.New(),
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		TenantID:    req.TenantID,
		IsSystem:    req.IsSystem,
		Scope:       scope,
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *RBACService) GetRole(ctx context.Context, id string) (*model.Role, error) {
	return s.roleRepo.GetByID(ctx, id)
}

func (s *RBACService) ListRoles(ctx context.Context, tenantID string, page, pageSize int, keyword string) ([]*model.Role, int64, error) {
	return s.roleRepo.List(ctx, tenantID, page, pageSize, keyword)
}

// ListRolesByScope 根据作用域获取角色列表（用于下拉选择）
func (s *RBACService) ListRolesByScope(ctx context.Context, scope string) ([]*model.Role, error) {
	return s.roleRepo.ListByScope(ctx, scope)
}

// ListSystemAdminRoles 获取系统管理员角色列表（用于创建系统管理员时选择角色）
func (s *RBACService) ListSystemAdminRoles(ctx context.Context) ([]*model.Role, error) {
	return s.roleRepo.ListSystemAdminRoles(ctx)
}

func (s *RBACService) UpdateRole(ctx context.Context, id string, req dto.UpdateRoleReq) (*model.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role.IsSystem {
		return nil, errors.New("系统内置角色不可修改")
	}

	role.Name = req.Name
	role.Code = req.Code
	role.Description = req.Description
	role.Status = req.Status
	// 如果指定了 tenant_id 则更新租户归属
	if req.TenantID != "" {
		role.TenantID = req.TenantID
	}
	// 如果指定了 scope 则更新作用域
	if req.Scope != "" {
		role.Scope = req.Scope
	}
	role.UpdatedAt = time.Now()

	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *RBACService) DeleteRole(ctx context.Context, id string) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return errors.New("系统内置角色不可删除")
	}

	// 检查关联：是否有用户在使用该角色
	userCount, err := s.userRoleRepo.CountByRoleID(ctx, id)
	if err != nil {
		return err
	}
	if userCount > 0 {
		return errors.New("该角色已被 " + strconv.FormatInt(userCount, 10) + " 个用户使用，请先解除用户角色绑定后再删除")
	}

	// 检查关联：是否有权限绑定
	permCount, err := s.rolePermissionRepo.CountByRoleID(ctx, id)
	if err != nil {
		return err
	}
	if permCount > 0 {
		return errors.New("该角色已绑定 " + strconv.FormatInt(permCount, 10) + " 个权限，请先解除权限绑定后再删除")
	}

	return s.roleRepo.Delete(ctx, id)
}

// ListDeletedRoles 获取已软删除的角色列表
func (s *RBACService) ListDeletedRoles(ctx context.Context, page, pageSize int, keyword string) ([]*model.Role, int64, error) {
	return s.roleRepo.FindDeleted(ctx, page, pageSize, keyword)
}

// RestoreRole 恢复已软删除的角色
func (s *RBACService) RestoreRole(ctx context.Context, id string) error {
	return s.roleRepo.Restore(ctx, id)
}

// PermanentDeleteRole 永久删除角色（物理删除，不可恢复）
func (s *RBACService) PermanentDeleteRole(ctx context.Context, id string) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return errors.New("系统内置角色不可永久删除")
	}
	return s.roleRepo.PermanentDelete(ctx, id)
}

// BatchPermanentDeleteRoles 批量永久删除角色，返回实际删除的数量
func (s *RBACService) BatchPermanentDeleteRoles(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, errors.New("角色ID列表不能为空")
	}

	// 检查是否有系统角色
	for _, id := range ids {
		role, err := s.roleRepo.GetByID(ctx, id)
		if err != nil {
			return 0, err
		}
		if role.IsSystem {
			return 0, errors.New("角色 [" + role.Name + "] 是系统内置角色，不可永久删除")
		}
	}

	return s.roleRepo.BatchPermanentDelete(ctx, ids)
}

// UpdateRoleStatus 更改角色状态（启用/禁用）
func (s *RBACService) UpdateRoleStatus(ctx context.Context, id string, status int) error {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return errors.New("系统内置角色不可更改状态")
	}
	return s.roleRepo.UpdateStatus(ctx, id, status)
}

// BatchUpdateRoleStatus 批量更改角色状态，返回实际更新的数量
func (s *RBACService) BatchUpdateRoleStatus(ctx context.Context, ids []string, status int) (int64, error) {
	if len(ids) == 0 {
		return 0, errors.New("角色ID列表不能为空")
	}
	return s.roleRepo.BatchUpdateStatus(ctx, ids, status)
}

// BatchDeleteRoles 批量删除角色，返回实际删除的数量
func (s *RBACService) BatchDeleteRoles(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, errors.New("角色ID列表不能为空")
	}

	// 检查每个角色的关联
	for _, id := range ids {
		role, err := s.roleRepo.GetByID(ctx, id)
		if err != nil {
			return 0, err
		}
		if role.IsSystem {
			return 0, errors.New("角色 [" + role.Name + "] 是系统内置角色，不可删除")
		}

		userCount, err := s.userRoleRepo.CountByRoleID(ctx, id)
		if err != nil {
			return 0, err
		}
		if userCount > 0 {
			return 0, errors.New("角色 [" + role.Name + "] 已被 " + strconv.FormatInt(userCount, 10) + " 个用户使用，请先解除用户角色绑定后再删除")
		}

		permCount, err := s.rolePermissionRepo.CountByRoleID(ctx, id)
		if err != nil {
			return 0, err
		}
		if permCount > 0 {
			return 0, errors.New("角色 [" + role.Name + "] 已绑定 " + strconv.FormatInt(permCount, 10) + " 个权限，请先解除权限绑定后再删除")
		}
	}

	return s.roleRepo.BatchDelete(ctx, ids)
}

// --- Permission ---

func (s *RBACService) CreatePermission(ctx context.Context, req dto.CreatePermissionReq) (*model.Permission, error) {
	existing, _ := s.permissionRepo.GetByCode(ctx, req.Code)
	if existing != nil {
		return nil, errors.New("权限编码已存在")
	}

	status := req.Status
	if status == 0 {
		status = model.PermissionStatusEnabled
	}

	permission := &model.Permission{
		ID:          idgen.New(),
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Resource:    req.Resource,
		Action:      req.Action,
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.permissionRepo.Create(ctx, permission); err != nil {
		return nil, err
	}
	return permission, nil
}

// SeedPermissions 启动时批量注册权限到数据库
// 幂等：基于 code 字段判断，已存在则跳过，不存在则插入
// 返回 (新增数量, 总数量, error)
func (s *RBACService) SeedPermissions(ctx context.Context, defs []*model.Permission) (int, int, error) {
	now := time.Now()
	inserted := 0
	for _, p := range defs {
		existing, err := s.permissionRepo.GetByCode(ctx, p.Code)
		if err == nil && existing != nil {
			// 已存在则跳过（保持已有配置，不覆盖管理员可能修改过的信息）
			continue
		}

		// 不存在则插入
		permission := &model.Permission{
			ID:          idgen.New(),
			Name:        p.Name,
			Code:        p.Code,
			Description: p.Description,
			Resource:    p.Resource,
			Action:      p.Action,
			Status:      model.PermissionStatusEnabled,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.permissionRepo.Create(ctx, permission); err != nil {
			return inserted, len(defs), err
		}
		inserted++
	}
	return inserted, len(defs), nil
}

func (s *RBACService) GetPermission(ctx context.Context, id string) (*model.Permission, error) {
	return s.permissionRepo.GetByID(ctx, id)
}

func (s *RBACService) ListPermissions(ctx context.Context, page, pageSize int, resource, keyword string) ([]*model.Permission, int64, error) {
	return s.permissionRepo.List(ctx, page, pageSize, resource, keyword)
}

func (s *RBACService) UpdatePermission(ctx context.Context, id string, req dto.UpdatePermissionReq) (*model.Permission, error) {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	permission.Name = req.Name
	permission.Code = req.Code
	permission.Description = req.Description
	permission.Resource = req.Resource
	permission.Action = req.Action
	if req.Status != 0 {
		permission.Status = req.Status
	}
	permission.UpdatedAt = time.Now()

	if err := s.permissionRepo.Update(ctx, permission); err != nil {
		return nil, err
	}
	return permission, nil
}

func (s *RBACService) UpdatePermissionStatus(ctx context.Context, id string, status int) error {
	return s.permissionRepo.UpdateStatus(ctx, id, status)
}

func (s *RBACService) BatchUpdatePermissionStatus(ctx context.Context, ids []string, status int) (int64, error) {
	if len(ids) == 0 {
		return 0, errors.New("权限ID列表不能为空")
	}
	return s.permissionRepo.BatchUpdateStatus(ctx, ids, status)
}

func (s *RBACService) DeletePermission(ctx context.Context, id string) error {
	// 检查关联：是否有角色在使用该权限
	roleCount, err := s.rolePermissionRepo.CountByPermissionID(ctx, id)
	if err != nil {
		return err
	}
	if roleCount > 0 {
		return errors.New("该权限已被 " + strconv.FormatInt(roleCount, 10) + " 个角色使用，请先解除角色权限绑定后再删除")
	}

	return s.permissionRepo.Delete(ctx, id)
}

func (s *RBACService) BatchDeletePermissions(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, errors.New("权限ID列表不能为空")
	}

	// 检查每个权限的关联
	for _, id := range ids {
		perm, err := s.permissionRepo.GetByID(ctx, id)
		if err != nil {
			return 0, err
		}
		roleCount, err := s.rolePermissionRepo.CountByPermissionID(ctx, id)
		if err != nil {
			return 0, err
		}
		if roleCount > 0 {
			return 0, errors.New("权限 [" + perm.Name + "] 已被 " + strconv.FormatInt(roleCount, 10) + " 个角色使用，请先解除角色权限绑定后再删除")
		}
	}

	return s.permissionRepo.BatchDelete(ctx, ids)
}

// --- Role Permission ---

func (s *RBACService) BindRolePermissions(ctx context.Context, roleID string, req dto.BindRolePermissionsReq) error {
	// 验证角色是否存在
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return errors.New("角色不存在")
	}

	// 验证所有权限ID是否存在
	permissions := make([]*model.Permission, 0, len(req.PermissionIDs))
	for _, permissionID := range req.PermissionIDs {
		perm, err := s.permissionRepo.GetByID(ctx, permissionID)
		if err != nil {
			return errors.New("权限ID不存在: " + permissionID)
		}
		permissions = append(permissions, perm)
	}

	if err := s.rolePermissionRepo.BindPermissions(ctx, roleID, req.PermissionIDs); err != nil {
		return err
	}

	// 记录审计日志
	operatorID := logger.GetUserID(ctx)
	for _, perm := range permissions {
		s.auditLogger.LogPermissionGrant(ctx, roleID, operatorID, perm.Code, perm.Name)
	}

	// 失效所有用户的权限缓存（因为角色权限变更影响所有拥有该角色的用户）
	if s.permCache != nil {
		if err := s.permCache.InvalidateAllPermissions(ctx); err != nil {
			logger.Ctx(ctx).Warn("Failed to invalidate all permission caches", "error", err)
		}
	}

	return nil
}

func (s *RBACService) GetRolePermissions(ctx context.Context, roleID string) ([]*model.Permission, error) {
	return s.rolePermissionRepo.GetPermissionsByRoleID(ctx, roleID)
}

func (s *RBACService) GetRolePermissionsWithResource(ctx context.Context, roleID string, resource string) ([]*model.Permission, error) {
	return s.rolePermissionRepo.GetPermissionsByRoleIDWithResource(ctx, roleID, resource)
}

func (s *RBACService) GetRolePermissionCodes(ctx context.Context, roleID string) ([]string, error) {
	return s.rolePermissionRepo.GetPermissionCodesByRoleID(ctx, roleID)
}

func (s *RBACService) UnbindRolePermission(ctx context.Context, roleID, permissionID string) error {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return errors.New("角色不存在")
	}
	perm, err := s.permissionRepo.GetByID(ctx, permissionID)
	if err != nil {
		return errors.New("权限不存在")
	}
	if err := s.rolePermissionRepo.UnbindPermission(ctx, roleID, permissionID); err != nil {
		return err
	}

	// 记录审计日志
	operatorID := logger.GetUserID(ctx)
	s.auditLogger.LogPermissionRevoke(ctx, roleID, operatorID, perm.Code, perm.Name)

	// 失效所有用户的权限缓存
	if s.permCache != nil {
		if err := s.permCache.InvalidateAllPermissions(ctx); err != nil {
			logger.Ctx(ctx).Warn("Failed to invalidate all permission caches", "error", err)
		}
	}

	return nil
}

func (s *RBACService) UnbindRolePermissions(ctx context.Context, roleID string, permissionIDs []string) (int64, error) {
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return 0, errors.New("角色不存在")
	}

	permissions := make([]*model.Permission, 0, len(permissionIDs))
	for _, permissionID := range permissionIDs {
		perm, err := s.permissionRepo.GetByID(ctx, permissionID)
		if err != nil {
			return 0, errors.New("权限不存在: " + permissionID)
		}
		permissions = append(permissions, perm)
	}

	count, err := s.rolePermissionRepo.BatchUnbindPermissions(ctx, []string{roleID}, permissionIDs)
	if err != nil {
		return count, err
	}

	// 记录审计日志
	operatorID := logger.GetUserID(ctx)
	for _, perm := range permissions {
		s.auditLogger.LogPermissionRevoke(ctx, roleID, operatorID, perm.Code, perm.Name)
	}

	// 失效所有用户的权限缓存
	if s.permCache != nil {
		if err := s.permCache.InvalidateAllPermissions(ctx); err != nil {
			logger.Ctx(ctx).Warn("Failed to invalidate all permission caches", "error", err)
		}
	}

	return count, nil
}

func (s *RBACService) BatchBindRolesPermissions(ctx context.Context, req dto.BatchBindRolesPermissionsReq) (int64, error) {
	if len(req.RoleIDs) == 0 {
		return 0, errors.New("角色ID列表不能为空")
	}
	if len(req.PermissionIDs) == 0 {
		return 0, errors.New("权限ID列表不能为空")
	}

	// 验证所有权限ID是否存在
	permissions := make([]*model.Permission, 0, len(req.PermissionIDs))
	for _, permissionID := range req.PermissionIDs {
		perm, err := s.permissionRepo.GetByID(ctx, permissionID)
		if err != nil {
			return 0, errors.New("权限ID不存在: " + permissionID)
		}
		permissions = append(permissions, perm)
	}

	count, err := s.rolePermissionRepo.BatchBindPermissions(ctx, req.RoleIDs, req.PermissionIDs)
	if err != nil {
		return count, err
	}

	// 记录审计日志
	operatorID := logger.GetUserID(ctx)
	for _, roleID := range req.RoleIDs {
		for _, perm := range permissions {
			s.auditLogger.LogPermissionGrant(ctx, roleID, operatorID, perm.Code, perm.Name)
		}
	}

	// 失效所有用户的权限缓存
	if s.permCache != nil {
		if err := s.permCache.InvalidateAllPermissions(ctx); err != nil {
			logger.Ctx(ctx).Warn("Failed to invalidate all permission caches", "error", err)
		}
	}

	return count, nil
}

func (s *RBACService) BatchUnbindRolesPermissions(ctx context.Context, req dto.BatchUnbindRolesPermissionsReq) (int64, error) {
	if len(req.RoleIDs) == 0 {
		return 0, errors.New("角色ID列表不能为空")
	}
	if len(req.PermissionIDs) == 0 {
		return 0, errors.New("权限ID列表不能为空")
	}

	count, err := s.rolePermissionRepo.BatchUnbindPermissions(ctx, req.RoleIDs, req.PermissionIDs)
	if err != nil {
		return count, err
	}

	// 记录审计日志
	operatorID := logger.GetUserID(ctx)
	for _, roleID := range req.RoleIDs {
		for _, permissionID := range req.PermissionIDs {
			s.auditLogger.LogPermissionRevoke(ctx, roleID, operatorID, permissionID, "")
		}
	}

	// 失效所有用户的权限缓存
	if s.permCache != nil {
		if err := s.permCache.InvalidateAllPermissions(ctx); err != nil {
			logger.Ctx(ctx).Warn("Failed to invalidate all permission caches", "error", err)
		}
	}

	return count, nil
}

func (s *RBACService) ListRolePermissions(ctx context.Context, page, pageSize int, roleID, permissionID string) ([]*model.RolePermission, int64, error) {
	return s.rolePermissionRepo.List(ctx, page, pageSize, roleID, permissionID)
}

// --- User Role ---

// GetUserPermissionCodes 获取用户的所有权限code集合（合并所有角色的权限）
func (s *RBACService) GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error) {
	roleIDs, err := s.userRoleRepo.GetRoleIDsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		return []string{}, nil
	}

	// 去重合并所有角色的权限code
	codeSet := make(map[string]bool)
	for _, roleID := range roleIDs {
		codes, err := s.rolePermissionRepo.GetPermissionCodesByRoleID(ctx, roleID)
		if err != nil {
			continue
		}
		for _, code := range codes {
			codeSet[code] = true
		}
	}

	result := make([]string, 0, len(codeSet))
	for code := range codeSet {
		result = append(result, code)
	}
	return result, nil
}

// AssignUserRoles 为用户分配角色
func (s *RBACService) AssignUserRoles(ctx context.Context, userID string, req dto.AssignUserRolesReq) error {
	if len(req.RoleIDs) == 0 {
		return errors.New("角色ID列表不能为空")
	}

	if err := s.userRoleRepo.CheckUserExists(ctx, userID); err != nil {
		return err
	}

	// 验证所有角色ID是否存在
	roles := make([]*model.Role, 0, len(req.RoleIDs))
	for _, roleID := range req.RoleIDs {
		role, err := s.roleRepo.GetByID(ctx, roleID)
		if err != nil {
			return errors.New("角色ID不存在: " + roleID)
		}
		roles = append(roles, role)
	}

	if err := s.userRoleRepo.AssignRoles(ctx, userID, req.RoleIDs); err != nil {
		return err
	}

	// 记录审计日志
	operatorID := logger.GetUserID(ctx)
	for _, role := range roles {
		s.auditLogger.LogRoleAssignment(ctx, userID, operatorID, role.ID, role.Name)
	}

	// 失效权限缓存
	if s.permCache != nil {
		if err := s.permCache.InvalidateUserPermissions(ctx, userID); err != nil {
			logger.Ctx(ctx).Warn("Failed to invalidate permission cache", "user_id", userID, "error", err)
		}
	}

	return nil
}

// GetUserRoles 获取用户的角色列表
func (s *RBACService) GetUserRoles(ctx context.Context, userID string) ([]*model.Role, error) {
	if err := s.userRoleRepo.CheckUserExists(ctx, userID); err != nil {
		return nil, err
	}

	roleIDs, err := s.userRoleRepo.GetRoleIDsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		return []*model.Role{}, nil
	}

	roles := make([]*model.Role, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		role, err := s.roleRepo.GetByID(ctx, roleID)
		if err != nil {
			continue
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// RemoveUserRole 删除用户的单个角色
func (s *RBACService) RemoveUserRole(ctx context.Context, userID, roleID string) error {
	if err := s.userRoleRepo.CheckUserExists(ctx, userID); err != nil {
		return err
	}
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return errors.New("角色ID不存在: " + roleID)
	}
	if err := s.userRoleRepo.DeleteByUserIDAndRoleID(ctx, userID, roleID); err != nil {
		return err
	}

	// 记录审计日志
	operatorID := logger.GetUserID(ctx)
	s.auditLogger.LogRoleRemoval(ctx, userID, operatorID, role.ID, role.Name)

	// 失效权限缓存
	if s.permCache != nil {
		if err := s.permCache.InvalidateUserPermissions(ctx, userID); err != nil {
			logger.Ctx(ctx).Warn("Failed to invalidate permission cache", "user_id", userID, "error", err)
		}
	}

	return nil
}

// RemoveAllUserRoles 删除用户的所有角色
func (s *RBACService) RemoveAllUserRoles(ctx context.Context, userID string) error {
	if err := s.userRoleRepo.CheckUserExists(ctx, userID); err != nil {
		return err
	}

	// 获取用户当前角色，用于审计日志
	roles, _ := s.GetUserRoles(ctx, userID)

	if err := s.userRoleRepo.DeleteByUserID(ctx, userID); err != nil {
		return err
	}

	// 记录审计日志
	operatorID := logger.GetUserID(ctx)
	for _, role := range roles {
		s.auditLogger.LogRoleRemoval(ctx, userID, operatorID, role.ID, role.Name)
	}

	// 失效权限缓存
	if s.permCache != nil {
		if err := s.permCache.InvalidateUserPermissions(ctx, userID); err != nil {
			logger.Ctx(ctx).Warn("Failed to invalidate permission cache", "user_id", userID, "error", err)
		}
	}

	return nil
}

// GetRoleUsers 获取拥有某角色的用户列表
func (s *RBACService) GetRoleUsers(ctx context.Context, roleID string) ([]string, error) {
	if _, err := s.roleRepo.GetByID(ctx, roleID); err != nil {
		return nil, errors.New("角色ID不存在: " + roleID)
	}
	return s.userRoleRepo.GetUserIDsByRoleID(ctx, roleID)
}

// ListUserRoles 分页查询用户-角色关系列表，支持按 user_id / role_id 过滤
func (s *RBACService) ListUserRoles(ctx context.Context, page, pageSize int, userID, roleID string) ([]*model.UserRole, int64, error) {
	return s.userRoleRepo.ListUserRoles(ctx, page, pageSize, userID, roleID)
}

// BatchAssignUserRoles 批量为用户分配角色
func (s *RBACService) BatchAssignUserRoles(ctx context.Context, req dto.BatchAssignUserRolesReq) (int64, error) {
	if len(req.UserRoleAssignments) == 0 {
		return 0, errors.New("用户角色分配列表不能为空")
	}

	// 收集所有需要验证的角色ID
	allRoleIDs := make(map[string]bool)
	allUserIDs := make(map[string]bool)
	for _, assignment := range req.UserRoleAssignments {
		allUserIDs[assignment.UserID] = true
		for _, roleID := range assignment.RoleIDs {
			allRoleIDs[roleID] = true
		}
	}

	// 验证所有角色ID是否存在
	roleNames := make(map[string]string)
	for roleID := range allRoleIDs {
		role, err := s.roleRepo.GetByID(ctx, roleID)
		if err != nil {
			return 0, errors.New("角色ID不存在: " + roleID)
		}
		roleNames[roleID] = role.Name
	}
	// 验证所有用户ID是否存在
	for userID := range allUserIDs {
		if err := s.userRoleRepo.CheckUserExists(ctx, userID); err != nil {
			return 0, err
		}
	}

	operatorID := logger.GetUserID(ctx)
	var count int64
	for _, assignment := range req.UserRoleAssignments {
		if len(assignment.RoleIDs) == 0 {
			continue
		}
		if err := s.userRoleRepo.AssignRoles(ctx, assignment.UserID, assignment.RoleIDs); err != nil {
			return count, err
		}

		// 记录审计日志
		for _, roleID := range assignment.RoleIDs {
			if name, ok := roleNames[roleID]; ok {
				s.auditLogger.LogRoleAssignment(ctx, assignment.UserID, operatorID, roleID, name)
			}
		}

		// 失效权限缓存
		if s.permCache != nil {
			if err := s.permCache.InvalidateUserPermissions(ctx, assignment.UserID); err != nil {
				logger.Ctx(ctx).Warn("Failed to invalidate permission cache", "user_id", assignment.UserID, "error", err)
			}
		}

		count++
	}
	return count, nil
}

// --- Dashboard Statistics ---

// CountRoles 统计角色总数（按租户）
func (s *RBACService) CountRoles(ctx context.Context, tenantID string) (int64, error) {
	return s.roleRepo.Count(ctx, tenantID)
}

// CountPermissions 统计权限总数
func (s *RBACService) CountPermissions(ctx context.Context) (int64, error) {
	return s.permissionRepo.Count(ctx)
}

// CountUserPermissions 统计当前用户拥有的权限总数（去重）
func (s *RBACService) CountUserPermissions(ctx context.Context, userID string) (int64, error) {
	if userID == "" {
		return 0, nil
	}
	// 1. 获取用户所有角色
	roleIDs, err := s.userRoleRepo.GetRoleIDsByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	if len(roleIDs) == 0 {
		return 0, nil
	}
	// 2. 汇总所有角色的权限 code，去重
	codeSet := make(map[string]bool)
	for _, roleID := range roleIDs {
		codes, err := s.rolePermissionRepo.GetPermissionCodesByRoleID(ctx, roleID)
		if err != nil {
			continue
		}
		for _, code := range codes {
			codeSet[code] = true
		}
	}
	return int64(len(codeSet)), nil
}