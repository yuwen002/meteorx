package service

import (
	"context"
	"errors"
	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/rbac/dto"
	"meteorx/internal/modules/rbac/model"
	"meteorx/internal/modules/rbac/repository"
	"meteorx/pkg/ulid"
	"time"
)

type RBACService struct {
	roleRepo           repository.RoleRepository
	permissionRepo     repository.PermissionRepository
	rolePermissionRepo repository.RolePermissionRepository
}

func NewRBACService(
	rr repository.RoleRepository,
	pr repository.PermissionRepository,
	rpr repository.RolePermissionRepository,
) *RBACService {
	return &RBACService{
		roleRepo:           rr,
		permissionRepo:     pr,
		rolePermissionRepo: rpr,
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
		ID:          ulid.Generate(),
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
	return s.roleRepo.BatchDelete(ctx, ids)
}

// --- Permission ---

func (s *RBACService) CreatePermission(ctx context.Context, req dto.CreatePermissionReq) (*model.Permission, error) {
	existing, _ := s.permissionRepo.GetByCode(ctx, req.Code)
	if existing != nil {
		return nil, errors.New("权限编码已存在")
	}

	permission := &model.Permission{
		ID:          ulid.Generate(),
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Resource:    req.Resource,
		Action:      req.Action,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.permissionRepo.Create(ctx, permission); err != nil {
		return nil, err
	}
	return permission, nil
}

func (s *RBACService) GetPermission(ctx context.Context, id string) (*model.Permission, error) {
	return s.permissionRepo.GetByID(ctx, id)
}

func (s *RBACService) ListPermissions(ctx context.Context, page, pageSize int, resource, keyword string) ([]*model.Permission, int64, error) {
	return s.permissionRepo.List(ctx, page, pageSize, resource, keyword)
}

func (s *RBACService) UpdatePermission(ctx context.Context, id string, req dto.UpdatePermissionReq) error {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	permission.Name = req.Name
	permission.Code = req.Code
	permission.Description = req.Description
	permission.Resource = req.Resource
	permission.Action = req.Action
	permission.UpdatedAt = time.Now()

	return s.permissionRepo.Update(ctx, permission)
}

func (s *RBACService) DeletePermission(ctx context.Context, id string) error {
	return s.permissionRepo.Delete(ctx, id)
}

// --- Role Permission ---

func (s *RBACService) BindRolePermissions(ctx context.Context, roleID string, req dto.BindRolePermissionsReq) error {
	// 验证角色是否存在
	_, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return errors.New("角色不存在")
	}
	return s.rolePermissionRepo.BindPermissions(ctx, roleID, req.PermissionIDs)
}

func (s *RBACService) GetRolePermissions(ctx context.Context, roleID string) ([]*model.Permission, error) {
	return s.rolePermissionRepo.GetPermissionsByRoleID(ctx, roleID)
}

func (s *RBACService) GetRolePermissionCodes(ctx context.Context, roleID string) ([]string, error) {
	return s.rolePermissionRepo.GetPermissionCodesByRoleID(ctx, roleID)
}

func (s *RBACService) UnbindRolePermission(ctx context.Context, roleID, permissionID string) error {
	return s.rolePermissionRepo.UnbindPermission(ctx, roleID, permissionID)
}
