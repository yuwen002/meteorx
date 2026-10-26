package service

import (
	"context"
	"testing"

	"meteorx/internal/config"
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/auth/model"
	rbacmodel "meteorx/internal/modules/rbac/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// stubUserRoleRepo UserRoleRepository 的轻量测试桩，仅驱动 scope 收敛所需的方法。
type stubUserRoleRepo struct {
	roleCodesByUser map[string][]string
	roleIDsByUser   map[string][]string
}

func (s *stubUserRoleRepo) AssignRoles(context.Context, string, []string) error { return nil }
func (s *stubUserRoleRepo) GetRoleIDsByUserID(_ context.Context, userID string) ([]string, error) {
	return s.roleIDsByUser[userID], nil
}
func (s *stubUserRoleRepo) GetRoleCodesByUserID(_ context.Context, userID string) ([]string, error) {
	return s.roleCodesByUser[userID], nil
}
func (s *stubUserRoleRepo) BatchGetRoleIDsByUserIDs(context.Context, []string) (map[string][]string, error) {
	return nil, nil
}
func (s *stubUserRoleRepo) DeleteByUserID(context.Context, string) error { return nil }
func (s *stubUserRoleRepo) DeleteByUserIDAndRoleID(context.Context, string, string) error {
	return nil
}
func (s *stubUserRoleRepo) GetUserIDsByRoleID(context.Context, string) ([]string, error) { return nil, nil }
func (s *stubUserRoleRepo) CountByRoleID(context.Context, string) (int64, error)         { return 0, nil }
func (s *stubUserRoleRepo) CountByUserID(context.Context, string) (int64, error)         { return 0, nil }
func (s *stubUserRoleRepo) CheckUserExists(context.Context, string) error                { return nil }
func (s *stubUserRoleRepo) GetUserScope(context.Context, string) (string, bool, error) {
	return "", false, nil
}
func (s *stubUserRoleRepo) ListUserRoles(context.Context, int, int, string, string) ([]*rbacmodel.UserRole, int64, error) {
	return nil, 0, nil
}

// stubRolePermRepo RolePermissionRepository 的轻量测试桩。
type stubRolePermRepo struct {
	codesByRole map[string][]string
}

func (s *stubRolePermRepo) BindPermissions(context.Context, string, []string) error { return nil }
func (s *stubRolePermRepo) GetPermissionsByRoleID(context.Context, string) ([]*rbacmodel.Permission, error) {
	return nil, nil
}
func (s *stubRolePermRepo) GetPermissionsByRoleIDWithResource(context.Context, string, string) ([]*rbacmodel.Permission, error) {
	return nil, nil
}
func (s *stubRolePermRepo) GetPermissionCodesByRoleID(_ context.Context, roleID string) ([]string, error) {
	return s.codesByRole[roleID], nil
}
func (s *stubRolePermRepo) UnbindPermission(context.Context, string, string) error { return nil }
func (s *stubRolePermRepo) BatchBindPermissions(context.Context, []string, []string) (int64, error) {
	return 0, nil
}
func (s *stubRolePermRepo) BatchUnbindPermissions(context.Context, []string, []string) (int64, error) {
	return 0, nil
}
func (s *stubRolePermRepo) List(context.Context, int, int, string, string) ([]*rbacmodel.RolePermission, int64, error) {
	return nil, 0, nil
}
func (s *stubRolePermRepo) CountByRoleID(context.Context, string) (int64, error)         { return 0, nil }
func (s *stubRolePermRepo) CountByPermissionID(context.Context, string) (int64, error)   { return 0, nil }

// newScopedSvc 构造带 RBAC 仓储桩的 APITokenService，用于验证 scope 收敛逻辑。
func newScopedSvc(repo *mockAPITokenRepo, ur *stubUserRoleRepo, rp *stubRolePermRepo) *APITokenService {
	return NewAPITokenService(repo, nil, ur, rp, nil, config.AuthConfig{})
}

// TestValidateAPIToken_NoScope_Inherit 未配 scope 时 Scoped=false，沿用属主权限。
func TestValidateAPIToken_NoScope_Inherit(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newScopedSvc(repo, &stubUserRoleRepo{}, &stubRolePermRepo{})

	repo.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return(&model.APIToken{
		ID:       "tok-1",
		UserID:   "user-1",
		TenantID: "tenant-1",
	}, nil)
	repo.On("UpdateLastUsedAt", mock.Anything, "tok-1").Return(nil)

	p, err := svc.Validate(context.Background(), "mxat_abcdef1234567890")
	assert.NoError(t, err)
	assert.Equal(t, "user-1", p.UserID)
	assert.Equal(t, "tenant-1", p.TenantID)
	assert.False(t, p.Scoped)
	assert.Nil(t, p.Permissions)
}

// TestValidateAPIToken_ScopeIntersect scope 与属主权限取交集，越出属主的 scope 被剔除。
func TestValidateAPIToken_ScopeIntersect(t *testing.T) {
	repo := new(mockAPITokenRepo)
	ur := &stubUserRoleRepo{
		roleCodesByUser: map[string][]string{"user-1": {"member"}},
		roleIDsByUser:   map[string][]string{"user-1": {"role-basic"}},
	}
	rp := &stubRolePermRepo{
		// 属主仅拥有 user:list，未拥有 user:create
		codesByRole: map[string][]string{"role-basic": {"user:list", "task:read"}},
	}
	svc := newScopedSvc(repo, ur, rp)

	repo.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return(&model.APIToken{
		ID:       "tok-1",
		UserID:   "user-1",
		TenantID: "tenant-1",
		Scopes:   `["user:list","user:create","wiki:read"]`,
	}, nil)
	repo.On("UpdateLastUsedAt", mock.Anything, "tok-1").Return(nil)

	p, err := svc.Validate(context.Background(), "mxat_abcdef1234567890")
	assert.NoError(t, err)
	assert.True(t, p.Scoped)
	// 期望交集只保留 user:list（user:create 属主没有，wiki:read 属主没有）
	assert.ElementsMatch(t, []string{"user:list"}, p.Permissions)
}

// TestValidateAPIToken_ScopeSuperadminOwner 属主为超管时权限天然是全集，有效集合即 scopes。
func TestValidateAPIToken_ScopeSuperadminOwner(t *testing.T) {
	repo := new(mockAPITokenRepo)
	ur := &stubUserRoleRepo{
		roleCodesByUser: map[string][]string{"user-1": {"superadmin"}},
		roleIDsByUser:   map[string][]string{"user-1": {"role-super"}},
	}
	rp := &stubRolePermRepo{
		// 超管角色未显式绑定权限码（依赖旁路），验证 scope 收敛仍能直接采用 scopes
		codesByRole: map[string][]string{"role-super": {}},
	}
	svc := newScopedSvc(repo, ur, rp)

	repo.On("GetByTokenHash", mock.Anything, mock.AnythingOfType("string")).Return(&model.APIToken{
		ID:       "tok-1",
		UserID:   "user-1",
		TenantID: "tenant-1",
		Scopes:   `["admin:tenant:list","admin:tenant:create"]`,
	}, nil)
	repo.On("UpdateLastUsedAt", mock.Anything, "tok-1").Return(nil)

	p, err := svc.Validate(context.Background(), "mxat_abcdef1234567890")
	assert.NoError(t, err)
	assert.True(t, p.Scoped)
	assert.ElementsMatch(t, []string{"admin:tenant:list", "admin:tenant:create"}, p.Permissions)
}

// TestCreateAPIToken_WithScopes 创建时 scopes 落库并在响应回显。
func TestCreateAPIToken_WithScopes(t *testing.T) {
	repo := new(mockAPITokenRepo)
	svc := newScopedSvc(repo, &stubUserRoleRepo{}, &stubRolePermRepo{})

	repo.On("ListByUserID", mock.Anything, "user-1").Return([]*model.APIToken{}, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.APIToken")).
		Run(func(args mock.Arguments) {
			tk := args.Get(1).(*model.APIToken)
			assert.JSONEq(t, `["user:list","task:read"]`, tk.Scopes)
		}).Return(nil)

	resp, err := svc.Create(context.Background(), "user-1", "tenant-1", dto.CreateAPITokenReq{
		Name:   "scoped-token",
		Scopes: []string{"user:list", "task:read"},
	})
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"user:list", "task:read"}, resp.Scopes)
	repo.AssertExpectations(t)
}

// TestSerializeParseList_RoundTrip 列表序列化与解析的往返一致性与空值语义。
func TestSerializeParseList_RoundTrip(t *testing.T) {
	assert.Equal(t, "", serializeList(nil))
	assert.Nil(t, parseList(""))

	encoded := serializeList([]string{"a:b", "/x/y"})
	assert.JSONEq(t, `["a:b","/x/y"]`, encoded)
	assert.Equal(t, []string{"a:b", "/x/y"}, parseList(encoded))
}
