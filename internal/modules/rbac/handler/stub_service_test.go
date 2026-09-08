package handler_test

import (
	"context"

	"meteorx/internal/modules/rbac/dto"
	"meteorx/internal/modules/rbac/model"
)

// stubRBACService 桩实现 handler.RBACService
type stubRBACService struct {
	Err        error
	Role       *model.Role
	Roles      []*model.Role
	Perm       *model.Permission
	Perms      []*model.Permission
	RolePerms  []*model.RolePermission
	UserRoles  []*model.UserRole
	UserIDList []string
	Affected   int64
	RoleCount  int64
	PermCount  int64
	MyPerms    int64

	GotID        string
	GotRoleID    string
	GotUserID    string
	GotPermID    string
	GotIDs       []string
	GotTenantID  string
	GotScope     string
	GotResource  string
	GotKeyword   string
	GotStatus    int
	GotPage      int
	GotPageSize  int
	GotCreateR   *dto.CreateRoleReq
	GotUpdateR   *dto.UpdateRoleReq
	GotCreateP   *dto.CreatePermissionReq
	GotUpdateP   *dto.UpdatePermissionReq
	GotBind      *dto.BindRolePermissionsReq
	GotAssign    *dto.AssignUserRolesReq
	GotBatchAsg  *dto.BatchAssignUserRolesReq
	GotBatchBind *dto.BatchBindRolesPermissionsReq
	GotBatchUnb  *dto.BatchUnbindRolesPermissionsReq
}

func (s *stubRBACService) CreateRole(_ context.Context, req dto.CreateRoleReq) (*model.Role, error) {
	s.GotCreateR = &req
	return s.Role, s.Err
}

func (s *stubRBACService) GetRole(_ context.Context, id string) (*model.Role, error) {
	s.GotID = id
	return s.Role, s.Err
}

func (s *stubRBACService) ListRoles(_ context.Context, tenantID string, page, pageSize int, keyword string) ([]*model.Role, int64, error) {
	s.GotTenantID, s.GotPage, s.GotPageSize, s.GotKeyword = tenantID, page, pageSize, keyword
	return s.Roles, s.Affected, s.Err
}

func (s *stubRBACService) ListRolesByScope(_ context.Context, scope string) ([]*model.Role, error) {
	s.GotScope = scope
	return s.Roles, s.Err
}

func (s *stubRBACService) ListSystemAdminRoles(_ context.Context) ([]*model.Role, error) {
	return s.Roles, s.Err
}

func (s *stubRBACService) UpdateRole(_ context.Context, id string, req dto.UpdateRoleReq) (*model.Role, error) {
	s.GotID, s.GotUpdateR = id, &req
	return s.Role, s.Err
}

func (s *stubRBACService) DeleteRole(_ context.Context, id string) error {
	s.GotID = id
	return s.Err
}

func (s *stubRBACService) PermanentDeleteRole(_ context.Context, id string) error {
	s.GotID = id
	return s.Err
}

func (s *stubRBACService) BatchPermanentDeleteRoles(_ context.Context, ids []string) (int64, error) {
	s.GotIDs = ids
	return s.Affected, s.Err
}

func (s *stubRBACService) UpdateRoleStatus(_ context.Context, id string, status int) error {
	s.GotID, s.GotStatus = id, status
	return s.Err
}

func (s *stubRBACService) BatchUpdateRoleStatus(_ context.Context, ids []string, status int) (int64, error) {
	s.GotIDs, s.GotStatus = ids, status
	return s.Affected, s.Err
}

func (s *stubRBACService) BatchDeleteRoles(_ context.Context, ids []string) (int64, error) {
	s.GotIDs = ids
	return s.Affected, s.Err
}

func (s *stubRBACService) ListDeletedRoles(_ context.Context, page, pageSize int, keyword string) ([]*model.Role, int64, error) {
	s.GotPage, s.GotPageSize, s.GotKeyword = page, pageSize, keyword
	return s.Roles, s.Affected, s.Err
}

func (s *stubRBACService) RestoreRole(_ context.Context, id string) error {
	s.GotID = id
	return s.Err
}

func (s *stubRBACService) CreatePermission(_ context.Context, req dto.CreatePermissionReq) (*model.Permission, error) {
	s.GotCreateP = &req
	return s.Perm, s.Err
}

func (s *stubRBACService) GetPermission(_ context.Context, id string) (*model.Permission, error) {
	s.GotID = id
	return s.Perm, s.Err
}

func (s *stubRBACService) ListPermissions(_ context.Context, page, pageSize int, resource, keyword string) ([]*model.Permission, int64, error) {
	s.GotPage, s.GotPageSize, s.GotResource, s.GotKeyword = page, pageSize, resource, keyword
	return s.Perms, s.Affected, s.Err
}

func (s *stubRBACService) UpdatePermission(_ context.Context, id string, req dto.UpdatePermissionReq) (*model.Permission, error) {
	s.GotID, s.GotUpdateP = id, &req
	return s.Perm, s.Err
}

func (s *stubRBACService) UpdatePermissionStatus(_ context.Context, id string, status int) error {
	s.GotID, s.GotStatus = id, status
	return s.Err
}

func (s *stubRBACService) BatchUpdatePermissionStatus(_ context.Context, ids []string, status int) (int64, error) {
	s.GotIDs, s.GotStatus = ids, status
	return s.Affected, s.Err
}

func (s *stubRBACService) DeletePermission(_ context.Context, id string) error {
	s.GotID = id
	return s.Err
}

func (s *stubRBACService) BatchDeletePermissions(_ context.Context, ids []string) (int64, error) {
	s.GotIDs = ids
	return s.Affected, s.Err
}

func (s *stubRBACService) BindRolePermissions(_ context.Context, roleID string, req dto.BindRolePermissionsReq) error {
	s.GotRoleID, s.GotBind = roleID, &req
	return s.Err
}

func (s *stubRBACService) GetRolePermissions(_ context.Context, roleID string) ([]*model.Permission, error) {
	s.GotRoleID = roleID
	return s.Perms, s.Err
}

func (s *stubRBACService) GetRolePermissionsWithResource(_ context.Context, roleID, resource string) ([]*model.Permission, error) {
	s.GotRoleID, s.GotResource = roleID, resource
	return s.Perms, s.Err
}

func (s *stubRBACService) UnbindRolePermission(_ context.Context, roleID, permissionID string) error {
	s.GotRoleID, s.GotPermID = roleID, permissionID
	return s.Err
}

func (s *stubRBACService) UnbindRolePermissions(_ context.Context, roleID string, permissionIDs []string) (int64, error) {
	s.GotRoleID, s.GotIDs = roleID, permissionIDs
	return s.Affected, s.Err
}

func (s *stubRBACService) BatchBindRolesPermissions(_ context.Context, req dto.BatchBindRolesPermissionsReq) (int64, error) {
	s.GotBatchBind = &req
	return s.Affected, s.Err
}

func (s *stubRBACService) BatchUnbindRolesPermissions(_ context.Context, req dto.BatchUnbindRolesPermissionsReq) (int64, error) {
	s.GotBatchUnb = &req
	return s.Affected, s.Err
}

func (s *stubRBACService) ListRolePermissions(_ context.Context, page, pageSize int, roleID, permissionID string) ([]*model.RolePermission, int64, error) {
	s.GotPage, s.GotPageSize, s.GotRoleID, s.GotPermID = page, pageSize, roleID, permissionID
	return s.RolePerms, s.Affected, s.Err
}

func (s *stubRBACService) AssignUserRoles(_ context.Context, userID string, req dto.AssignUserRolesReq) error {
	s.GotUserID, s.GotAssign = userID, &req
	return s.Err
}

func (s *stubRBACService) GetUserRoles(_ context.Context, userID string) ([]*model.Role, error) {
	s.GotUserID = userID
	return s.Roles, s.Err
}

func (s *stubRBACService) RemoveUserRole(_ context.Context, userID, roleID string) error {
	s.GotUserID, s.GotRoleID = userID, roleID
	return s.Err
}

func (s *stubRBACService) RemoveAllUserRoles(_ context.Context, userID string) error {
	s.GotUserID = userID
	return s.Err
}

func (s *stubRBACService) GetRoleUsers(_ context.Context, roleID string) ([]string, error) {
	s.GotRoleID = roleID
	return s.UserIDList, s.Err
}

func (s *stubRBACService) ListUserRoles(_ context.Context, page, pageSize int, userID, roleID string) ([]*model.UserRole, int64, error) {
	s.GotPage, s.GotPageSize, s.GotUserID, s.GotRoleID = page, pageSize, userID, roleID
	return s.UserRoles, s.Affected, s.Err
}

func (s *stubRBACService) BatchAssignUserRoles(_ context.Context, req dto.BatchAssignUserRolesReq) (int64, error) {
	s.GotBatchAsg = &req
	return s.Affected, s.Err
}

func (s *stubRBACService) CountRoles(_ context.Context, tenantID string) (int64, error) {
	s.GotTenantID = tenantID
	return s.RoleCount, s.Err
}

func (s *stubRBACService) CountPermissions(_ context.Context) (int64, error) {
	return s.PermCount, s.Err
}

func (s *stubRBACService) CountUserPermissions(_ context.Context, userID string) (int64, error) {
	s.GotUserID = userID
	return s.MyPerms, s.Err
}
