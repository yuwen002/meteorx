package handler_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/user/handler"
	"meteorx/internal/modules/user/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newImportRouter 只挂载导入相关的两个端点，模拟真实挂载形态。
func newImportRouter(stub handler.UserService) http.Handler {
	h := handler.NewUserHandler(stub)
	r := chi.NewRouter()
	r.Route("/users", func(r chi.Router) {
		r.Post("/import", h.ImportUsers)
	})
	r.Route("/admin/tenant-users", func(r chi.Router) {
		r.Post("/import", h.AdminImportTenantUsers)
	})
	return r
}

// multipartForm 构造导入请求体：filename 为上传文件名，fields 为普通表单字段。
func multipartForm(t *testing.T, filename string, content []byte, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	if filename != "" {
		part, err := w.CreateFormFile("file", filename)
		require.NoError(t, err)
		_, err = part.Write(content)
		require.NoError(t, err)
	}
	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}
	require.NoError(t, w.Close())
	return body, w.FormDataContentType()
}

// doImport 发送带登录态的 multipart 请求。
func doImport(t *testing.T, router http.Handler, path, filename string, content []byte, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	return doImportAs(t, router, path, filename, content, fields, "tenant-1")
}

// doImportAs 与 doImport 相同，但可指定登录态租户（用于 SYSTEM_ROOT 这类特殊分支）。
func doImportAs(t *testing.T, router http.Handler, path, filename string, content []byte, fields map[string]string, tenantID string) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := multipartForm(t, filename, content, fields)
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", contentType)
	req = req.WithContext(contextx.SetVars(req.Context(), tenantID, "user-1", []string{"admin"}))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

const sampleCSV = "username,nickname,email,roles\nalice,Alice,alice@example.com,member\nbobby,Bob,bob@example.com,member\n"

// ---------- POST /users/import ----------

func TestImportUsers_Success(t *testing.T) {
	stub := &stubUserService{ImportResult: &service.ImportResult{
		Total:   2,
		Created: 2,
		Users:   []service.ImportUserResult{{Row: 2, Username: "alice", InitialPassword: "Abc123456789"}},
	}}
	router := newImportRouter(stub)

	w := doImport(t, router, "/users/import", "users.csv", []byte(sampleCSV), map[string]string{"role_ids": "r1,r2"})

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	var resp struct {
		Data service.ImportResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Data.Created)
	// 租户ID 取自登录态，默认角色按逗号拆分
	assert.Equal(t, "tenant-1", stub.GotTenantID)
	assert.Equal(t, []string{"r1", "r2"}, stub.GotRoleIDs)
	require.Len(t, stub.GotImportRows, 2)
	assert.Equal(t, "alice", stub.GotImportRows[0].Username)
	assert.Equal(t, "member", stub.GotImportRows[0].RoleCodes[0])
	// 行号按文件物理行号回传
	assert.Equal(t, 2, stub.GotImportRows[0].LineNo)
	assert.Equal(t, 3, stub.GotImportRows[1].LineNo)
}

func TestImportUsers_RowErrorsReturnedAsDetails(t *testing.T) {
	stub := &stubUserService{ImportRowErrs: []service.ImportRowError{
		{Row: 2, Field: "username", Message: "用户名已被使用"},
		{Row: 3, Field: "roles", Message: "角色编码 'ghost' 不存在"},
	}}
	router := newImportRouter(stub)

	w := doImport(t, router, "/users/import", "users.csv", []byte(sampleCSV), nil)

	require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	var resp struct {
		Code    string                  `json:"code"`
		Message string                  `json:"message"`
		Details []service.ImportRowError `json:"details"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_PARAM", resp.Code)
	assert.Contains(t, resp.Message, "2 处问题")
	require.Len(t, resp.Details, 2)
	assert.Equal(t, 3, resp.Details[1].Row)
}

func TestImportUsers_ValidationErrorPassedThrough(t *testing.T) {
	stub := &stubUserService{Err: &service.ValidationError{Msg: "套餐用户数上限 10，当前已用 9"}}
	router := newImportRouter(stub)

	w := doImport(t, router, "/users/import", "users.csv", []byte(sampleCSV), nil)

	assertFailCode(t, w, http.StatusBadRequest, "套餐用户数上限")
}

func TestImportUsers_InternalError(t *testing.T) {
	stub := &stubUserService{Err: assertErr("db down")}
	router := newImportRouter(stub)

	w := doImport(t, router, "/users/import", "users.csv", []byte(sampleCSV), nil)

	assertFailCode(t, w, http.StatusInternalServerError, "")
}

func TestImportUsers_UnauthorizedWithoutTenant(t *testing.T) {
	router := newImportRouter(&stubUserService{})
	body, contentType := multipartForm(t, "users.csv", []byte(sampleCSV), nil)

	req := httptest.NewRequest(http.MethodPost, "/users/import", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req) // 无 contextx 变量

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestImportUsers_SystemTenantRejected(t *testing.T) {
	stub := &stubUserService{ImportResult: &service.ImportResult{Total: 2, Created: 2}}
	router := newImportRouter(stub)

	// 系统管理员的登录态租户是虚拟租户，导入到这里等于把账号丢进一个没人能登录的租户
	w := doImportAs(t, router, "/users/import", "users.csv", []byte(sampleCSV), nil, contextx.SystemTenantID)

	assertFailCode(t, w, http.StatusBadRequest, "跨租户接口")
	assert.Empty(t, stub.GotImportRows, "系统租户下不得落库到租户接口")
}

func TestImportUsers_UnsupportedExtension(t *testing.T) {
	router := newImportRouter(&stubUserService{})

	w := doImport(t, router, "/users/import", "users.txt", []byte(sampleCSV), nil)

	assertFailCode(t, w, http.StatusBadRequest, "仅支持 .csv 或 .xlsx 文件")
}

func TestImportUsers_MissingHeader(t *testing.T) {
	router := newImportRouter(&stubUserService{})

	w := doImport(t, router, "/users/import", "users.csv", []byte("nickname,email\nAlice,a@example.com\n"), nil)

	assertFailCode(t, w, http.StatusBadRequest, "用户名")
}

func TestImportUsers_EmptyFile(t *testing.T) {
	router := newImportRouter(&stubUserService{})

	w := doImport(t, router, "/users/import", "users.csv", []byte(""), nil)

	assertFailCode(t, w, http.StatusBadRequest, "文件内容为空")
}

func TestImportUsers_NoFileField(t *testing.T) {
	router := newImportRouter(&stubUserService{})

	// 只提交普通字段，不带 file
	w := doImport(t, router, "/users/import", "", nil, map[string]string{"role_ids": "r1"})

	assertFailCode(t, w, http.StatusBadRequest, "未找到上传文件")
}

func TestImportUsers_WarningsMerged(t *testing.T) {
	stub := &stubUserService{ImportResult: &service.ImportResult{Total: 1, Created: 1}}
	router := newImportRouter(stub)

	// 含未知列 → importer 产生告警，应与服务层告警合并后返回
	csv := "username,nickname,手机号\nalice,Alice,13800000000\n"
	w := doImport(t, router, "/users/import", "users.csv", []byte(csv), nil)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	var resp struct {
		Data service.ImportResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Warnings, 1)
	assert.Contains(t, resp.Data.Warnings[0], "手机号")
}

// ---------- POST /admin/tenant-users/import ----------

func TestAdminImportTenantUsers_RequiresTenantID(t *testing.T) {
	router := newImportRouter(&stubUserService{})

	w := doImport(t, router, "/admin/tenant-users/import", "users.csv", []byte(sampleCSV), nil)

	assertFailCode(t, w, http.StatusBadRequest, "租户ID不能为空")
}

func TestAdminImportTenantUsers_UsesFormTenantID(t *testing.T) {
	stub := &stubUserService{ImportResult: &service.ImportResult{Total: 2, Created: 2}}
	router := newImportRouter(stub)

	w := doImport(t, router, "/admin/tenant-users/import", "users.csv", []byte(sampleCSV), map[string]string{
		"tenant_id": "tenant-9",
		"role_ids":  "r9",
	})

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	assert.Equal(t, "tenant-9", stub.GotTenantID)
	// 角色ID 属于其他租户，管理端入口必须忽略，避免跨租户角色被挂到目标租户用户上
	assert.Empty(t, stub.GotRoleIDs)
}

// assertErr 构造一个普通错误（避免引入额外依赖）。
type assertErr string

func (e assertErr) Error() string { return string(e) }
