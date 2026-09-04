package service

import (
	"context"
	"errors"

	rbacModel "meteorx/internal/modules/rbac/model"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	userModel "meteorx/internal/modules/user/model"
	userRepo "meteorx/internal/modules/user/repository"
)

// 说明：通过嵌入接口仅覆写 auth 流程实际调用的方法，
// 其余未调用方法即便被意外调用也会在测试中暴露为 panic，便于及时补充。

// ---------- 内存 mock：UserRepository ----------

type mockUserRepo struct {
	userRepo.UserRepository
	users     map[string]*userModel.User
	byName    map[string]*userModel.User
	byEmail   map[string]*userModel.User
	createErr error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:   make(map[string]*userModel.User),
		byName:  make(map[string]*userModel.User),
		byEmail: make(map[string]*userModel.User),
	}
}

func (m *mockUserRepo) seed(u *userModel.User) {
	m.users[u.ID] = u
	m.byName[u.Username] = u
	if u.Email != "" {
		m.byEmail[u.Email] = u
	}
}

func (m *mockUserRepo) Create(_ context.Context, u *userModel.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.users[u.ID] = u
	m.byName[u.Username] = u
	if u.Email != "" {
		m.byEmail[u.Email] = u
	}
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
	if u, ok := m.byEmail[email]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepo) Update(_ context.Context, u *userModel.User) error {
	m.users[u.ID] = u
	m.byName[u.Username] = u
	if u.Email != "" {
		m.byEmail[u.Email] = u
	}
	return nil
}

var _ userRepo.UserRepository = (*mockUserRepo)(nil)

// ---------- 内存 mock：RoleRepository ----------

type mockRoleRepo struct {
	rbacRepo.RoleRepository
	roles  map[string]*rbacModel.Role
	byCode map[string]*rbacModel.Role
}

func newMockRoleRepo() *mockRoleRepo {
	return &mockRoleRepo{
		roles:  make(map[string]*rbacModel.Role),
		byCode: make(map[string]*rbacModel.Role),
	}
}

func (m *mockRoleRepo) seed(r *rbacModel.Role) {
	m.roles[r.ID] = r
	m.byCode[r.TenantID+"|"+r.Code] = r
}

func (m *mockRoleRepo) GetByCode(_ context.Context, tenantID, code string) (*rbacModel.Role, error) {
	if r, ok := m.byCode[tenantID+"|"+code]; ok {
		return r, nil
	}
	return nil, errors.New("role not found")
}

var _ rbacRepo.RoleRepository = (*mockRoleRepo)(nil)

// ---------- 内存 mock：UserRoleRepository ----------

type mockUserRoleRepo struct {
	rbacRepo.UserRoleRepository
	userRoles map[string][]string // userID -> roleID 列表
	userCodes map[string][]string // userID -> 角色编码列表
	assigned  []string            // AssignRoles 调用记录：userID
}

func newMockUserRoleRepo() *mockUserRoleRepo {
	return &mockUserRoleRepo{
		userRoles: make(map[string][]string),
		userCodes: make(map[string][]string),
	}
}

func (m *mockUserRoleRepo) AssignRoles(_ context.Context, userID string, roleIDs []string) error {
	m.userRoles[userID] = append([]string{}, roleIDs...)
	m.assigned = append(m.assigned, userID)
	return nil
}

func (m *mockUserRoleRepo) seedCodes(userID string, codes []string) {
	m.userCodes[userID] = append([]string{}, codes...)
}

func (m *mockUserRoleRepo) GetRoleIDsByUserID(_ context.Context, userID string) ([]string, error) {
	return m.userRoles[userID], nil
}

func (m *mockUserRoleRepo) GetRoleCodesByUserID(_ context.Context, userID string) ([]string, error) {
	return m.userCodes[userID], nil
}

var _ rbacRepo.UserRoleRepository = (*mockUserRoleRepo)(nil)

// ---------- 内存 mock：RolePermissionRepository ----------

type mockRolePermRepo struct {
	rbacRepo.RolePermissionRepository
	codesByRole map[string][]string
}

func newMockRolePermRepo() *mockRolePermRepo {
	return &mockRolePermRepo{
		codesByRole: make(map[string][]string),
	}
}

func (m *mockRolePermRepo) seed(roleID string, codes []string) {
	m.codesByRole[roleID] = append([]string{}, codes...)
}

func (m *mockRolePermRepo) GetPermissionCodesByRoleID(_ context.Context, roleID string) ([]string, error) {
	return m.codesByRole[roleID], nil
}

var _ rbacRepo.RolePermissionRepository = (*mockRolePermRepo)(nil)
