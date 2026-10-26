package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"meteorx/internal/config"
	rbacModel "meteorx/internal/modules/rbac/model"
	"meteorx/internal/modules/user/importer"
	userModel "meteorx/internal/modules/user/model"
)

// ---------- 内存 mock：ImportRepository ----------

type mockImportRepo struct {
	active    map[string]struct{}
	trash     map[string]struct{}
	checkErr  error
	createErr error

	created []*userModel.User
	links   map[string][]string
}

func newMockImportRepo() *mockImportRepo {
	return &mockImportRepo{
		active: make(map[string]struct{}),
		trash:  make(map[string]struct{}),
		links:  make(map[string][]string),
	}
}

func (m *mockImportRepo) CheckUsernames(_ context.Context, _ string, usernames []string) (map[string]struct{}, map[string]struct{}, error) {
	if m.checkErr != nil {
		return nil, nil, m.checkErr
	}
	active := make(map[string]struct{})
	trash := make(map[string]struct{})
	for _, name := range usernames {
		if _, ok := m.active[name]; ok {
			active[name] = struct{}{}
		}
		if _, ok := m.trash[name]; ok {
			trash[name] = struct{}{}
		}
	}
	return active, trash, nil
}

func (m *mockImportRepo) CreateUsersWithRoles(_ context.Context, users []*userModel.User, roleIDsByUserID map[string][]string) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = append(m.created, users...)
	for uid, roleIDs := range roleIDsByUserID {
		m.links[uid] = roleIDs
	}
	return nil
}

// newImportTestService 构造带导入仓储的服务实例。
func newImportTestService() (*UserService, *mockRoleRepo, *mockImportRepo) {
	svc, _, _, rr, _ := newTestUserService()
	ir := newMockImportRepo()
	svc.SetImportRepository(ir)
	return svc, rr, ir
}

// row 构造一行导入数据。
func row(line int, username, nickname string, roles ...string) importer.Row {
	return importer.Row{LineNo: line, Username: username, Nickname: nickname, RoleCodes: roles}
}

// fieldOf 判断行错误集合中是否存在指定行与字段的错误。
func hasRowError(errs []ImportRowError, line int, field string) bool {
	for _, e := range errs {
		if e.Row == line && e.Field == field {
			return true
		}
	}
	return false
}

// ---------- ImportUsers ----------

func TestImportUsers_Success(t *testing.T) {
	svc, rr, ir := newImportTestService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1)

	result, rowErrs, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice", "member"),
		row(3, "bobby", "Bob", "member"),
	}, nil)
	if err != nil {
		t.Fatalf("ImportUsers error: %v", err)
	}
	if len(rowErrs) != 0 {
		t.Fatalf("unexpected row errors: %+v", rowErrs)
	}
	if result.Total != 2 || result.Created != 2 || len(result.Users) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(ir.created) != 2 {
		t.Fatalf("expected 2 users written, got %d", len(ir.created))
	}
	for _, u := range ir.created {
		if u.TenantID != "t1" || u.Status != 1 || u.IsMaster {
			t.Fatalf("unexpected user fields: %+v", u)
		}
		if u.Password == "" || u.Password == result.Users[0].InitialPassword {
			t.Fatalf("password should be hashed, got %q", u.Password)
		}
		if len(ir.links[u.ID]) != 1 || ir.links[u.ID][0] != "r1" {
			t.Fatalf("expected role r1 for %s, got %v", u.Username, ir.links[u.ID])
		}
	}
	// 回显的初始密码应是可直接下发的明文，长度满足默认口径
	if len(result.Users[0].InitialPassword) < defaultInitialPasswordLength {
		t.Fatalf("initial password too short: %q", result.Users[0].InitialPassword)
	}
	if result.Users[0].Row != 2 || result.Users[1].Row != 3 {
		t.Fatalf("row numbers should follow file order, got %d/%d", result.Users[0].Row, result.Users[1].Row)
	}
}

func TestImportUsers_FallbackToDefaultRole(t *testing.T) {
	svc, rr, ir := newImportTestService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1)

	result, rowErrs, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice"), // 未填角色，回落到表单默认角色
	}, []string{"r1"})
	if err != nil {
		t.Fatalf("ImportUsers error: %v", err)
	}
	if len(rowErrs) != 0 {
		t.Fatalf("unexpected row errors: %+v", rowErrs)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "未填写角色") {
		t.Fatalf("expected fallback warning, got %v", result.Warnings)
	}
	if len(ir.links[ir.created[0].ID]) != 1 {
		t.Fatalf("expected default role assigned, got %v", ir.links[ir.created[0].ID])
	}
	if len(result.Users[0].RoleCodes) != 1 || result.Users[0].RoleCodes[0] != "member" {
		t.Fatalf("unexpected role codes: %v", result.Users[0].RoleCodes)
	}
}

func TestImportUsers_NoRoleAtAll(t *testing.T) {
	svc, _, ir := newImportTestService()

	_, rowErrs, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice"),
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasRowError(rowErrs, 2, "roles") {
		t.Fatalf("expected roles error, got %+v", rowErrs)
	}
	if len(ir.created) != 0 {
		t.Fatalf("nothing should be written on row errors")
	}
}

func TestImportUsers_DuplicateInFile(t *testing.T) {
	svc, rr, ir := newImportTestService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1)

	_, rowErrs, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice", "member"),
		row(5, "alice", "Alice2", "member"),
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rowErrs) != 1 || rowErrs[0].Row != 5 || rowErrs[0].Field != "username" {
		t.Fatalf("expected duplicate at row 5, got %+v", rowErrs)
	}
	if !strings.Contains(rowErrs[0].Message, "第 2 行") {
		t.Fatalf("expected first occurrence in message, got %q", rowErrs[0].Message)
	}
	if len(ir.created) != 0 {
		t.Fatalf("whole file should be rejected")
	}
}

func TestImportUsers_UsernameTaken(t *testing.T) {
	svc, rr, ir := newImportTestService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1)
	ir.active["alice"] = struct{}{}

	_, rowErrs, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice", "member"),
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasRowError(rowErrs, 2, "username") {
		t.Fatalf("expected username taken error, got %+v", rowErrs)
	}
}

func TestImportUsers_UsernameInTrash(t *testing.T) {
	svc, rr, ir := newImportTestService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1)
	ir.trash["alice"] = struct{}{}

	_, rowErrs, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice", "member"),
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasRowError(rowErrs, 2, "username") || !strings.Contains(rowErrs[0].Message, "回收站") {
		t.Fatalf("expected trash hint, got %+v", rowErrs)
	}
}

func TestImportUsers_FieldValidation(t *testing.T) {
	svc, _, _ := newImportTestService()

	_, rowErrs, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		{LineNo: 2, Username: "ab", Nickname: "短名"},                       // 用户名过短
		{LineNo: 3, Username: "carol", Email: "not-an-email"},              // 缺昵称 + 邮箱非法
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasRowError(rowErrs, 2, "username") {
		t.Fatalf("expected username length error, got %+v", rowErrs)
	}
	if !hasRowError(rowErrs, 3, "nickname") || !hasRowError(rowErrs, 3, "email") {
		t.Fatalf("expected nickname+email errors, got %+v", rowErrs)
	}
	// 错误应按行号升序返回，便于前端逐行展示
	for i := 1; i < len(rowErrs); i++ {
		if rowErrs[i-1].Row > rowErrs[i].Row {
			t.Fatalf("row errors not sorted: %+v", rowErrs)
		}
	}
}

func TestImportUsers_QuotaNotEnough(t *testing.T) {
	svc, rr, ir := newImportTestService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1)
	svc.SetQuotaVerifier(quotaVerifierFunc(func(context.Context, string) (bool, int64, int, error) {
		return false, 9, 10, nil // 仅剩 1 个名额
	}))

	_, _, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice", "member"),
		row(3, "bobby", "Bob", "member"),
	}, nil)
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if !strings.Contains(err.Error(), "剩余可导入 1 个") {
		t.Fatalf("unexpected quota message: %v", err)
	}
	if len(ir.created) != 0 {
		t.Fatalf("nothing should be written when quota is not enough")
	}
}

// 套餐上限 -1 表示不限量，若仍按 remaining 比较会把整批导入全部拒绝。
func TestImportUsers_UnlimitedQuotaAllowsImport(t *testing.T) {
	svc, rr, _ := newImportTestService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1)
	svc.SetQuotaVerifier(quotaVerifierFunc(func(context.Context, string) (bool, int64, int, error) {
		return false, 0, -1, nil
	}))

	result, rowErrs, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice", "member"),
		row(3, "bobby", "Bob", "member"),
	}, nil)
	if err != nil || len(rowErrs) != 0 {
		t.Fatalf("unexpected failure: err=%v rowErrs=%+v", err, rowErrs)
	}
	if result.Created != 2 {
		t.Fatalf("expected 2 created, got %d", result.Created)
	}
}

func TestImportUsers_RoleCodeNotFound(t *testing.T) {
	svc, _, _ := newImportTestService()

	_, rowErrs, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice", "ghost"),
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasRowError(rowErrs, 2, "roles") || !strings.Contains(rowErrs[0].Message, "ghost") {
		t.Fatalf("expected unknown role error, got %+v", rowErrs)
	}
}

func TestImportUsers_RoleDisabledAndWrongScope(t *testing.T) {
	svc, rr, _ := newImportTestService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 0) // 已禁用
	seedTenantRole(rr, "r2", "sysrole", rbacModel.RoleScopeSystem, 1)

	_, rowErrs, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice", "member"),
		row(3, "bobby", "Bob", "sysrole"),
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasRowError(rowErrs, 2, "roles") || !strings.Contains(rowErrs[0].Message, "已禁用") {
		t.Fatalf("expected disabled role error, got %+v", rowErrs)
	}
	if !hasRowError(rowErrs, 3, "roles") || !strings.Contains(rowErrs[len(rowErrs)-1].Message, "不允许分配给租户用户") {
		t.Fatalf("expected scope error, got %+v", rowErrs)
	}
}

func TestImportUsers_PlatformRoleAllowed(t *testing.T) {
	svc, rr, _ := newImportTestService()
	seedTenantRole(rr, "r1", "auditor", rbacModel.RoleScopeAll, 1)

	result, rowErrs, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice", "auditor"),
	}, nil)
	if err != nil || len(rowErrs) != 0 {
		t.Fatalf("unexpected failure: err=%v rowErrs=%+v", err, rowErrs)
	}
	if result.Created != 1 {
		t.Fatalf("expected 1 created, got %d", result.Created)
	}
}

func TestImportUsers_WriteFailedRollsBack(t *testing.T) {
	svc, rr, ir := newImportTestService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1)
	ir.createErr = errors.New("duplicate key")

	_, _, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice", "member"),
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "已整体回滚") {
		t.Fatalf("expected rollback error, got %v", err)
	}
}

func TestImportUsers_CheckUsernamesFailed(t *testing.T) {
	svc, rr, ir := newImportTestService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1)
	ir.checkErr = errors.New("db down")

	_, _, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice", "member"),
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "检查用户名占用失败") {
		t.Fatalf("expected check error, got %v", err)
	}
}

func TestImportUsers_RepositoryNotInjected(t *testing.T) {
	svc, _, _, _, _ := newTestUserService()

	_, _, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice", "member"),
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "未启用") {
		t.Fatalf("expected not enabled error, got %v", err)
	}
}

func TestImportUsers_InvalidArguments(t *testing.T) {
	svc, _, _ := newImportTestService()

	if _, _, err := svc.ImportUsers(context.Background(), "", nil, nil); err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected tenant validation error, got %v", err)
	}
	if _, _, err := svc.ImportUsers(context.Background(), "t1", nil, nil); err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected empty rows validation error, got %v", err)
	}
}

func TestImportUsers_DefaultRoleInvalid(t *testing.T) {
	svc, _, _ := newImportTestService()

	_, _, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice"),
	}, []string{"ghost"})
	if err == nil || !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("expected default role validation error, got %v", err)
	}
}

func TestImportUsers_PasswordLengthFollowsPolicy(t *testing.T) {
	svc, rr, _ := newImportTestService()
	seedTenantRole(rr, "r1", "member", rbacModel.RoleScopeTenant, 1)
	svc.SetPasswordPolicy(config.PasswordPolicyConfig{Enabled: true, MinLength: 18})

	result, rowErrs, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		row(2, "alice", "Alice", "member"),
	}, nil)
	if err != nil || len(rowErrs) != 0 {
		t.Fatalf("unexpected failure: err=%v rowErrs=%+v", err, rowErrs)
	}
	if got := len(result.Users[0].InitialPassword); got != 20 {
		t.Fatalf("expected password length 20 (policy min+2), got %d", got)
	}
}

func TestImportUsers_InvalidRowDoesNotProduceOccupiedError(t *testing.T) {
	svc, _, ir := newImportTestService()
	ir.active["ab"] = struct{}{} // 即便库里有同名，字段校验先失败时也不应报"已被使用"

	_, rowErrs, err := svc.ImportUsers(context.Background(), "t1", []importer.Row{
		{LineNo: 2, Username: "ab", Nickname: "x"}, // 用户名长度不满足 min=4
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, e := range rowErrs {
		if strings.Contains(e.Message, "已被使用") {
			t.Fatalf("invalid row should not reach occupancy check: %+v", rowErrs)
		}
	}
}
