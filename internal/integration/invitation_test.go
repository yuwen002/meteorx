package integration

import (
	"net/http"
	"testing"

	invModel "meteorx/internal/modules/invitation/model"
	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_Invitation_CreateAcceptFlow 整体验证邀请模块：
//   - 管理端创建邀请（需 invitation:create 权限，验证 AutoRequirePermission 真拦截）
//   - 公开 info/accept：被邀请人凭令牌查看并注册加入租户，随后可正常登录
//   - 负向：无权限 403、非法令牌 404、重复接受 400
func TestInteg_Invitation_CreateAcceptFlow(t *testing.T) {
	app := testkit.NewTestApp(t)
	seedTenantAdminRole(t, app.DB)

	const tenant = "t-inv"
	// 先注册一个用户以建立租户（accept 后登录需租户处于启用态）
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/auth/register",
		`{"tenant_id":"`+tenant+`","username":"inviter123","password":"pass1234","nickname":"邀请人","email":"inv@x.com"}`, "").Code)

	// 授予邀请权限（修复后推导码为单数 invitation:create / invitation:list）
	grantPermissions(t, app.DB, "u-inv", tenant, "invitation:create", "invitation:list")
	invAuth := app.AuthHeader("u-inv", tenant, []string{"member"})

	// 无权限用户创建邀请 → 403
	noPerm := app.AuthHeader("u-noperm", tenant, []string{"member"})
	assert.Equal(t, http.StatusForbidden, app.Do(t, http.MethodPost, "/api/v1/invitations",
		`{"email":"x@x.com","role_ids":["role-tenant-admin"]}`, noPerm).Code, "无 invitation:create 应 403")

	// 创建邀请（分配 tenant_admin 角色）
	const invitee = "newbie@example.com"
	w := app.Do(t, http.MethodPost, "/api/v1/invitations",
		`{"email":"`+invitee+`","role_ids":["role-tenant-admin"]}`, invAuth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 从 DB 取邀请令牌（响应不回传 token，仅邮件下发）
	var inv invModel.Invitation
	require.NoError(t, app.DB.Where("email = ?", invitee).First(&inv).Error)
	require.NotEmpty(t, inv.Token, "邀请令牌应已生成")

	// 公开 info：凭令牌查看邀请信息（无需登录）
	w = app.Do(t, http.MethodGet, "/api/v1/invitations/info?token="+inv.Token, "", "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), invitee)

	// 公开 accept：被邀请人注册加入租户
	w = app.Do(t, http.MethodPost, "/api/v1/invitations/accept",
		`{"token":"`+inv.Token+`","username":"newbie123","password":"pass1234","nickname":"新人"}`, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 新账号可正常登录（已加入租户）
	w = app.Do(t, http.MethodPost, "/api/v1/auth/login",
		`{"tenant_id":"`+tenant+`","username":"newbie123","password":"pass1234"}`, "")
	require.Equal(t, http.StatusOK, w.Code, "接受邀请后的新账号应可登录")

	// 重复接受 → 400（邀请已被使用）
	w = app.Do(t, http.MethodPost, "/api/v1/invitations/accept",
		`{"token":"`+inv.Token+`","username":"newbie999","password":"pass1234","nickname":"重复"}`, "")
	assert.Equal(t, http.StatusBadRequest, w.Code, "重复接受应 400")

	// 非法令牌 info → 404
	assert.Equal(t, http.StatusNotFound, app.Do(t, http.MethodGet, "/api/v1/invitations/info?token=bogus", "", "").Code)
}
