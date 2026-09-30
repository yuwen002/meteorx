package service

import (
	"context"
	"encoding/json"
	"errors"
	"meteorx/internal/config"
	"meteorx/internal/modules/invitation/dto"
	"meteorx/internal/modules/invitation/model"
	"testing"
	"time"

	rbacModel "meteorx/internal/modules/rbac/model"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	tenantModel "meteorx/internal/modules/tenant/model"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	userModel "meteorx/internal/modules/user/model"
	userRepo "meteorx/internal/modules/user/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubInvRepo struct {
	invitations map[string]*model.Invitation
	byToken     map[string]string
}

func newStubInvRepo() *stubInvRepo {
	return &stubInvRepo{
		invitations: make(map[string]*model.Invitation),
		byToken:     make(map[string]string),
	}
}

func (r *stubInvRepo) Create(_ context.Context, inv *model.Invitation) error {
	r.invitations[inv.ID] = inv
	r.byToken[inv.Token] = inv.ID
	return nil
}

func (r *stubInvRepo) GetByToken(_ context.Context, token string) (*model.Invitation, error) {
	id, ok := r.byToken[token]
	if !ok {
		return nil, errors.New("not found")
	}
	return r.invitations[id], nil
}

func (r *stubInvRepo) GetByID(_ context.Context, id string) (*model.Invitation, error) {
	inv, ok := r.invitations[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return inv, nil
}

func (r *stubInvRepo) ListByTenant(_ context.Context, tenantID string, page, pageSize int, keyword, status string) ([]*model.Invitation, int64, error) {
	var result []*model.Invitation
	for _, inv := range r.invitations {
		if inv.TenantID == tenantID {
			result = append(result, inv)
		}
	}
	return result, int64(len(result)), nil
}

func (r *stubInvRepo) UpdateStatus(_ context.Context, id, status string) error {
	inv, ok := r.invitations[id]
	if !ok {
		return errors.New("not found")
	}
	inv.Status = status
	return nil
}

func (r *stubInvRepo) Delete(_ context.Context, id string) error {
	delete(r.invitations, id)
	return nil
}

func (r *stubInvRepo) CountPendingByTenant(_ context.Context, tenantID string) (int64, error) {
	var count int64
	for _, inv := range r.invitations {
		if inv.TenantID == tenantID && inv.Status == model.InvitationStatusPending {
			count++
		}
	}
	return count, nil
}

func (r *stubInvRepo) FindByEmailAndTenant(_ context.Context, email, tenantID string) (*model.Invitation, error) {
	for _, inv := range r.invitations {
		if inv.Email == email && inv.TenantID == tenantID {
			return inv, nil
		}
	}
	return nil, errors.New("not found")
}

type stubUserRepo struct {
	users   map[string]*userModel.User
	byEmail map[string]*userModel.User
}

func newStubUserRepo() *stubUserRepo {
	return &stubUserRepo{
		users:   make(map[string]*userModel.User),
		byEmail: make(map[string]*userModel.User),
	}
}

func (r *stubUserRepo) Create(_ context.Context, user *userModel.User) error {
	r.users[user.ID] = user
	r.byEmail[user.Email] = user
	return nil
}
func (r *stubUserRepo) GetByUsername(_ context.Context, tenantID, username string) (*userModel.User, error) {
	return nil, errors.New("not found")
}
func (r *stubUserRepo) GetByID(_ context.Context, id string) (*userModel.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}
func (r *stubUserRepo) GetByEmail(_ context.Context, email string) (*userModel.User, error) {
	u, ok := r.byEmail[email]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}
func (r *stubUserRepo) UsernameExists(_ context.Context, username string) (bool, error) {
	return false, nil
}
func (r *stubUserRepo) ListByTenant(_ context.Context, tenantID string, page, pageSize int, keyword string, status *int) ([]*userModel.User, int64, error) {
	return nil, 0, nil
}
func (r *stubUserRepo) ListMasterAdmins(_ context.Context, page, pageSize int, keyword string) ([]*userModel.User, int64, error) {
	return nil, 0, nil
}
func (r *stubUserRepo) ListAllTenantUsers(_ context.Context, page, pageSize int, keyword string) ([]*userModel.User, int64, error) {
	return nil, 0, nil
}
func (r *stubUserRepo) Update(_ context.Context, user *userModel.User) error { return nil }
func (r *stubUserRepo) Delete(_ context.Context, id string) error             { return nil }
func (r *stubUserRepo) UpdateStatus(_ context.Context, id string, status int) error {
	return nil
}
func (r *stubUserRepo) FindDeletedMasterAdmins(_ context.Context, page, pageSize int, keyword string) ([]*userModel.User, int64, error) {
	return nil, 0, nil
}
func (r *stubUserRepo) RestoreMasterAdmin(_ context.Context, id string) error { return nil }
func (r *stubUserRepo) PermanentDeleteMasterAdmin(_ context.Context, id string) error {
	return nil
}
func (r *stubUserRepo) BatchUpdateStatus(_ context.Context, ids []string, status int) (int64, error) {
	return 0, nil
}
func (r *stubUserRepo) BatchDelete(_ context.Context, ids []string) (int64, error) {
	return 0, nil
}
func (r *stubUserRepo) FindDeletedTenantUsers(_ context.Context, tenantID string, page, pageSize int, keyword string) ([]*userModel.User, int64, error) {
	return nil, 0, nil
}
func (r *stubUserRepo) FindAllDeletedTenantUsers(_ context.Context, page, pageSize int, keyword string) ([]*userModel.User, int64, error) {
	return nil, 0, nil
}
func (r *stubUserRepo) RestoreTenantUser(_ context.Context, tenantID, userID string) error {
	return nil
}
func (r *stubUserRepo) PermanentDeleteTenantUser(_ context.Context, tenantID, userID string) error {
	return nil
}
func (r *stubUserRepo) BatchUpdateTenantUserStatus(_ context.Context, tenantID string, ids []string, status int) (int64, error) {
	return 0, nil
}
func (r *stubUserRepo) BatchDeleteTenantUsers(_ context.Context, tenantID string, ids []string) (int64, error) {
	return 0, nil
}
func (r *stubUserRepo) CountByTenant(_ context.Context, tenantID string) (int64, error) {
	return 0, nil
}
func (r *stubUserRepo) CountAllUsers(_ context.Context) (int64, error) { return 0, nil }
func (r *stubUserRepo) UpdateEmailVerified(_ context.Context, userID string, verified bool) error {
	return nil
}
func (r *stubUserRepo) GetByPhone(_ context.Context, phone string) (*userModel.User, error) {
	return nil, errors.New("not found")
}

type stubTenantRepo struct {
	tenants map[string]*tenantModel.Tenant
}

func newStubTenantRepo() *stubTenantRepo {
	return &stubTenantRepo{tenants: make(map[string]*tenantModel.Tenant)}
}

func (r *stubTenantRepo) GetByID(_ context.Context, id string) (*tenantModel.Tenant, error) {
	t, ok := r.tenants[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return t, nil
}
func (r *stubTenantRepo) Create(_ context.Context, tenant *tenantModel.Tenant) error {
	r.tenants[tenant.ID] = tenant
	return nil
}
func (r *stubTenantRepo) GetByDomain(_ context.Context, domain string) (*tenantModel.Tenant, error) {
	return nil, errors.New("not found")
}
func (r *stubTenantRepo) GetByName(_ context.Context, name string) (*tenantModel.Tenant, error) {
	return nil, errors.New("not found")
}
func (r *stubTenantRepo) CreateTenantWithAdmin(_ context.Context, tenant *tenantModel.Tenant, user *userModel.User, roleIDs []string) error {
	return nil
}
func (r *stubTenantRepo) UpdateStatus(_ context.Context, id string, status int) error {
	return nil
}
func (r *stubTenantRepo) Update(_ context.Context, id string, tenant *tenantModel.Tenant) error {
	return nil
}
func (r *stubTenantRepo) Delete(_ context.Context, id string) error  { return nil }
func (r *stubTenantRepo) HardDelete(_ context.Context, id string) error { return nil }
func (r *stubTenantRepo) FindPage(_ context.Context, page, pageSize int, name string, status *int) ([]*tenantModel.Tenant, int64, error) {
	return nil, 0, nil
}
func (r *stubTenantRepo) BatchUpdateStatus(_ context.Context, ids []string, status int) (int64, []string, error) {
	return 0, nil, nil
}
func (r *stubTenantRepo) BatchDelete(_ context.Context, ids []string) (int64, []string, error) {
	return 0, nil, nil
}
func (r *stubTenantRepo) FindDeleted(_ context.Context, page, pageSize int, name string) ([]*tenantModel.Tenant, int64, error) {
	return nil, 0, nil
}
func (r *stubTenantRepo) Restore(_ context.Context, id string) error { return nil }
func (r *stubTenantRepo) CreateCancelRequest(_ context.Context, req *tenantModel.CancelRequest) error {
	return nil
}
func (r *stubTenantRepo) GetCancelRequestByID(_ context.Context, id string) (*tenantModel.CancelRequest, error) {
	return nil, errors.New("not found")
}
func (r *stubTenantRepo) GetPendingCancelRequestByTenant(_ context.Context, tenantID string) (*tenantModel.CancelRequest, error) {
	return nil, errors.New("not found")
}
func (r *stubTenantRepo) UpdateCancelRequest(_ context.Context, req *tenantModel.CancelRequest) error {
	return nil
}
func (r *stubTenantRepo) FindCancelRequests(_ context.Context, page, pageSize int, status int, keyword string) ([]*tenantModel.CancelRequest, int64, error) {
	return nil, 0, nil
}
func (r *stubTenantRepo) FindApprovedDueCancelRequests(_ context.Context, now time.Time) ([]*tenantModel.CancelRequest, error) {
	return nil, nil
}
func (r *stubTenantRepo) ListActive(_ context.Context) ([]*tenantModel.Tenant, error) {
	return nil, nil
}

type stubRoleRepo struct{}

func (r *stubRoleRepo) Create(_ context.Context, role *rbacModel.Role) error { return nil }
func (r *stubRoleRepo) GetByID(_ context.Context, id string) (*rbacModel.Role, error) {
	return nil, errors.New("not found")
}
func (r *stubRoleRepo) GetByCode(_ context.Context, tenantID, code string) (*rbacModel.Role, error) {
	return nil, errors.New("not found")
}
func (r *stubRoleRepo) List(_ context.Context, tenantID string, page, pageSize int, keyword string) ([]*rbacModel.Role, int64, error) {
	return nil, 0, nil
}
func (r *stubRoleRepo) ListByScope(_ context.Context, scope string) ([]*rbacModel.Role, error) {
	return nil, nil
}
func (r *stubRoleRepo) ListSystemAdminRoles(_ context.Context) ([]*rbacModel.Role, error) {
	return nil, nil
}
func (r *stubRoleRepo) Update(_ context.Context, role *rbacModel.Role) error { return nil }
func (r *stubRoleRepo) UpdateStatus(_ context.Context, id string, status int) error {
	return nil
}
func (r *stubRoleRepo) BatchUpdateStatus(_ context.Context, ids []string, status int) (int64, error) {
	return 0, nil
}
func (r *stubRoleRepo) Delete(_ context.Context, id string) error { return nil }
func (r *stubRoleRepo) BatchDelete(_ context.Context, ids []string) (int64, error) { return 0, nil }
func (r *stubRoleRepo) FindDeleted(_ context.Context, page, pageSize int, keyword string) ([]*rbacModel.Role, int64, error) {
	return nil, 0, nil
}
func (r *stubRoleRepo) Restore(_ context.Context, id string) error { return nil }
func (r *stubRoleRepo) PermanentDelete(_ context.Context, id string) error { return nil }
func (r *stubRoleRepo) BatchPermanentDelete(_ context.Context, ids []string) (int64, error) {
	return 0, nil
}
func (r *stubRoleRepo) Count(_ context.Context, tenantID string) (int64, error) { return 0, nil }

type stubUserRoleRepo struct{}

func newStubUserRoleRepo() *stubUserRoleRepo { return &stubUserRoleRepo{} }

func (r *stubUserRoleRepo) AssignRoles(_ context.Context, userID string, roleIDs []string) error {
	return nil
}
func (r *stubUserRoleRepo) GetRoleIDsByUserID(_ context.Context, userID string) ([]string, error) {
	return nil, nil
}
func (r *stubUserRoleRepo) GetRoleCodesByUserID(_ context.Context, userID string) ([]string, error) {
	return nil, nil
}
func (r *stubUserRoleRepo) BatchGetRoleIDsByUserIDs(_ context.Context, userIDs []string) (map[string][]string, error) {
	return nil, nil
}
func (r *stubUserRoleRepo) DeleteByUserID(_ context.Context, userID string) error { return nil }
func (r *stubUserRoleRepo) DeleteByUserIDAndRoleID(_ context.Context, userID, roleID string) error {
	return nil
}
func (r *stubUserRoleRepo) GetUserIDsByRoleID(_ context.Context, roleID string) ([]string, error) {
	return nil, nil
}
func (r *stubUserRoleRepo) CountByRoleID(_ context.Context, roleID string) (int64, error) { return 0, nil }
func (r *stubUserRoleRepo) CountByUserID(_ context.Context, userID string) (int64, error) { return 0, nil }
func (r *stubUserRoleRepo) CheckUserExists(_ context.Context, userID string) error { return nil }
func (r *stubUserRoleRepo) ListUserRoles(_ context.Context, page, pageSize int, userID, roleID string) ([]*rbacModel.UserRole, int64, error) {
	return nil, 0, nil
}

func newTestService(invRepo *stubInvRepo, userRepo *stubUserRepo) *InvitationService {
	return NewInvitationService(
		invRepo,
		userRepo,
		newStubTenantRepo(),
		&stubRoleRepo{},
		newStubUserRoleRepo(),
		config.EmailConfig{Enabled: false},
		config.ClientConfig{BaseURL: "http://localhost:3000"},
		config.SecurityConfig{PasswordPolicy: config.PasswordPolicyConfig{MinLength: 6}},
	)
}

func TestCreate_Success(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	req := dto.CreateInvitationReq{
		Email:   "test@example.com",
		RoleIDs: []string{"role-1"},
	}

	inv, err := svc.Create(context.Background(), "tenant-1", "user-1", req)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", inv.Email)
	assert.Equal(t, model.InvitationStatusPending, inv.Status)
	assert.Equal(t, "tenant-1", inv.TenantID)
	assert.Equal(t, "user-1", inv.InvitedBy)

	var roleIDs []string
	require.NoError(t, json.Unmarshal([]byte(inv.RoleIDs), &roleIDs))
	assert.Equal(t, []string{"role-1"}, roleIDs)
}

func TestCreate_DuplicateEmail(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	req := dto.CreateInvitationReq{
		Email:   "dup@example.com",
		RoleIDs: []string{"role-1"},
	}

	_, err := svc.Create(context.Background(), "tenant-1", "user-1", req)
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), "tenant-1", "user-1", req)
	assert.ErrorIs(t, err, ErrEmailAlreadyInvited)
}

func TestCreate_EmailAlreadyMember(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	userRepo.byEmail["member@example.com"] = &userModel.User{
		ID:       "u-1",
		TenantID: "tenant-1",
		Email:    "member@example.com",
	}
	svc := newTestService(invRepo, userRepo)

	req := dto.CreateInvitationReq{
		Email:   "member@example.com",
		RoleIDs: []string{"role-1"},
	}

	_, err := svc.Create(context.Background(), "tenant-1", "user-1", req)
	assert.ErrorIs(t, err, ErrEmailAlreadyMember)
}

func TestAccept_Success(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	roleIDs, _ := json.Marshal([]string{"role-1"})
	now := time.Now()
	inv := &model.Invitation{
		ID:        "inv-1",
		TenantID:  "tenant-1",
		Email:     "new@example.com",
		Token:     "token-123",
		RoleIDs:   string(roleIDs),
		Status:    model.InvitationStatusPending,
		InvitedBy: "user-1",
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	invRepo.invitations[inv.ID] = inv
	invRepo.byToken[inv.Token] = inv.ID

	req := dto.AcceptInvitationReq{
		Token:    "token-123",
		Username: "newuser",
		Password: "password123",
		Nickname: "New User",
	}

	err := svc.Accept(context.Background(), req)
	require.NoError(t, err)

	updated := invRepo.invitations[inv.ID]
	assert.Equal(t, model.InvitationStatusAccepted, updated.Status)
}

func TestAccept_NotFound(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	req := dto.AcceptInvitationReq{
		Token:    "invalid-token",
		Username: "newuser",
		Password: "password123",
		Nickname: "New User",
	}

	err := svc.Accept(context.Background(), req)
	assert.ErrorIs(t, err, ErrInvitationNotFound)
}

func TestAccept_Expired(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	roleIDs, _ := json.Marshal([]string{"role-1"})
	now := time.Now()
	inv := &model.Invitation{
		ID:        "inv-1",
		TenantID:  "tenant-1",
		Email:     "expired@example.com",
		Token:     "token-exp",
		RoleIDs:   string(roleIDs),
		Status:    model.InvitationStatusPending,
		InvitedBy: "user-1",
		ExpiresAt: now.Add(-1 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	invRepo.invitations[inv.ID] = inv
	invRepo.byToken[inv.Token] = inv.ID

	req := dto.AcceptInvitationReq{
		Token:    "token-exp",
		Username: "newuser",
		Password: "password123",
		Nickname: "New User",
	}

	err := svc.Accept(context.Background(), req)
	assert.ErrorIs(t, err, ErrInvitationExpired)
}

func TestAccept_AlreadyUsed(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	roleIDs, _ := json.Marshal([]string{"role-1"})
	now := time.Now()
	inv := &model.Invitation{
		ID:        "inv-1",
		TenantID:  "tenant-1",
		Email:     "used@example.com",
		Token:     "token-used",
		RoleIDs:   string(roleIDs),
		Status:    model.InvitationStatusAccepted,
		InvitedBy: "user-1",
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	invRepo.invitations[inv.ID] = inv
	invRepo.byToken[inv.Token] = inv.ID

	req := dto.AcceptInvitationReq{
		Token:    "token-used",
		Username: "newuser",
		Password: "password123",
		Nickname: "New User",
	}

	err := svc.Accept(context.Background(), req)
	assert.ErrorIs(t, err, ErrInvitationAlreadyUsed)
}

func TestCancel_Success(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	now := time.Now()
	inv := &model.Invitation{
		ID:        "inv-1",
		TenantID:  "tenant-1",
		Email:     "cancel@example.com",
		Token:     "token-cancel",
		Status:    model.InvitationStatusPending,
		InvitedBy: "user-1",
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	invRepo.invitations[inv.ID] = inv

	err := svc.Cancel(context.Background(), "tenant-1", "inv-1")
	require.NoError(t, err)
	assert.Equal(t, model.InvitationStatusCancelled, invRepo.invitations[inv.ID].Status)
}

func TestCancel_NotFound(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	err := svc.Cancel(context.Background(), "tenant-1", "nonexistent")
	assert.ErrorIs(t, err, ErrInvitationNotFound)
}

func TestCancel_AlreadyUsed(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	now := time.Now()
	inv := &model.Invitation{
		ID:        "inv-1",
		TenantID:  "tenant-1",
		Email:     "used@example.com",
		Token:     "token-used",
		Status:    model.InvitationStatusAccepted,
		InvitedBy: "user-1",
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	invRepo.invitations[inv.ID] = inv

	err := svc.Cancel(context.Background(), "tenant-1", "inv-1")
	assert.ErrorIs(t, err, ErrInvitationAlreadyUsed)
}

func TestDelete_Success(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	now := time.Now()
	inv := &model.Invitation{
		ID:        "inv-1",
		TenantID:  "tenant-1",
		Email:     "del@example.com",
		Token:     "token-del",
		Status:    model.InvitationStatusPending,
		InvitedBy: "user-1",
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	invRepo.invitations[inv.ID] = inv

	err := svc.Delete(context.Background(), "tenant-1", "inv-1")
	require.NoError(t, err)
	_, exists := invRepo.invitations[inv.ID]
	assert.False(t, exists)
}

func TestDelete_NotFound(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	err := svc.Delete(context.Background(), "tenant-1", "nonexistent")
	assert.ErrorIs(t, err, ErrInvitationNotFound)
}

func TestGetByToken_Success(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	now := time.Now()
	inv := &model.Invitation{
		ID:        "inv-1",
		TenantID:  "tenant-1",
		Email:     "info@example.com",
		Token:     "token-info",
		Status:    model.InvitationStatusPending,
		InvitedBy: "user-1",
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}
	invRepo.invitations[inv.ID] = inv
	invRepo.byToken[inv.Token] = inv.ID

	result, err := svc.GetByToken(context.Background(), "token-info")
	require.NoError(t, err)
	assert.Equal(t, "info@example.com", result.Email)
}

func TestGetByToken_NotFound(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	_, err := svc.GetByToken(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrInvitationNotFound)
}

func TestList(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	now := time.Now()
	invRepo.invitations["inv-1"] = &model.Invitation{
		ID: "inv-1", TenantID: "tenant-1", Email: "a@example.com",
		Token: "t1", Status: model.InvitationStatusPending,
		ExpiresAt: now.Add(7 * 24 * time.Hour), CreatedAt: now, UpdatedAt: now,
	}
	invRepo.invitations["inv-2"] = &model.Invitation{
		ID: "inv-2", TenantID: "tenant-1", Email: "b@example.com",
		Token: "t2", Status: model.InvitationStatusAccepted,
		ExpiresAt: now.Add(7 * 24 * time.Hour), CreatedAt: now, UpdatedAt: now,
	}

	items, total, err := svc.List(context.Background(), "tenant-1", 1, 10, "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 2)
}

func TestResend_EmailNotConfigured(t *testing.T) {
	invRepo := newStubInvRepo()
	userRepo := newStubUserRepo()
	svc := newTestService(invRepo, userRepo)

	now := time.Now()
	inv := &model.Invitation{
		ID: "inv-1", TenantID: "tenant-1", Email: "resend@example.com",
		Token: "token-resend", Status: model.InvitationStatusPending,
		InvitedBy: "user-1", ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now, UpdatedAt: now,
	}
	invRepo.invitations[inv.ID] = inv

	_, err := svc.Resend(context.Background(), "tenant-1", "inv-1")
	assert.ErrorIs(t, err, ErrEmailNotConfigured)
}

var _ userRepo.UserRepository = (*stubUserRepo)(nil)
var _ tenantRepo.TenantRepository = (*stubTenantRepo)(nil)
var _ rbacRepo.RoleRepository = (*stubRoleRepo)(nil)
var _ rbacRepo.UserRoleRepository = (*stubUserRoleRepo)(nil)