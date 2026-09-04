package service

import (
	"context"
	"errors"
	"sort"
	"time"

	planModel "meteorx/internal/modules/plan/model"
	planRepo "meteorx/internal/modules/plan/repository"
	rbacModel "meteorx/internal/modules/rbac/model"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	tenantModel "meteorx/internal/modules/tenant/model"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	userModel "meteorx/internal/modules/user/model"
	userRepo "meteorx/internal/modules/user/repository"
)

// ---------- 内存 mock：TenantRepository ----------

type mockTenantRepo struct {
	tenantRepo.TenantRepository
	tenants        map[string]*tenantModel.Tenant
	cancelRequests map[string]*tenantModel.CancelRequest
}

func newMockTenantRepo() *mockTenantRepo {
	return &mockTenantRepo{
		tenants:        make(map[string]*tenantModel.Tenant),
		cancelRequests: make(map[string]*tenantModel.CancelRequest),
	}
}

func (m *mockTenantRepo) seed(t *tenantModel.Tenant) {
	m.tenants[t.ID] = t
}

func (m *mockTenantRepo) GetByID(_ context.Context, id string) (*tenantModel.Tenant, error) {
	if t, ok := m.tenants[id]; ok && t.DeletedAt == nil {
		return t, nil
	}
	return nil, errors.New("tenant not found")
}

func (m *mockTenantRepo) GetByDomain(_ context.Context, domain string) (*tenantModel.Tenant, error) {
	for _, t := range m.tenants {
		if t.Domain == domain && t.DeletedAt == nil {
			return t, nil
		}
	}
	return nil, errors.New("tenant not found")
}

func (m *mockTenantRepo) GetByName(_ context.Context, name string) (*tenantModel.Tenant, error) {
	for _, t := range m.tenants {
		if t.Name == name && t.DeletedAt == nil {
			return t, nil
		}
	}
	return nil, errors.New("tenant not found")
}

func (m *mockTenantRepo) CreateTenantWithAdmin(_ context.Context, tenant *tenantModel.Tenant, _ *userModel.User) error {
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *mockTenantRepo) Create(_ context.Context, tenant *tenantModel.Tenant) error {
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *mockTenantRepo) UpdateStatus(_ context.Context, id string, status int) error {
	if t, ok := m.tenants[id]; ok {
		t.Status = status
		return nil
	}
	return errors.New("tenant not found")
}

func (m *mockTenantRepo) Update(_ context.Context, id string, tenant *tenantModel.Tenant) error {
	existing, ok := m.tenants[id]
	if !ok {
		return errors.New("tenant not found")
	}
	// 保留 id / 状态 / 时间戳，覆盖业务字段
	tenant.ID = existing.ID
	tenant.Status = existing.Status
	tenant.CreatedAt = existing.CreatedAt
	tenant.UpdatedAt = time.Now()
	m.tenants[id] = tenant
	return nil
}

func (m *mockTenantRepo) Delete(_ context.Context, id string) error {
	if t, ok := m.tenants[id]; ok {
		now := time.Now()
		t.DeletedAt = &now
		return nil
	}
	return errors.New("tenant not found")
}

func (m *mockTenantRepo) HardDelete(_ context.Context, id string) error {
	if _, ok := m.tenants[id]; !ok {
		return errors.New("tenant not found")
	}
	delete(m.tenants, id)
	return nil
}

func (m *mockTenantRepo) FindPage(_ context.Context, _ int, _ int, _ string, _ *int) ([]*tenantModel.Tenant, int64, error) {
	var out []*tenantModel.Tenant
	for _, t := range m.tenants {
		if t.DeletedAt == nil {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, int64(len(out)), nil
}

func (m *mockTenantRepo) BatchUpdateStatus(_ context.Context, ids []string, status int) (int64, []string, error) {
	updated := int64(0)
	var failed []string
	for _, id := range ids {
		if err := m.UpdateStatus(context.Background(), id, status); err != nil {
			failed = append(failed, id)
		} else {
			updated++
		}
	}
	return updated, failed, nil
}

func (m *mockTenantRepo) BatchDelete(_ context.Context, ids []string) (int64, []string, error) {
	deleted := int64(0)
	var failed []string
	for _, id := range ids {
		if err := m.Delete(context.Background(), id); err != nil {
			failed = append(failed, id)
		} else {
			deleted++
		}
	}
	return deleted, failed, nil
}

func (m *mockTenantRepo) FindDeleted(_ context.Context, _ int, _ int, _ string) ([]*tenantModel.Tenant, int64, error) {
	var out []*tenantModel.Tenant
	for _, t := range m.tenants {
		if t.DeletedAt != nil {
			out = append(out, t)
		}
	}
	return out, int64(len(out)), nil
}

func (m *mockTenantRepo) Restore(_ context.Context, id string) error {
	if t, ok := m.tenants[id]; ok && t.DeletedAt != nil {
		t.DeletedAt = nil
		return nil
	}
	return errors.New("tenant not deleted")
}

// ---------- CancelRequest 方法 ----------

func (m *mockTenantRepo) seedCancelRequest(c *tenantModel.CancelRequest) {
	m.cancelRequests[c.ID] = c
}

func (m *mockTenantRepo) CreateCancelRequest(_ context.Context, req *tenantModel.CancelRequest) error {
	m.cancelRequests[req.ID] = req
	return nil
}

func (m *mockTenantRepo) GetCancelRequestByID(_ context.Context, id string) (*tenantModel.CancelRequest, error) {
	if c, ok := m.cancelRequests[id]; ok {
		return c, nil
	}
	return nil, errors.New("cancel request not found")
}

func (m *mockTenantRepo) GetPendingCancelRequestByTenant(_ context.Context, tenantID string) (*tenantModel.CancelRequest, error) {
	for _, c := range m.cancelRequests {
		if c.TenantID == tenantID && c.Status == tenantModel.CancelRequestStatusPending {
			return c, nil
		}
	}
	return nil, nil
}

func (m *mockTenantRepo) UpdateCancelRequest(_ context.Context, req *tenantModel.CancelRequest) error {
	if _, ok := m.cancelRequests[req.ID]; !ok {
		return errors.New("cancel request not found")
	}
	m.cancelRequests[req.ID] = req
	return nil
}

func (m *mockTenantRepo) FindCancelRequests(_ context.Context, _ int, _ int, status int, _ string) ([]*tenantModel.CancelRequest, int64, error) {
	var out []*tenantModel.CancelRequest
	for _, c := range m.cancelRequests {
		if status == 0 || c.Status == status {
			out = append(out, c)
		}
	}
	return out, int64(len(out)), nil
}

func (m *mockTenantRepo) FindApprovedDueCancelRequests(_ context.Context, now time.Time) ([]*tenantModel.CancelRequest, error) {
	var out []*tenantModel.CancelRequest
	for _, c := range m.cancelRequests {
		if c.Status == tenantModel.CancelRequestStatusApproved && c.EffectiveAt != nil && !c.EffectiveAt.After(now) {
			out = append(out, c)
		}
	}
	return out, nil
}

var _ tenantRepo.TenantRepository = (*mockTenantRepo)(nil)

// ---------- 内存 mock：UserRepository（仅 UsernameExists 等） ----------

type mockUserRepo struct {
	userRepo.UserRepository
	names map[string]bool
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{names: make(map[string]bool)}
}

func (m *mockUserRepo) UsernameExists(_ context.Context, username string) (bool, error) {
	return m.names[username], nil
}

func (m *mockUserRepo) Create(_ context.Context, u *userModel.User) error {
	m.names[u.Username] = true
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
	assigned map[string][]string
}

func newMockUserRoleRepo() *mockUserRoleRepo {
	return &mockUserRoleRepo{assigned: make(map[string][]string)}
}

func (m *mockUserRoleRepo) AssignRoles(_ context.Context, userID string, roleIDs []string) error {
	m.assigned[userID] = append([]string{}, roleIDs...)
	return nil
}

var _ rbacRepo.UserRoleRepository = (*mockUserRoleRepo)(nil)

// ---------- 内存 mock：SubscriptionRepository ----------

type mockSubRepo struct {
	planRepo.SubscriptionRepository
	active  map[string]*planModel.TenantSubscription
	updated map[string]int
}

func newMockSubRepo() *mockSubRepo {
	return &mockSubRepo{
		active:  make(map[string]*planModel.TenantSubscription),
		updated: make(map[string]int),
	}
}

func (m *mockSubRepo) GetActiveByTenant(_ context.Context, tenantID string) (*planModel.TenantSubscription, error) {
	if sub, ok := m.active[tenantID]; ok {
		return sub, nil
	}
	return nil, errors.New("no active subscription")
}

func (m *mockSubRepo) UpdateStatus(_ context.Context, id string, status int) error {
	m.updated[id] = status
	return nil
}

var _ planRepo.SubscriptionRepository = (*mockSubRepo)(nil)

// ---------- 内存 mock：TenantSettingsRepository ----------

type mockSettingsRepo struct {
	settings map[string]*tenantModel.TenantSettings
}

func newMockSettingsRepo() *mockSettingsRepo {
	return &mockSettingsRepo{settings: make(map[string]*tenantModel.TenantSettings)}
}

func (m *mockSettingsRepo) GetByTenantID(_ context.Context, tenantID string) (*tenantModel.TenantSettings, error) {
	if s, ok := m.settings[tenantID]; ok {
		return s, nil
	}
	return nil, nil
}

func (m *mockSettingsRepo) Create(_ context.Context, s *tenantModel.TenantSettings) error {
	m.settings[s.TenantID] = s
	return nil
}

func (m *mockSettingsRepo) Update(_ context.Context, s *tenantModel.TenantSettings) error {
	existing, ok := m.settings[s.TenantID]
	if !ok {
		return errors.New("settings not found")
	}
	// 模拟仅更新非空字段
	if s.Logo != "" {
		existing.Logo = s.Logo
	}
	if s.Favicon != "" {
		existing.Favicon = s.Favicon
	}
	if s.PrimaryColor != "" {
		existing.PrimaryColor = s.PrimaryColor
	}
	if s.Theme != "" {
		existing.Theme = s.Theme
	}
	if s.Language != "" {
		existing.Language = s.Language
	}
	if s.Timezone != "" {
		existing.Timezone = s.Timezone
	}
	if s.Description != "" {
		existing.Description = s.Description
	}
	if s.WelcomeText != "" {
		existing.WelcomeText = s.WelcomeText
	}
	if s.ContactName != "" {
		existing.ContactName = s.ContactName
	}
	if s.ContactEmail != "" {
		existing.ContactEmail = s.ContactEmail
	}
	if s.ContactPhone != "" {
		existing.ContactPhone = s.ContactPhone
	}
	if s.Address != "" {
		existing.Address = s.Address
	}
	if s.Extra != "" {
		existing.Extra = s.Extra
	}
	existing.UpdatedAt = time.Now()
	return nil
}

func (m *mockSettingsRepo) Upsert(_ context.Context, s *tenantModel.TenantSettings) error {
	existing, err := m.GetByTenantID(context.Background(), s.TenantID)
	if err != nil {
		return err
	}
	if existing == nil {
		return m.Create(context.Background(), s)
	}
	return m.Update(context.Background(), s)
}

var _ tenantRepo.TenantSettingsRepository = (*mockSettingsRepo)(nil)
