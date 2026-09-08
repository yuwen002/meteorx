package service

import (
	"context"
	"errors"
	"time"

	rbacModel "meteorx/internal/modules/rbac/model"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	tenantModel "meteorx/internal/modules/tenant/model"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	userModel "meteorx/internal/modules/user/model"
	userRepo "meteorx/internal/modules/user/repository"
)

// ---------- 内存 mock：UserRepository ----------

type mockUserRepo struct {
	users     map[string]*userModel.User
	byName    map[string]*userModel.User
	createErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:  make(map[string]*userModel.User),
		byName: make(map[string]*userModel.User),
	}
}

func (m *mockUserRepo) seed(u *userModel.User) {
	m.users[u.ID] = u
	m.byName[u.Username] = u
}

func (m *mockUserRepo) Create(_ context.Context, u *userModel.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.users[u.ID] = u
	m.byName[u.Username] = u
	return nil
}

func (m *mockUserRepo) GetByUsername(_ context.Context, _ string, username string) (*userModel.User, error) {
	if u, ok := m.byName[username]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepo) GetByID(_ context.Context, id string) (*userModel.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*userModel.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepo) UsernameExists(_ context.Context, username string) (bool, error) {
	_, ok := m.byName[username]
	return ok, nil
}

func (m *mockUserRepo) ListByTenant(_ context.Context, tenantID string, _ int, _ int, _ string, _ *int) ([]*userModel.User, int64, error) {
	var out []*userModel.User
	for _, u := range m.users {
		if u.TenantID == tenantID {
			out = append(out, u)
		}
	}
	return out, int64(len(out)), nil
}

func (m *mockUserRepo) ListMasterAdmins(_ context.Context, _ int, _ int, _ string) ([]*userModel.User, int64, error) {
	var out []*userModel.User
	for _, u := range m.users {
		if u.IsMaster {
			out = append(out, u)
		}
	}
	return out, int64(len(out)), nil
}

func (m *mockUserRepo) ListAllTenantUsers(_ context.Context, _ int, _ int, _ string) ([]*userModel.User, int64, error) {
	var out []*userModel.User
	for _, u := range m.users {
		if !u.IsMaster {
			out = append(out, u)
		}
	}
	return out, int64(len(out)), nil
}

func (m *mockUserRepo) Update(_ context.Context, u *userModel.User) error {
	m.users[u.ID] = u
	m.byName[u.Username] = u
	return nil
}

func (m *mockUserRepo) Delete(_ context.Context, id string) error {
	if u, ok := m.users[id]; ok {
		delete(m.users, id)
		delete(m.byName, u.Username)
	}
	return nil
}

func (m *mockUserRepo) UpdateStatus(_ context.Context, id string, status int) error {
	if u, ok := m.users[id]; ok {
		u.Status = status
	}
	return nil
}

func (m *mockUserRepo) FindDeletedMasterAdmins(_ context.Context, _ int, _ int, _ string) ([]*userModel.User, int64, error) {
	return nil, 0, nil
}

func (m *mockUserRepo) RestoreMasterAdmin(_ context.Context, _ string) error {
	return nil
}

func (m *mockUserRepo) PermanentDeleteMasterAdmin(_ context.Context, id string) error {
	delete(m.users, id)
	return nil
}

func (m *mockUserRepo) BatchUpdateStatus(_ context.Context, _ []string, _ int) (int64, error) {
	return 0, nil
}

func (m *mockUserRepo) BatchDelete(_ context.Context, ids []string) (int64, error) {
	for _, id := range ids {
		delete(m.users, id)
	}
	return int64(len(ids)), nil
}

func (m *mockUserRepo) FindDeletedTenantUsers(_ context.Context, _ string, _ int, _ int, _ string) ([]*userModel.User, int64, error) {
	return nil, 0, nil
}

func (m *mockUserRepo) FindAllDeletedTenantUsers(_ context.Context, _ int, _ int, _ string) ([]*userModel.User, int64, error) {
	return nil, 0, nil
}

func (m *mockUserRepo) RestoreTenantUser(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockUserRepo) PermanentDeleteTenantUser(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockUserRepo) BatchUpdateTenantUserStatus(_ context.Context, _ string, _ []string, _ int) (int64, error) {
	return 0, nil
}

func (m *mockUserRepo) BatchDeleteTenantUsers(_ context.Context, _ string, _ []string) (int64, error) {
	return 0, nil
}

func (m *mockUserRepo) CountByTenant(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (m *mockUserRepo) CountAllUsers(_ context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

var _ userRepo.UserRepository = (*mockUserRepo)(nil)

// ---------- 内存 mock：TenantRepository ----------

type mockTenantRepo struct {
	tenants map[string]*tenantModel.Tenant
}

func newMockTenantRepo() *mockTenantRepo {
	return &mockTenantRepo{tenants: make(map[string]*tenantModel.Tenant)}
}

func (m *mockTenantRepo) GetByID(_ context.Context, id string) (*tenantModel.Tenant, error) {
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, errors.New("tenant not found")
}

func (m *mockTenantRepo) Create(_ context.Context, t *tenantModel.Tenant) error {
	m.tenants[t.ID] = t
	return nil
}

func (m *mockTenantRepo) GetByDomain(_ context.Context, _ string) (*tenantModel.Tenant, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTenantRepo) GetByName(_ context.Context, _ string) (*tenantModel.Tenant, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTenantRepo) CreateTenantWithAdmin(_ context.Context, _ *tenantModel.Tenant, _ *userModel.User, _ []string) error {
	return nil
}

func (m *mockTenantRepo) UpdateStatus(_ context.Context, _ string, _ int) error {
	return nil
}

func (m *mockTenantRepo) Update(_ context.Context, _ string, _ *tenantModel.Tenant) error {
	return nil
}

func (m *mockTenantRepo) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *mockTenantRepo) HardDelete(_ context.Context, _ string) error {
	return nil
}

func (m *mockTenantRepo) FindPage(_ context.Context, _ int, _ int, _ string, _ *int) ([]*tenantModel.Tenant, int64, error) {
	return nil, 0, nil
}

func (m *mockTenantRepo) BatchUpdateStatus(_ context.Context, _ []string, _ int) (int64, []string, error) {
	return 0, nil, nil
}

func (m *mockTenantRepo) BatchDelete(_ context.Context, _ []string) (int64, []string, error) {
	return 0, nil, nil
}

func (m *mockTenantRepo) FindDeleted(_ context.Context, _ int, _ int, _ string) ([]*tenantModel.Tenant, int64, error) {
	return nil, 0, nil
}

func (m *mockTenantRepo) Restore(_ context.Context, _ string) error {
	return nil
}

func (m *mockTenantRepo) CreateCancelRequest(_ context.Context, _ *tenantModel.CancelRequest) error {
	return nil
}

func (m *mockTenantRepo) GetCancelRequestByID(_ context.Context, _ string) (*tenantModel.CancelRequest, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTenantRepo) GetPendingCancelRequestByTenant(_ context.Context, _ string) (*tenantModel.CancelRequest, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTenantRepo) UpdateCancelRequest(_ context.Context, _ *tenantModel.CancelRequest) error {
	return nil
}

func (m *mockTenantRepo) FindCancelRequests(_ context.Context, _ int, _ int, _ int, _ string) ([]*tenantModel.CancelRequest, int64, error) {
	return nil, 0, nil
}

func (m *mockTenantRepo) FindApprovedDueCancelRequests(_ context.Context, _ time.Time) ([]*tenantModel.CancelRequest, error) {
	return nil, nil
}

var _ tenantRepo.TenantRepository = (*mockTenantRepo)(nil)

// ---------- 内存 mock：RoleRepository ----------

type mockRoleRepo struct {
	roles   map[string]*rbacModel.Role
	byCode  map[string]*rbacModel.Role
	codeErr error
}

func newMockRoleRepo() *mockRoleRepo {
	return &mockRoleRepo{
		roles:  make(map[string]*rbacModel.Role),
		byCode: make(map[string]*rbacModel.Role),
	}
}

func (m *mockRoleRepo) seed(r *rbacModel.Role) {
	m.roles[r.ID] = r
	if r.TenantID == "" {
		m.byCode[r.TenantID+"|"+r.Code] = r
	} else {
		m.byCode[r.TenantID+"|"+r.Code] = r
	}
	m.byCode["__any__|"+r.Code] = r
}

func (m *mockRoleRepo) Create(_ context.Context, r *rbacModel.Role) error {
	m.roles[r.ID] = r
	m.byCode[r.TenantID+"|"+r.Code] = r
	return nil
}

func (m *mockRoleRepo) GetByID(_ context.Context, id string) (*rbacModel.Role, error) {
	if r, ok := m.roles[id]; ok {
		return r, nil
	}
	return nil, errors.New("role not found")
}

func (m *mockRoleRepo) GetByCode(_ context.Context, tenantID, code string) (*rbacModel.Role, error) {
	if m.codeErr != nil {
		return nil, m.codeErr
	}
	if r, ok := m.byCode[tenantID+"|"+code]; ok {
		return r, nil
	}
	if r, ok := m.byCode["__any__|"+code]; ok {
		return r, nil
	}
	return nil, errors.New("role not found")
}

func (m *mockRoleRepo) List(_ context.Context, _ string, _ int, _ int, _ string) ([]*rbacModel.Role, int64, error) {
	var out []*rbacModel.Role
	for _, r := range m.roles {
		out = append(out, r)
	}
	return out, int64(len(out)), nil
}

func (m *mockRoleRepo) ListByScope(_ context.Context, _ string) ([]*rbacModel.Role, error) {
	return nil, nil
}

func (m *mockRoleRepo) ListSystemAdminRoles(_ context.Context) ([]*rbacModel.Role, error) {
	return nil, nil
}

func (m *mockRoleRepo) Update(_ context.Context, r *rbacModel.Role) error {
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

func (m *mockRoleRepo) FindDeleted(_ context.Context, _ int, _ int, _ string) ([]*rbacModel.Role, int64, error) {
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

var _ rbacRepo.RoleRepository = (*mockRoleRepo)(nil)

// ---------- 内存 mock：UserRoleRepository ----------

type mockUserRoleRepo struct {
	userRoles map[string][]string
	roleUsers map[string][]string
}

func newMockUserRoleRepo() *mockUserRoleRepo {
	return &mockUserRoleRepo{
		userRoles: make(map[string][]string),
		roleUsers: make(map[string][]string),
	}
}

func (m *mockUserRoleRepo) AssignRoles(_ context.Context, userID string, roleIDs []string) error {
	m.userRoles[userID] = append([]string{}, roleIDs...)
	for _, rid := range roleIDs {
		if !contains(m.roleUsers[rid], userID) {
			m.roleUsers[rid] = append(m.roleUsers[rid], userID)
		}
	}
	return nil
}

func (m *mockUserRoleRepo) GetRoleIDsByUserID(_ context.Context, userID string) ([]string, error) {
	return m.userRoles[userID], nil
}

func (m *mockUserRoleRepo) GetRoleCodesByUserID(_ context.Context, userID string) ([]string, error) {
	return m.userRoles[userID], nil
}

func (m *mockUserRoleRepo) BatchGetRoleIDsByUserIDs(_ context.Context, userIDs []string) (map[string][]string, error) {
	out := make(map[string][]string)
	for _, uid := range userIDs {
		out[uid] = m.userRoles[uid]
	}
	return out, nil
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

func (m *mockUserRoleRepo) CountByRoleID(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (m *mockUserRoleRepo) CountByUserID(_ context.Context, userID string) (int64, error) {
	return int64(len(m.userRoles[userID])), nil
}

func (m *mockUserRoleRepo) CheckUserExists(_ context.Context, _ string) error {
	return nil
}

func (m *mockUserRoleRepo) ListUserRoles(_ context.Context, _ int, _ int, _ string, _ string) ([]*rbacModel.UserRole, int64, error) {
	return nil, 0, nil
}

var _ rbacRepo.UserRoleRepository = (*mockUserRoleRepo)(nil)

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
