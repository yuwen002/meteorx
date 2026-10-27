package service

import (
	"context"
	"strings"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/rbac/dto"
	"meteorx/internal/modules/rbac/model"
)

func newTestRBAC() (*RBACService, *mockRoleRepo, *mockPermissionRepo, *mockRolePermRepo, *mockUserRoleRepo) {
	rr := newMockRoleRepo()
	pr := newMockPermissionRepo()
	rpr := newMockRolePermRepo()
	urr := newMockUserRoleRepo()
	svc := NewRBACService(rr, pr, rpr, urr, nil, nil)
	return svc, rr, pr, rpr, urr
}

func seedRole(rr *mockRoleRepo, id, code, scope string, isSystem bool) *model.Role {
	r := &model.Role{ID: id, Name: code, Code: code, Scope: scope, IsSystem: isSystem, Status: model.RoleStatusEnabled, TenantID: contextx.SystemTenantID}
	rr.seed(r)
	return r
}

// ---------- Role ----------

func TestCreateRole_DefaultScopeTenant(t *testing.T) {
	svc, _, _, _, _ := newTestRBAC()
	r, err := svc.CreateRole(context.Background(), dto.CreateRoleReq{Code: "editor", Name: "编辑"})
	if err != nil {
		t.Fatalf("CreateRole error: %v", err)
	}
	if r.Scope != model.RoleScopeTenant {
		t.Fatalf("expected default scope tenant, got %s", r.Scope)
	}
	if r.Status != model.RoleStatusEnabled {
		t.Fatalf("expected default status enabled, got %d", r.Status)
	}
	if r.TenantID == "" {
		t.Fatal("expected system tenant id for empty tenant")
	}
}

func TestCreateRole_CodeExists(t *testing.T) {
	svc, rr, _, _, _ := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)

	_, err := svc.CreateRole(context.Background(), dto.CreateRoleReq{Code: "editor", Name: "编辑"})
	if err == nil || !strings.Contains(err.Error(), "角色编码已存在") {
		t.Fatalf("expected code exists error, got: %v", err)
	}
}

func TestCreateRole_ExplicitScopeSystem(t *testing.T) {
	svc, _, _, _, _ := newTestRBAC()
	r, err := svc.CreateRole(context.Background(), dto.CreateRoleReq{Code: "sys", Name: "系统", Scope: model.RoleScopeSystem})
	if err != nil {
		t.Fatalf("CreateRole error: %v", err)
	}
	if r.Scope != model.RoleScopeSystem {
		t.Fatalf("expected scope system, got %s", r.Scope)
	}
}

func TestUpdateRole_SystemRoleProtected(t *testing.T) {
	svc, rr, _, _, _ := newTestRBAC()
	seedRole(rr, "r1", "superadmin", model.RoleScopeSystem, true)

	_, err := svc.UpdateRole(context.Background(), "r1", dto.UpdateRoleReq{Name: "new"})
	if err == nil || !strings.Contains(err.Error(), "系统内置角色不可修改") {
		t.Fatalf("expected system role protected, got: %v", err)
	}
}

func TestUpdateRole_Success(t *testing.T) {
	svc, rr, _, _, _ := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)

	r, err := svc.UpdateRole(context.Background(), "r1", dto.UpdateRoleReq{Name: "高级编辑", Code: "editor", Scope: model.RoleScopeAll})
	if err != nil {
		t.Fatalf("UpdateRole error: %v", err)
	}
	if r.Name != "高级编辑" || r.Scope != model.RoleScopeAll {
		t.Fatalf("unexpected role: %+v", r)
	}
}

func TestDeleteRole_SystemRoleProtected(t *testing.T) {
	svc, rr, _, _, _ := newTestRBAC()
	seedRole(rr, "r1", "superadmin", model.RoleScopeSystem, true)

	err := svc.DeleteRole(context.Background(), "r1")
	if err == nil || !strings.Contains(err.Error(), "系统内置角色不可删除") {
		t.Fatalf("expected system role protected, got: %v", err)
	}
}

func TestDeleteRole_InUseByUsers(t *testing.T) {
	svc, rr, _, rpr, urr := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	urr.roleUsers["r1"] = []string{"u1", "u2"}

	err := svc.DeleteRole(context.Background(), "r1")
	if err == nil || !strings.Contains(err.Error(), "个用户使用") {
		t.Fatalf("expected in-use error, got: %v", err)
	}
	_ = rpr
}

func TestDeleteRole_HasPerms(t *testing.T) {
	svc, rr, _, rpr, _ := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	rpr.countByRole["r1"] = 3

	err := svc.DeleteRole(context.Background(), "r1")
	if err == nil || !strings.Contains(err.Error(), "个权限") {
		t.Fatalf("expected bound perm error, got: %v", err)
	}
}

func TestDeleteRole_Success(t *testing.T) {
	svc, rr, _, _, _ := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)

	if err := svc.DeleteRole(context.Background(), "r1"); err != nil {
		t.Fatalf("DeleteRole error: %v", err)
	}
	if _, err := rr.GetByID(context.Background(), "r1"); err == nil {
		t.Fatal("expected role deleted")
	}
}

func TestUpdateRoleStatus_SystemRoleProtected(t *testing.T) {
	svc, rr, _, _, _ := newTestRBAC()
	seedRole(rr, "r1", "superadmin", model.RoleScopeSystem, true)

	err := svc.UpdateRoleStatus(context.Background(), "r1", 0)
	if err == nil || !strings.Contains(err.Error(), "系统内置角色不可更改状态") {
		t.Fatalf("expected system role protected, got: %v", err)
	}
}

func TestBatchUpdateRoleStatus_Empty(t *testing.T) {
	svc, _, _, _, _ := newTestRBAC()
	_, err := svc.BatchUpdateRoleStatus(context.Background(), nil, 0)
	if err == nil || !strings.Contains(err.Error(), "不能为空") {
		t.Fatalf("expected empty list error, got: %v", err)
	}
}

func TestBatchDeleteRoles_SystemRoleProtected(t *testing.T) {
	svc, rr, _, _, _ := newTestRBAC()
	seedRole(rr, "r1", "superadmin", model.RoleScopeSystem, true)

	_, err := svc.BatchDeleteRoles(context.Background(), []string{"r1"})
	if err == nil || !strings.Contains(err.Error(), "系统内置角色") {
		t.Fatalf("expected system role protected, got: %v", err)
	}
}

func TestBatchDeleteRoles_InUse(t *testing.T) {
	svc, rr, _, _, urr := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	urr.roleUsers["r1"] = []string{"u1"}

	_, err := svc.BatchDeleteRoles(context.Background(), []string{"r1"})
	if err == nil || !strings.Contains(err.Error(), "个用户使用") {
		t.Fatalf("expected in-use error, got: %v", err)
	}
}

// ---------- Permission ----------

func TestCreatePermission_CodeExists(t *testing.T) {
	svc, _, pr, _, _ := newTestRBAC()
	pr.seed(&model.Permission{ID: "p1", Code: "user:read"})

	_, err := svc.CreatePermission(context.Background(), dto.CreatePermissionReq{Code: "user:read", Name: "读"})
	if err == nil || !strings.Contains(err.Error(), "权限编码已存在") {
		t.Fatalf("expected code exists error, got: %v", err)
	}
}

func TestCreatePermission_Success(t *testing.T) {
	svc, _, _, _, _ := newTestRBAC()
	p, err := svc.CreatePermission(context.Background(), dto.CreatePermissionReq{Code: "user:create", Name: "创建用户", Resource: "user", Action: "create"})
	if err != nil {
		t.Fatalf("CreatePermission error: %v", err)
	}
	if p.Status != model.PermissionStatusEnabled {
		t.Fatalf("expected enabled, got %d", p.Status)
	}
}

func TestSeedPermissions_Idempotent(t *testing.T) {
	svc, _, pr, _, _ := newTestRBAC()
	pr.seed(&model.Permission{ID: "p1", Code: "existing"})

	defs := []*model.Permission{
		{Code: "existing", Name: "已有"},
		{Code: "new1", Name: "新增1"},
		{Code: "new2", Name: "新增2"},
	}
	inserted, total, err := svc.SeedPermissions(context.Background(), defs)
	if err != nil {
		t.Fatalf("SeedPermissions error: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if inserted != 2 {
		t.Fatalf("expected 2 inserted (skip existing), got %d", inserted)
	}
}

func TestDeletePermission_InUseByRole(t *testing.T) {
	svc, _, pr, rpr, _ := newTestRBAC()
	pr.seed(&model.Permission{ID: "p1", Code: "user:read"})
	rpr.countByPerm["p1"] = 2

	err := svc.DeletePermission(context.Background(), "p1")
	if err == nil || !strings.Contains(err.Error(), "个角色使用") {
		t.Fatalf("expected in-use error, got: %v", err)
	}
}

func TestDeletePermission_Success(t *testing.T) {
	svc, _, pr, _, _ := newTestRBAC()
	pr.seed(&model.Permission{ID: "p1", Code: "user:read"})

	if err := svc.DeletePermission(context.Background(), "p1"); err != nil {
		t.Fatalf("DeletePermission error: %v", err)
	}
	if _, err := pr.GetByID(context.Background(), "p1"); err == nil {
		t.Fatal("expected permission deleted")
	}
}

// ---------- Role Permission ----------

func TestBindRolePermissions_RoleNotFound(t *testing.T) {
	svc, _, pr, _, _ := newTestRBAC()
	pr.seed(&model.Permission{ID: "p1", Code: "user:read"})

	err := svc.BindRolePermissions(context.Background(), "no-role", dto.BindRolePermissionsReq{PermissionIDs: []string{"p1"}})
	if err == nil || !strings.Contains(err.Error(), "角色不存在") {
		t.Fatalf("expected role not found, got: %v", err)
	}
}

func TestBindRolePermissions_PermissionNotFound(t *testing.T) {
	svc, rr, _, _, _ := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)

	err := svc.BindRolePermissions(context.Background(), "r1", dto.BindRolePermissionsReq{PermissionIDs: []string{"no-perm"}})
	if err == nil || !strings.Contains(err.Error(), "权限ID不存在") {
		t.Fatalf("expected permission not found, got: %v", err)
	}
}

func TestBindRolePermissions_Success(t *testing.T) {
	svc, rr, pr, rpr, urr := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	pr.seed(&model.Permission{ID: "p1", Code: "user:read"})
	// 操作者 op1 自身持有 user:read，绑定同编码权限属于子集，放行
	urr.userRoles["op1"] = []string{"op-role"}
	rpr.codesByRole["op-role"] = []string{"user:read"}

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	if err := svc.BindRolePermissions(ctx, "r1", dto.BindRolePermissionsReq{PermissionIDs: []string{"p1"}}); err != nil {
		t.Fatalf("BindRolePermissions error: %v", err)
	}
	if len(rpr.rolePerms["r1"]) != 1 || rpr.rolePerms["r1"][0] != "p1" {
		t.Fatalf("expected p1 bound, got %v", rpr.rolePerms["r1"])
	}
}

// ---------- 绑定权限子集收紧（防止绑到自身角色直接提权或先绑再分配绕过） ----------

func TestBindRolePermissions_PermissionSubsetRejected(t *testing.T) {
	svc, rr, pr, rpr, urr := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	pr.seed(&model.Permission{ID: "p1", Code: "user:read"})
	pr.seed(&model.Permission{ID: "p9", Code: "tenant:delete"})
	// 操作者 op1 只持有 user:read，绑定其未持有的 tenant:delete 应拒绝
	urr.userRoles["op1"] = []string{"op-role"}
	rpr.codesByRole["op-role"] = []string{"user:read"}

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	err := svc.BindRolePermissions(ctx, "r1", dto.BindRolePermissionsReq{PermissionIDs: []string{"p1", "p9"}})
	if err == nil || !strings.Contains(err.Error(), "未持有的权限") || !strings.Contains(err.Error(), "tenant:delete") {
		t.Fatalf("expected subset rejected naming tenant:delete, got: %v", err)
	}
	if len(rpr.rolePerms["r1"]) != 0 {
		t.Fatalf("expected no binding, got %v", rpr.rolePerms["r1"])
	}
}

func TestBindRolePermissions_SystemOperatorBypassesSubset(t *testing.T) {
	svc, rr, pr, rpr, _ := newTestRBAC()
	seedRole(rr, "r1", "tenant_admin", model.RoleScopeTenant, false)
	pr.seed(&model.Permission{ID: "p9", Code: "tenant:delete"})
	// 系统操作者不受子集限制：即便自身不持有 tenant:delete 也可绑定
	ctx := contextx.SetVars(context.Background(), contextx.SystemTenantID, "admin", nil)
	if err := svc.BindRolePermissions(ctx, "r1", dto.BindRolePermissionsReq{PermissionIDs: []string{"p9"}}); err != nil {
		t.Fatalf("expected system operator bypass subset check, got: %v", err)
	}
	if len(rpr.rolePerms["r1"]) != 1 || rpr.rolePerms["r1"][0] != "p9" {
		t.Fatalf("expected p9 bound, got %v", rpr.rolePerms["r1"])
	}
}

func TestBindRolePermissions_MissingOperatorContextRejected(t *testing.T) {
	svc, rr, pr, rpr, _ := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	pr.seed(&model.Permission{ID: "p1", Code: "user:read"})

	// 租户上下文但缺操作者 ID，应失败关闭而非默认放行
	ctx := contextx.SetVars(context.Background(), "t1", "", nil)
	err := svc.BindRolePermissions(ctx, "r1", dto.BindRolePermissionsReq{PermissionIDs: []string{"p1"}})
	if err == nil || !strings.Contains(err.Error(), "缺少操作者上下文") {
		t.Fatalf("expected missing operator context rejected, got: %v", err)
	}
	if len(rpr.rolePerms["r1"]) != 0 {
		t.Fatalf("expected no binding, got %v", rpr.rolePerms["r1"])
	}
}

func TestBatchBindRolesPermissions_PermissionSubsetRejected(t *testing.T) {
	svc, rr, pr, rpr, urr := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	pr.seed(&model.Permission{ID: "p9", Code: "tenant:delete"})
	urr.userRoles["op1"] = []string{"op-role"}
	rpr.codesByRole["op-role"] = []string{"user:read"}

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	_, err := svc.BatchBindRolesPermissions(ctx, dto.BatchBindRolesPermissionsReq{
		RoleIDs:       []string{"r1"},
		PermissionIDs: []string{"p9"},
	})
	if err == nil || !strings.Contains(err.Error(), "未持有的权限") || !strings.Contains(err.Error(), "tenant:delete") {
		t.Fatalf("expected batch subset rejected naming tenant:delete, got: %v", err)
	}
}

func TestBatchBindRolesPermissions_PermissionSubsetAllowed(t *testing.T) {
	svc, rr, pr, rpr, urr := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	pr.seed(&model.Permission{ID: "p1", Code: "user:read"})
	urr.userRoles["op1"] = []string{"op-role"}
	rpr.codesByRole["op-role"] = []string{"user:read"}

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	if _, err := svc.BatchBindRolesPermissions(ctx, dto.BatchBindRolesPermissionsReq{
		RoleIDs:       []string{"r1"},
		PermissionIDs: []string{"p1"},
	}); err != nil {
		t.Fatalf("expected batch subset allowed, got: %v", err)
	}
}

// ---------- 绑定角色归属收口（非系统操作者不能操作系统级/他租户角色的权限） ----------

func TestBindRolePermissions_SystemScopeRoleRejected(t *testing.T) {
	svc, rr, pr, rpr, urr := newTestRBAC()
	seedRole(rr, "rs", "superadmin", model.RoleScopeSystem, true)
	pr.seed(&model.Permission{ID: "p1", Code: "user:read"})
	// 操作者持有被绑权限，仍因目标为系统级角色而越权（验证归属拦截独立于子集）
	urr.userRoles["op1"] = []string{"op-role"}
	rpr.codesByRole["op-role"] = []string{"user:read"}

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	err := svc.BindRolePermissions(ctx, "rs", dto.BindRolePermissionsReq{PermissionIDs: []string{"p1"}})
	if err == nil || !strings.Contains(err.Error(), "系统级角色") {
		t.Fatalf("expected system-scope role rejected, got: %v", err)
	}
	if len(rpr.rolePerms["rs"]) != 0 {
		t.Fatalf("expected no binding, got %v", rpr.rolePerms["rs"])
	}
}

func TestBindRolePermissions_ForeignTenantRoleRejected(t *testing.T) {
	svc, rr, pr, rpr, urr := newTestRBAC()
	rr.seed(&model.Role{ID: "r9", Name: "别的租户角色", Code: "other", Scope: model.RoleScopeTenant, Status: model.RoleStatusEnabled, TenantID: "t2"})
	pr.seed(&model.Permission{ID: "p1", Code: "user:read"})
	urr.userRoles["op1"] = []string{"op-role"}
	rpr.codesByRole["op-role"] = []string{"user:read"}

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	err := svc.BindRolePermissions(ctx, "r9", dto.BindRolePermissionsReq{PermissionIDs: []string{"p1"}})
	if err == nil || !strings.Contains(err.Error(), "不属于当前租户") {
		t.Fatalf("expected foreign tenant role rejected, got: %v", err)
	}
	if len(rpr.rolePerms["r9"]) != 0 {
		t.Fatalf("expected no binding, got %v", rpr.rolePerms["r9"])
	}
}

func TestBindRolePermissions_SystemOperatorBypassesRoleScope(t *testing.T) {
	svc, rr, pr, rpr, _ := newTestRBAC()
	rr.seed(&model.Role{ID: "r9", Name: "别的租户角色", Code: "other", Scope: model.RoleScopeTenant, Status: model.RoleStatusEnabled, TenantID: "t2"})
	pr.seed(&model.Permission{ID: "p1", Code: "user:read"})

	// 系统操作者可跨租户修改角色权限，不受归属限制
	ctx := contextx.SetVars(context.Background(), contextx.SystemTenantID, "admin", nil)
	if err := svc.BindRolePermissions(ctx, "r9", dto.BindRolePermissionsReq{PermissionIDs: []string{"p1"}}); err != nil {
		t.Fatalf("expected system operator bypass role scope, got: %v", err)
	}
	if len(rpr.rolePerms["r9"]) != 1 {
		t.Fatalf("expected p1 bound, got %v", rpr.rolePerms["r9"])
	}
}

func TestBatchBindRolesPermissions_ForeignTenantRoleRejected(t *testing.T) {
	svc, rr, pr, rpr, urr := newTestRBAC()
	rr.seed(&model.Role{ID: "r9", Name: "别的租户角色", Code: "other", Scope: model.RoleScopeTenant, Status: model.RoleStatusEnabled, TenantID: "t2"})
	pr.seed(&model.Permission{ID: "p1", Code: "user:read"})
	urr.userRoles["op1"] = []string{"op-role"}
	rpr.codesByRole["op-role"] = []string{"user:read"}

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	_, err := svc.BatchBindRolesPermissions(ctx, dto.BatchBindRolesPermissionsReq{
		RoleIDs:       []string{"r9"},
		PermissionIDs: []string{"p1"},
	})
	if err == nil || !strings.Contains(err.Error(), "不属于当前租户") {
		t.Fatalf("expected batch foreign role rejected, got: %v", err)
	}
}

func TestBatchBindRolesPermissions_RoleNotFound(t *testing.T) {
	svc, _, pr, _, _ := newTestRBAC()
	pr.seed(&model.Permission{ID: "p1", Code: "user:read"})

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	_, err := svc.BatchBindRolesPermissions(ctx, dto.BatchBindRolesPermissionsReq{
		RoleIDs:       []string{"no-role"},
		PermissionIDs: []string{"p1"},
	})
	if err == nil || !strings.Contains(err.Error(), "角色ID不存在") {
		t.Fatalf("expected role not found in batch, got: %v", err)
	}
}

func TestUnbindRolePermission_RoleNotFound(t *testing.T) {
	svc, _, _, _, _ := newTestRBAC()
	err := svc.UnbindRolePermission(context.Background(), "no-role", "p1")
	if err == nil || !strings.Contains(err.Error(), "角色不存在") {
		t.Fatalf("expected role not found, got: %v", err)
	}
}

// ---------- User Role ----------

func TestGetUserPermissionCodes_MergedDedup(t *testing.T) {
	svc, _, _, rpr, urr := newTestRBAC()
	urr.userRoles["u1"] = []string{"r1", "r2"}
	rpr.codesByRole["r1"] = []string{"user:read", "user:write", "user:read"}
	rpr.codesByRole["r2"] = []string{"user:read", "file:upload"}

	codes, err := svc.GetUserPermissionCodes(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetUserPermissionCodes error: %v", err)
	}
	if len(codes) != 3 {
		t.Fatalf("expected 3 unique codes, got %v", codes)
	}
	seen := map[string]bool{}
	for _, c := range codes {
		if seen[c] {
			t.Fatalf("duplicate code: %s", c)
		}
		seen[c] = true
	}
}

func TestGetUserPermissionCodes_NoRoles(t *testing.T) {
	svc, _, _, _, urr := newTestRBAC()
	codes, err := svc.GetUserPermissionCodes(context.Background(), "u-empty")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(codes) != 0 {
		t.Fatalf("expected empty, got %v", codes)
	}
	_ = urr
}

func TestAssignUserRoles_EmptyList(t *testing.T) {
	svc, _, _, _, _ := newTestRBAC()
	err := svc.AssignUserRoles(context.Background(), "u1", dto.AssignUserRolesReq{})
	if err == nil || !strings.Contains(err.Error(), "不能为空") {
		t.Fatalf("expected empty list error, got: %v", err)
	}
}

func TestAssignUserRoles_UserNotExists(t *testing.T) {
	svc, _, _, _, urr := newTestRBAC()
	urr.userExists["u1"] = true

	err := svc.AssignUserRoles(context.Background(), "u1", dto.AssignUserRolesReq{RoleIDs: []string{"r1"}})
	if err == nil || !strings.Contains(err.Error(), "角色ID不存在") {
		t.Fatalf("expected role not found, got: %v", err)
	}
}

func TestAssignUserRoles_Success(t *testing.T) {
	svc, rr, _, _, urr := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	urr.seedUser("u1", "t1", false)

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	if err := svc.AssignUserRoles(ctx, "u1", dto.AssignUserRolesReq{RoleIDs: []string{"r1"}}); err != nil {
		t.Fatalf("AssignUserRoles error: %v", err)
	}
	if len(urr.userRoles["u1"]) != 1 || urr.userRoles["u1"][0] != "r1" {
		t.Fatalf("expected r1 assigned, got %v", urr.userRoles["u1"])
	}
}

func TestAssignUserRoles_CrossTenantRejected(t *testing.T) {
	svc, rr, _, _, urr := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	urr.seedUser("u2", "t2", false)

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	err := svc.AssignUserRoles(ctx, "u2", dto.AssignUserRolesReq{RoleIDs: []string{"r1"}})
	if err == nil || !strings.Contains(err.Error(), "不属于当前租户") {
		t.Fatalf("expected cross tenant error, got: %v", err)
	}
	if len(urr.userRoles["u2"]) != 0 {
		t.Fatalf("expected no assignment, got %v", urr.userRoles["u2"])
	}
}

func TestAssignUserRoles_MasterUserRejected(t *testing.T) {
	svc, rr, _, _, urr := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	urr.seedUser("u1", "t1", true)

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	err := svc.AssignUserRoles(ctx, "u1", dto.AssignUserRolesReq{RoleIDs: []string{"r1"}})
	if err == nil || !strings.Contains(err.Error(), "不属于当前租户") {
		t.Fatalf("expected master user rejected, got: %v", err)
	}
}

func TestAssignUserRoles_SystemRoleRejected(t *testing.T) {
	svc, rr, _, _, urr := newTestRBAC()
	seedRole(rr, "r1", "superadmin", model.RoleScopeSystem, true)
	urr.seedUser("u1", "t1", false)

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	err := svc.AssignUserRoles(ctx, "u1", dto.AssignUserRolesReq{RoleIDs: []string{"r1"}})
	if err == nil || !strings.Contains(err.Error(), "系统级角色") {
		t.Fatalf("expected system role rejected, got: %v", err)
	}
}

func TestAssignUserRoles_ForeignRoleRejected(t *testing.T) {
	svc, rr, _, _, urr := newTestRBAC()
	rr.seed(&model.Role{ID: "r9", Name: "别的租户角色", Code: "other", Scope: model.RoleScopeTenant, Status: model.RoleStatusEnabled, TenantID: "t2"})
	urr.seedUser("u1", "t1", false)

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	err := svc.AssignUserRoles(ctx, "u1", dto.AssignUserRolesReq{RoleIDs: []string{"r9"}})
	if err == nil || !strings.Contains(err.Error(), "不属于当前租户") {
		t.Fatalf("expected foreign role rejected, got: %v", err)
	}
}

func TestAssignUserRoles_SystemOperatorBypassesScope(t *testing.T) {
	svc, rr, _, _, urr := newTestRBAC()
	seedRole(rr, "r1", "superadmin", model.RoleScopeSystem, true)
	urr.seedUser("u9", "t9", false)

	ctx := contextx.SetVars(context.Background(), contextx.SystemTenantID, "admin", nil)
	if err := svc.AssignUserRoles(ctx, "u9", dto.AssignUserRolesReq{RoleIDs: []string{"r1"}}); err != nil {
		t.Fatalf("expected system operator bypass, got: %v", err)
	}
	if len(urr.userRoles["u9"]) != 1 {
		t.Fatalf("expected r1 assigned, got %v", urr.userRoles["u9"])
	}
}

// ---------- 权限子集收紧（防止持 assign 权限者自授更高角色提权） ----------

func TestAssignUserRoles_PermissionSubsetAllowed(t *testing.T) {
	svc, rr, _, rpr, urr := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	rpr.codesByRole["r1"] = []string{"user:list"}
	// 操作者 op1 自身持有 user:list 与 user:update，授予只含 user:list 的 r1 属于子集
	urr.userRoles["op1"] = []string{"op-role"}
	rpr.codesByRole["op-role"] = []string{"user:list", "user:update"}
	urr.seedUser("u1", "t1", false)

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	if err := svc.AssignUserRoles(ctx, "u1", dto.AssignUserRolesReq{RoleIDs: []string{"r1"}}); err != nil {
		t.Fatalf("expected subset allowed, got: %v", err)
	}
	if len(urr.userRoles["u1"]) != 1 {
		t.Fatalf("expected r1 assigned, got %v", urr.userRoles["u1"])
	}
}

func TestAssignUserRoles_PermissionSubsetRejected(t *testing.T) {
	svc, rr, _, rpr, urr := newTestRBAC()
	seedRole(rr, "r2", "adminish", model.RoleScopeTenant, false)
	rpr.codesByRole["r2"] = []string{"user:list", "tenant:delete"}
	// 操作者 op1 只持有 user:list，r2 含其未持有的 tenant:delete，应拒绝
	urr.userRoles["op1"] = []string{"op-role"}
	rpr.codesByRole["op-role"] = []string{"user:list"}
	urr.seedUser("u1", "t1", false)

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	err := svc.AssignUserRoles(ctx, "u1", dto.AssignUserRolesReq{RoleIDs: []string{"r2"}})
	if err == nil || !strings.Contains(err.Error(), "未持有的权限") || !strings.Contains(err.Error(), "tenant:delete") {
		t.Fatalf("expected subset rejected naming tenant:delete, got: %v", err)
	}
	if len(urr.userRoles["u1"]) != 0 {
		t.Fatalf("expected no assignment, got %v", urr.userRoles["u1"])
	}
}

func TestAssignUserRoles_SystemOperatorBypassesSubset(t *testing.T) {
	svc, rr, _, rpr, urr := newTestRBAC()
	seedRole(rr, "r1", "tenant_admin", model.RoleScopeTenant, false)
	rpr.codesByRole["r1"] = []string{"tenant:delete"}
	// 系统操作者不受子集限制：即便自己不“持有” tenant:delete（无角色）也可授予
	urr.seedUser("u9", "t9", false)

	ctx := contextx.SetVars(context.Background(), contextx.SystemTenantID, "admin", nil)
	if err := svc.AssignUserRoles(ctx, "u9", dto.AssignUserRolesReq{RoleIDs: []string{"r1"}}); err != nil {
		t.Fatalf("expected system operator bypass subset check, got: %v", err)
	}
	if len(urr.userRoles["u9"]) != 1 {
		t.Fatalf("expected r1 assigned, got %v", urr.userRoles["u9"])
	}
}

func TestBatchAssignUserRoles_PermissionSubsetRejectedWholeBatch(t *testing.T) {
	svc, rr, _, rpr, urr := newTestRBAC()
	seedRole(rr, "r2", "adminish", model.RoleScopeTenant, false)
	rpr.codesByRole["r2"] = []string{"user:list", "tenant:delete"}
	urr.userRoles["op1"] = []string{"op-role"}
	rpr.codesByRole["op-role"] = []string{"user:list"}
	urr.seedUser("u1", "t1", false)
	urr.seedUser("u2", "t1", false)

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	_, err := svc.BatchAssignUserRoles(ctx, dto.BatchAssignUserRolesReq{UserRoleAssignments: []dto.UserRoleAssignment{
		{UserID: "u1", RoleIDs: []string{"r2"}},
		{UserID: "u2", RoleIDs: []string{"r2"}},
	}})
	if err == nil || !strings.Contains(err.Error(), "未持有的权限") {
		t.Fatalf("expected whole batch rejected, got: %v", err)
	}
	if len(urr.userRoles["u1"]) != 0 || len(urr.userRoles["u2"]) != 0 {
		t.Fatalf("expected no assignment, got u1=%v u2=%v", urr.userRoles["u1"], urr.userRoles["u2"])
	}
}

func TestBatchAssignUserRoles_CrossTenantRejectedWholeBatch(t *testing.T) {
	svc, rr, _, _, urr := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	urr.seedUser("u1", "t1", false)
	urr.seedUser("u2", "t2", false)

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	_, err := svc.BatchAssignUserRoles(ctx, dto.BatchAssignUserRolesReq{UserRoleAssignments: []dto.UserRoleAssignment{
		{UserID: "u1", RoleIDs: []string{"r1"}},
		{UserID: "u2", RoleIDs: []string{"r1"}},
	}})
	if err == nil || !strings.Contains(err.Error(), "不属于当前租户") {
		t.Fatalf("expected cross tenant error, got: %v", err)
	}
	if len(urr.userRoles["u1"]) != 0 || len(urr.userRoles["u2"]) != 0 {
		t.Fatalf("expected whole batch rejected, got u1=%v u2=%v", urr.userRoles["u1"], urr.userRoles["u2"])
	}
}

func TestBatchAssignUserRoles_Success(t *testing.T) {
	svc, rr, _, _, urr := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	urr.seedUser("u1", "t1", false)
	urr.seedUser("u2", "t1", false)

	ctx := contextx.SetVars(context.Background(), "t1", "op1", nil)
	assigned, err := svc.BatchAssignUserRoles(ctx, dto.BatchAssignUserRolesReq{UserRoleAssignments: []dto.UserRoleAssignment{
		{UserID: "u1", RoleIDs: []string{"r1"}},
		{UserID: "u2", RoleIDs: []string{"r1"}},
	}})
	if err != nil {
		t.Fatalf("BatchAssignUserRoles error: %v", err)
	}
	if assigned != 2 {
		t.Fatalf("expected 2 assigned, got %d", assigned)
	}
	if len(urr.userRoles["u1"]) != 1 || len(urr.userRoles["u2"]) != 1 {
		t.Fatalf("expected both assigned, got u1=%v u2=%v", urr.userRoles["u1"], urr.userRoles["u2"])
	}
}

func TestGetUserRoles_FiltersMissing(t *testing.T) {
	svc, rr, _, _, urr := newTestRBAC()
	seedRole(rr, "r1", "editor", model.RoleScopeTenant, false)
	urr.userExists["u1"] = true
	urr.userRoles["u1"] = []string{"r1", "missing"}

	roles, err := svc.GetUserRoles(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetUserRoles error: %v", err)
	}
	if len(roles) != 1 || roles[0].ID != "r1" {
		t.Fatalf("expected only r1, got %+v", roles)
	}
}

func TestCountUserPermissions_Dedup(t *testing.T) {
	svc, _, _, rpr, urr := newTestRBAC()
	urr.userRoles["u1"] = []string{"r1", "r2"}
	rpr.codesByRole["r1"] = []string{"a", "b"}
	rpr.codesByRole["r2"] = []string{"b", "c"}

	n, err := svc.CountUserPermissions(context.Background(), "u1")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if n != 3 {
		t.Fatalf("expected 3 unique, got %d", n)
	}
}

func TestCountUserPermissions_EmptyUserID(t *testing.T) {
	svc, _, _, _, _ := newTestRBAC()
	n, err := svc.CountUserPermissions(context.Background(), "")
	if err != nil || n != 0 {
		t.Fatalf("expected 0,nil for empty userID, got %d,%v", n, err)
	}
}
