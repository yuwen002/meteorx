package service

import (
	"context"
	"errors"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/rbac/model"
	"meteorx/internal/modules/rbac/repository"
)

// codeKey 将租户编码为角色唯一 key；空租户视为系统级
func codeKey(tenantID, code string) string {
	if tenantID == "" {
		tenantID = contextx.SystemTenantID
	}
	return tenantID + "|" + code
}

// ---------- mock: RoleRepository ----------

type mockRoleRepo struct {
	roles       map[string]*model.Role
	byCode      map[string]*model.Role
	countBy     map[string]int64
	permCountBy map[string]int64
}

func newMockRoleRepo() *mockRoleRepo {
	return &mockRoleRepo{
		roles:       make(map[string]*model.Role),
		byCode:      make(map[string]*model.Role),
		countBy:     make(map[string]int64),
		permCountBy: make(map[string]int64),
	}
}

func (m *mockRoleRepo) seed(r *model.Role) {
	m.roles[r.ID] = r
	m.byCode[codeKey(r.TenantID, r.Code)] = r
}

func (m *mockRoleRepo) Create(_ context.Context, r *model.Role) error {
	m.roles[r.ID] = r
	m.byCode[codeKey(r.TenantID, r.Code)] = r
	return nil
}

func (m *mockRoleRepo) GetByID(_ context.Context, id string) (*model.Role, error) {
	if r, ok := m.roles[id]; ok {
		return r, nil
	}
	return nil, errors.New("role not found")
}

func (m *mockRoleRepo) GetByCode(_ context.Context, tenantID, code string) (*model.Role, error) {
	if r, ok := m.byCode[codeKey(tenantID, code)]; ok {
		return r, nil
	}
	return nil, errors.New("role not found")
}

func (m *mockRoleRepo) List(_ context.Context, _ string, _ int, _ int, _ string) ([]*model.Role, int64, error) {
	return nil, 0, nil
}

func (m *mockRoleRepo) ListByScope(_ context.Context, _ string) ([]*model.Role, error) {
	return nil, nil
}

func (m *mockRoleRepo) ListSystemAdminRoles(_ context.Context) ([]*model.Role, error) {
	return nil, nil
}

func (m *mockRoleRepo) Update(_ context.Context, r *model.Role) error {
	m.roles[r.ID] = r
	return nil
}

func (m *mockRoleRepo) UpdateStatus(_ context.Context, id string, status int) error {
	if r, ok := m.roles[id]; ok {
		r.Status = status
	}
	return nil
}

func (m *mockRoleRepo) BatchUpdateStatus(_ context.Context, _ []string, _ int) (int64, error) {
	return 0, nil
}

func (m *mockRoleRepo) Delete(_ context.Context, id string) error {
	delete(m.roles, id)
	return nil
}

func (m *mockRoleRepo) BatchDelete(_ context.Context, _ []string) (int64, error) {
	return 0, nil
}

func (m *mockRoleRepo) FindDeleted(_ context.Context, _ int, _ int, _ string) ([]*model.Role, int64, error) {
	return nil, 0, nil
}

func (m *mockRoleRepo) Restore(_ context.Context, _ string) error {
	return nil
}

func (m *mockRoleRepo) PermanentDelete(_ context.Context, id string) error {
	delete(m.roles, id)
	return nil
}

func (m *mockRoleRepo) BatchPermanentDelete(_ context.Context, ids []string) (int64, error) {
	deleted := int64(0)
	for _, id := range ids {
		if _, ok := m.roles[id]; ok {
			delete(m.roles, id)
			deleted++
		}
	}
	return deleted, nil
}

func (m *mockRoleRepo) Count(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

var _ repository.RoleRepository = (*mockRoleRepo)(nil)

// ---------- mock: PermissionRepository ----------

type mockPermissionRepo struct {
	perms  map[string]*model.Permission
	byCode map[string]*model.Permission
}

func newMockPermissionRepo() *mockPermissionRepo {
	return &mockPermissionRepo{
		perms:  make(map[string]*model.Permission),
		byCode: make(map[string]*model.Permission),
	}
}

func (m *mockPermissionRepo) seed(p *model.Permission) {
	m.perms[p.ID] = p
	m.byCode[p.Code] = p
}

func (m *mockPermissionRepo) Create(_ context.Context, p *model.Permission) error {
	m.perms[p.ID] = p
	m.byCode[p.Code] = p
	return nil
}

func (m *mockPermissionRepo) GetByID(_ context.Context, id string) (*model.Permission, error) {
	if p, ok := m.perms[id]; ok {
		return p, nil
	}
	return nil, errors.New("permission not found")
}

func (m *mockPermissionRepo) GetByCode(_ context.Context, code string) (*model.Permission, error) {
	if p, ok := m.byCode[code]; ok {
		return p, nil
	}
	return nil, errors.New("permission not found")
}

func (m *mockPermissionRepo) List(_ context.Context, _ int, _ int, _ string, _ string) ([]*model.Permission, int64, error) {
	return nil, 0, nil
}

func (m *mockPermissionRepo) Update(_ context.Context, p *model.Permission) error {
	m.perms[p.ID] = p
	return nil
}

func (m *mockPermissionRepo) UpdateStatus(_ context.Context, id string, status int) error {
	if p, ok := m.perms[id]; ok {
		p.Status = status
	}
	return nil
}

func (m *mockPermissionRepo) BatchUpdateStatus(_ context.Context, _ []string, _ int) (int64, error) {
	return 0, nil
}

func (m *mockPermissionRepo) Delete(_ context.Context, id string) error {
	delete(m.perms, id)
	return nil
}

func (m *mockPermissionRepo) BatchDelete(_ context.Context, _ []string) (int64, error) {
	return 0, nil
}

func (m *mockPermissionRepo) Count(_ context.Context) (int64, error) {
	return int64(len(m.perms)), nil
}

var _ repository.PermissionRepository = (*mockPermissionRepo)(nil)

// ---------- mock: RolePermissionRepository ----------

type mockRolePermRepo struct {
	rolePerms   map[string][]string
	codesByRole map[string][]string
	countByRole map[string]int64
	countByPerm map[string]int64
}

func newMockRolePermRepo() *mockRolePermRepo {
	return &mockRolePermRepo{
		rolePerms:   make(map[string][]string),
		codesByRole: make(map[string][]string),
		countByRole: make(map[string]int64),
		countByPerm: make(map[string]int64),
	}
}

func (m *mockRolePermRepo) BindPermissions(_ context.Context, roleID string, permissionIDs []string) error {
	m.rolePerms[roleID] = append([]string{}, permissionIDs...)
	return nil
}

func (m *mockRolePermRepo) GetPermissionsByRoleID(_ context.Context, _ string) ([]*model.Permission, error) {
	return nil, nil
}

func (m *mockRolePermRepo) GetPermissionsByRoleIDWithResource(_ context.Context, _ string, _ string) ([]*model.Permission, error) {
	return nil, nil
}

func (m *mockRolePermRepo) GetPermissionCodesByRoleID(_ context.Context, roleID string) ([]string, error) {
	return m.codesByRole[roleID], nil
}

func (m *mockRolePermRepo) UnbindPermission(_ context.Context, roleID, permissionID string) error {
	return nil
}

func (m *mockRolePermRepo) BatchBindPermissions(_ context.Context, _ []string, _ []string) (int64, error) {
	return 0, nil
}

func (m *mockRolePermRepo) BatchUnbindPermissions(_ context.Context, _ []string, _ []string) (int64, error) {
	return 0, nil
}

func (m *mockRolePermRepo) List(_ context.Context, _ int, _ int, _ string, _ string) ([]*model.RolePermission, int64, error) {
	return nil, 0, nil
}

func (m *mockRolePermRepo) CountByRoleID(_ context.Context, roleID string) (int64, error) {
	return m.countByRole[roleID], nil
}

func (m *mockRolePermRepo) CountByPermissionID(_ context.Context, permissionID string) (int64, error) {
	return m.countByPerm[permissionID], nil
}

var _ repository.RolePermissionRepository = (*mockRolePermRepo)(nil)

// ---------- mock: UserRoleRepository ----------

type mockUserRoleRepo struct {
	userRoles  map[string][]string
	roleUsers  map[string][]string
	userExists map[string]bool
}

func newMockUserRoleRepo() *mockUserRoleRepo {
	return &mockUserRoleRepo{
		userRoles:  make(map[string][]string),
		roleUsers:  make(map[string][]string),
		userExists: make(map[string]bool),
	}
}

func (m *mockUserRoleRepo) AssignRoles(_ context.Context, userID string, roleIDs []string) error {
	m.userRoles[userID] = append([]string{}, roleIDs...)
	for _, rid := range roleIDs {
		m.roleUsers[rid] = append(m.roleUsers[rid], userID)
	}
	return nil
}

func (m *mockUserRoleRepo) GetRoleIDsByUserID(_ context.Context, userID string) ([]string, error) {
	return m.userRoles[userID], nil
}

func (m *mockUserRoleRepo) GetRoleCodesByUserID(_ context.Context, userID string) ([]string, error) {
	return m.userRoles[userID], nil
}

func (m *mockUserRoleRepo) BatchGetRoleIDsByUserIDs(_ context.Context, _ []string) (map[string][]string, error) {
	return nil, nil
}

func (m *mockUserRoleRepo) DeleteByUserID(_ context.Context, userID string) error {
	delete(m.userRoles, userID)
	return nil
}

func (m *mockUserRoleRepo) DeleteByUserIDAndRoleID(_ context.Context, userID, roleID string) error {
	var out []string
	for _, rid := range m.userRoles[userID] {
		if rid != roleID {
			out = append(out, rid)
		}
	}
	m.userRoles[userID] = out
	return nil
}

func (m *mockUserRoleRepo) GetUserIDsByRoleID(_ context.Context, roleID string) ([]string, error) {
	return m.roleUsers[roleID], nil
}

func (m *mockUserRoleRepo) CountByRoleID(_ context.Context, roleID string) (int64, error) {
	return int64(len(m.roleUsers[roleID])), nil
}

func (m *mockUserRoleRepo) CountByUserID(_ context.Context, userID string) (int64, error) {
	return int64(len(m.userRoles[userID])), nil
}

func (m *mockUserRoleRepo) CheckUserExists(_ context.Context, userID string) error {
	if m.userExists[userID] {
		return nil
	}
	if _, ok := m.userRoles[userID]; ok {
		return nil
	}
	return errors.New("user not found")
}

func (m *mockUserRoleRepo) ListUserRoles(_ context.Context, _ int, _ int, _ string, _ string) ([]*model.UserRole, int64, error) {
	return nil, 0, nil
}

var _ repository.UserRoleRepository = (*mockUserRoleRepo)(nil)
