package service

import (
	"context"
	"strings"
	"testing"

	rbacModel "meteorx/internal/modules/rbac/model"
	tenantModel "meteorx/internal/modules/tenant/model"
	"meteorx/internal/modules/user/dto"
	userModel "meteorx/internal/modules/user/model"
	"meteorx/pkg/crypto"
)

func newTestUserService() (*UserService, *mockUserRepo, *mockTenantRepo, *mockRoleRepo, *mockUserRoleRepo) {
	ur := newMockUserRepo()
	tr := newMockTenantRepo()
	rr := newMockRoleRepo()
	urr := newMockUserRoleRepo()
	svc := NewUserService(ur, tr, rr, urr)
	return svc, ur, tr, rr, urr
}

func seedTenantRole(rr *mockRoleRepo, id, code, scope string, status int) *rbacModel.Role {
	r := &rbacModel.Role{ID: id, Name: code, Code: code, Scope: scope, Status: status}
	rr.seed(r)
	return r
}

// ---------- Create ----------

func TestCreate_UsernameTaken(t *testing.T) {
	svc, ur, _, _, _ := newTestUserService()
	ur.seed(&userModel.User{ID: "u1", TenantID: "t1", Username: "alice"})

	_, err := svc.Create(context.Background(), "t1", dto.CreateUserReq{Username: "alice", Password: "123456"})
	if err == nil || !strings.Contains(err.Error(), "用户名已被使用") {
		t.Fatalf("expected username taken error, got: %v", err)
	}
}

func TestCreate_QuotaExceeded(t *testing.T) {
	svc, ur, _, rr, _ := newTestUserService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1)
	// 配额校验：超限
	svc.SetQuotaVerifier(quotaVerifierFunc(func(ctx context.Context, tenantID string) (bool, int64, int, error) {
		return true, 10, 5, nil
	}))

	_, err := svc.Create(context.Background(), "t1", dto.CreateUserReq{
		Username: "bob", Password: "123456", Nickname: "Bob", RoleIDs: []string{"r1"},
	})
	if err == nil || !strings.Contains(err.Error(), "已达到套餐用户数上限") {
		t.Fatalf("expected quota error, got: %v", err)
	}
	_ = ur
}

func TestCreate_QuotaErrPlanExpired(t *testing.T) {
	svc, _, _, rr, _ := newTestUserService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1)
	svc.SetQuotaVerifier(quotaVerifierFunc(func(ctx context.Context, tenantID string) (bool, int64, int, error) {
		return false, 0, 0, errPlanExpiredMock
	}))

	_, err := svc.Create(context.Background(), "t1", dto.CreateUserReq{
		Username: "bob", Password: "123456", RoleIDs: []string{"r1"},
	})
	if err == nil || !strings.Contains(err.Error(), "套餐已到期") {
		t.Fatalf("expected plan expired error, got: %v", err)
	}
}

func TestCreate_RoleNotFound(t *testing.T) {
	svc, _, _, _, _ := newTestUserService()
	_, err := svc.Create(context.Background(), "t1", dto.CreateUserReq{
		Username: "bob", Password: "123456", RoleIDs: []string{"not-exist"},
	})
	if err == nil || !strings.Contains(err.Error(), "角色") {
		t.Fatalf("expected role error, got: %v", err)
	}
}

func TestCreate_RoleDisabled(t *testing.T) {
	svc, _, _, rr, _ := newTestUserService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 0) // 禁用

	_, err := svc.Create(context.Background(), "t1", dto.CreateUserReq{
		Username: "bob", Password: "123456", RoleIDs: []string{"r1"},
	})
	if err == nil || !strings.Contains(err.Error(), "已禁用") {
		t.Fatalf("expected disabled role error, got: %v", err)
	}
}

func TestCreate_RoleScopeMismatch(t *testing.T) {
	svc, _, _, rr, _ := newTestUserService()
	seedTenantRole(rr, "r1", "sysrole", rbacModel.RoleScopeSystem, 1) // 系统级角色，不能在租户上下文分配

	_, err := svc.Create(context.Background(), "t1", dto.CreateUserReq{
		Username: "bob", Password: "123456", RoleIDs: []string{"r1"},
	})
	if err == nil || !strings.Contains(err.Error(), "不允许在当前上下文分配") {
		t.Fatalf("expected scope error, got: %v", err)
	}
}

func TestCreate_Success(t *testing.T) {
	svc, _, tr, rr, urr := newTestUserService()
	tr.tenants["t1"] = &tenantModel.Tenant{ID: "t1", Name: "Tenant A"}
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1)

	resp, err := svc.Create(context.Background(), "t1", dto.CreateUserReq{
		Username: "bob", Password: "123456", Nickname: "Bob", Email: "bob@example.com", RoleIDs: []string{"r1"},
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if resp.Username != "bob" || resp.TenantName != "Tenant A" || resp.Status != 1 {
		t.Fatalf("unexpected resp: %+v", resp)
	}
	// 密码应被哈希
	ids := urr.userRoles[resp.ID]
	if len(ids) != 1 || ids[0] != "r1" {
		t.Fatalf("expected role r1 assigned, got %v", ids)
	}
}

// ---------- CreateMasterAdmin ----------

func TestCreateMasterAdmin_DefaultSuperadmin(t *testing.T) {
	svc, _, _, rr, urr := newTestUserService()
	seedTenantRole(rr, "sa-role", "superadmin", rbacModel.RoleScopeSystem, 1)

	resp, err := svc.CreateMasterAdmin(context.Background(), dto.CreateMasterAdminReq{
		Username: "root2", Password: "123456", Nickname: "Root",
	})
	if err != nil {
		t.Fatalf("CreateMasterAdmin error: %v", err)
	}
	if !resp.IsMaster || resp.TenantID != "SYSTEM_ROOT" {
		t.Fatalf("unexpected master admin: %+v", resp)
	}
	ids := urr.userRoles[resp.ID]
	if len(ids) != 1 || ids[0] != "sa-role" {
		t.Fatalf("expected superadmin role, got %v", ids)
	}
}

func TestCreateMasterAdmin_SuperadminNotFound(t *testing.T) {
	svc, _, _, _, _ := newTestUserService()
	// 未 seed superadmin 角色
	_, err := svc.CreateMasterAdmin(context.Background(), dto.CreateMasterAdminReq{
		Username: "root2", Password: "123456",
	})
	if err == nil || !strings.Contains(err.Error(), "superadmin") {
		t.Fatalf("expected superadmin not configured error, got: %v", err)
	}
}

func TestCreateMasterAdmin_SpecifiedRoleWrongScope(t *testing.T) {
	svc, _, _, rr, _ := newTestUserService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1) // 租户级角色

	_, err := svc.CreateMasterAdmin(context.Background(), dto.CreateMasterAdminReq{
		Username: "root2", Password: "123456", RoleID: "r1",
	})
	if err == nil || !strings.Contains(err.Error(), "不是系统级别角色") {
		t.Fatalf("expected system role error, got: %v", err)
	}
}

// ---------- MasterAdmin 删除保护 ----------

func TestDeleteMasterAdmin_ProtectedUserBlocked(t *testing.T) {
	svc, ur, _, _, _ := newTestUserService()
	ur.seed(&userModel.User{ID: "admin-id-000001", Username: "admin", IsMaster: true})

	err := svc.DeleteMasterAdmin(context.Background(), "admin-id-000001")
	if err == nil || !strings.Contains(err.Error(), "不允许删除") {
		t.Fatalf("expected protected user error, got: %v", err)
	}
}

func TestDeleteMasterAdmin_NonMaster(t *testing.T) {
	svc, ur, _, _, _ := newTestUserService()
	ur.seed(&userModel.User{ID: "u1", Username: "normal", IsMaster: false})

	err := svc.DeleteMasterAdmin(context.Background(), "u1")
	if err == nil || !strings.Contains(err.Error(), "不是系统管理员") {
		t.Fatalf("expected not master error, got: %v", err)
	}
}

func TestDeleteMasterAdmin_Success(t *testing.T) {
	svc, ur, _, _, urr := newTestUserService()
	ur.seed(&userModel.User{ID: "u2", Username: "admin2", IsMaster: true})
	urr.userRoles["u2"] = []string{"sa-role"}

	if err := svc.DeleteMasterAdmin(context.Background(), "u2"); err != nil {
		t.Fatalf("DeleteMasterAdmin error: %v", err)
	}
	if _, err := ur.GetByID(context.Background(), "u2"); err == nil {
		t.Fatal("expected user to be deleted")
	}
	if len(urr.userRoles["u2"]) != 0 {
		t.Fatal("expected roles to be unbound")
	}
}

func TestPermanentDeleteMasterAdmin_ProtectedUserBlocked(t *testing.T) {
	svc, ur, _, _, _ := newTestUserService()
	ur.seed(&userModel.User{ID: "admin-id-000001", Username: "admin", IsMaster: true})

	err := svc.PermanentDeleteMasterAdmin(context.Background(), "admin-id-000001")
	if err == nil || !strings.Contains(err.Error(), "不允许永久删除") {
		t.Fatalf("expected protected user error, got: %v", err)
	}
}

// ---------- ChangePassword ----------

func TestChangePassword_WrongOldPassword(t *testing.T) {
	svc, ur, _, _, _ := newTestUserService()
	hash, _ := crypto.HashPassword("oldpass")
	ur.seed(&userModel.User{ID: "u1", Username: "alice", Password: hash})

	err := svc.ChangePassword(context.Background(), "u1", "wrongpass", "newpass")
	if err == nil || !strings.Contains(err.Error(), "原密码错误") {
		t.Fatalf("expected wrong old password error, got: %v", err)
	}
}

func TestChangePassword_Success(t *testing.T) {
	svc, ur, _, _, _ := newTestUserService()
	hash, _ := crypto.HashPassword("oldpass")
	ur.seed(&userModel.User{ID: "u1", Username: "alice", Password: hash})

	if err := svc.ChangePassword(context.Background(), "u1", "oldpass", "newpass"); err != nil {
		t.Fatalf("ChangePassword error: %v", err)
	}
	u, _ := ur.GetByID(context.Background(), "u1")
	if !crypto.CheckPassword("newpass", u.Password) {
		t.Fatal("expected new password to be set")
	}
}

// ---------- 租户归属校验 ----------

func TestAdminUpdateTenantUser_WrongTenant(t *testing.T) {
	svc, ur, _, _, _ := newTestUserService()
	ur.seed(&userModel.User{ID: "u1", Username: "alice", TenantID: "t1"})

	_, err := svc.AdminUpdateTenantUser(context.Background(), "t2", "u1", dto.UpdateUserReq{})
	if err == nil || !strings.Contains(err.Error(), "不属于指定租户") {
		t.Fatalf("expected tenant mismatch error, got: %v", err)
	}
}

func TestAdminDeleteTenantUser_WrongTenant(t *testing.T) {
	svc, ur, _, _, _ := newTestUserService()
	ur.seed(&userModel.User{ID: "u1", Username: "alice", TenantID: "t1"})

	err := svc.AdminDeleteTenantUser(context.Background(), "t2", "u1")
	if err == nil || !strings.Contains(err.Error(), "不属于指定租户") {
		t.Fatalf("expected tenant mismatch error, got: %v", err)
	}
}

func TestGetMasterAdmin_NonMaster(t *testing.T) {
	svc, ur, _, _, _ := newTestUserService()
	ur.seed(&userModel.User{ID: "u1", Username: "normal", IsMaster: false})

	_, err := svc.GetMasterAdmin(context.Background(), "u1")
	if err == nil || !strings.Contains(err.Error(), "不是系统管理员") {
		t.Fatalf("expected not master error, got: %v", err)
	}
}

func TestBelongsToTenant(t *testing.T) {
	svc, ur, _, _, _ := newTestUserService()
	ur.seed(&userModel.User{ID: "u1", Username: "alice", TenantID: "t1"})

	if !svc.BelongsToTenant(context.Background(), "u1", "t1") {
		t.Fatal("expected true for matching tenant")
	}
	if svc.BelongsToTenant(context.Background(), "u1", "t2") {
		t.Fatal("expected false for different tenant")
	}
	if svc.BelongsToTenant(context.Background(), "missing", "t1") {
		t.Fatal("expected false for missing user")
	}
}

// ---------- checkUserQuota 边界 ----------

func TestCheckUserQuota_NoVerifierSkipped(t *testing.T) {
	svc, _, _, _, _ := newTestUserService()
	// quotaVerifier 为 nil，应跳过校验直接返回 nil
	if err := svc.checkUserQuota(context.Background(), "t1"); err != nil {
		t.Fatalf("expected nil when no verifier, got: %v", err)
	}
}

func TestCheckUserQuota_GenericError(t *testing.T) {
	svc, _, _, _, _ := newTestUserService()
	svc.SetQuotaVerifier(quotaVerifierFunc(func(ctx context.Context, tenantID string) (bool, int64, int, error) {
		return false, 0, 0, errGenericMock
	}))
	if err := svc.checkUserQuota(context.Background(), "t1"); err != errGenericMock {
		t.Fatalf("expected generic error propagated, got: %v", err)
	}
}
