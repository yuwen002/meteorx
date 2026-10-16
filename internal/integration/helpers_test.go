package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"net/textproto"

	rbacModel "meteorx/internal/modules/rbac/model"
	"meteorx/internal/testkit"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedTenantAdminRole 写入注册/后台建租户所需的默认角色 tenant_admin（已存在则复用，避免种子数据冲突）。
func seedTenantAdminRole(t *testing.T, db *gorm.DB) {
	var role rbacModel.Role
	if err := db.Where("code = ?", "tenant_admin").First(&role).Error; err != nil {
		role = rbacModel.Role{ID: "role-tenant-admin", Name: "租户管理员", Code: "tenant_admin",
			TenantID: "", Scope: "tenant", Status: 1}
		require.NoError(t, db.Create(&role).Error)
	}
}

// grantPermissionToUser 为用户在指定租户下绑定一个含目标权限码的角色（绕过 API，直接落库）。
// 权限/角色可能已由种子数据预置，故存在则复用其 ID，仅在缺失时新建。
func grantPermissionToUser(t *testing.T, db *gorm.DB, userID, tenantID, permCode string) {
	var perm rbacModel.Permission
	if err := db.Where("code = ?", permCode).First(&perm).Error; err != nil {
		perm = rbacModel.Permission{ID: "perm-" + permCode, Code: permCode, Status: 1}
		require.NoError(t, db.Create(&perm).Error)
	}

	roleCode := "role-" + permCode
	var role rbacModel.Role
	if err := db.Where("code = ?", roleCode).First(&role).Error; err != nil {
		role = rbacModel.Role{ID: roleCode, Code: roleCode, TenantID: tenantID, Status: 1}
		require.NoError(t, db.Create(&role).Error)
	}

	require.NoError(t, db.Where(rbacModel.RolePermission{RoleID: role.ID, PermissionID: perm.ID}).
		FirstOrCreate(&rbacModel.RolePermission{RoleID: role.ID, PermissionID: perm.ID}).Error)
	require.NoError(t, db.Where(rbacModel.UserRole{UserID: userID, RoleID: role.ID}).
		FirstOrCreate(&rbacModel.UserRole{UserID: userID, RoleID: role.ID}).Error)
}

// grantPermissions 批量授予用户多个权限码（每个码绑定一个角色，用户权限取其并集）。
func grantPermissions(t *testing.T, db *gorm.DB, userID, tenantID string, codes ...string) {
	for _, c := range codes {
		grantPermissionToUser(t, db, userID, tenantID, c)
	}
}

// ensureTenant 通过后台 API 创建一个启用状态的租户并返回其 ID（供 plan/tenant 等测试复用）。
func ensureTenant(t *testing.T, app *testkit.TestApp, name, domain, sa string) string {
	t.Helper()
	body := `{"name":"` + name + `","domain":"` + domain + `","status":1,` +
		`"admin_user":{"username":"` + domain + `adm","password":"secret123","nickname":"A","email":"a@x.com"}}`
	w := app.Do(t, "POST", "/api/v1/admin/tenants", body, sa)
	require.Equal(t, 200, w.Code, w.Body.String())
	var out struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.NotEmpty(t, out.Data.ID)
	return out.Data.ID
}

// decodeID 从响应体 data.id 取出资源 ID。
func decodeID(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var out struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.NotEmpty(t, out.Data.ID)
	return out.Data.ID
}

// decodeToken 从登录/创建响应体 data.token 取出令牌。
func decodeToken(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var out struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.NotEmpty(t, out.Data.Token)
	return out.Data.Token
}

// doMultipart 构造带指定 Content-Type 的 multipart 上传请求并驱动完整路由。
func doMultipart(t *testing.T, app *testkit.TestApp, method, path, field, filename, contentType string, content []byte, auth string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, field, filename))
	h.Set("Content-Type", contentType)
	fw, err := mw.CreatePart(h)
	require.NoError(t, err)
	_, err = fw.Write(content)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	return w
}
