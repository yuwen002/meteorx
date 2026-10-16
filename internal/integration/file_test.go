package integration

import (
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_File_UploadAndPerm 整体验证文件模块（受保护 + AutoRequirePermission）：
// 有 file:upload/list 的用户可上传 multipart 文件并查询；无权限用户 → 403。
func TestInteg_File_UploadAndPerm(t *testing.T) {
	app := testkit.NewTestApp(t)
	grantPermissions(t, app.DB, "u-file", "t-1", "file:list", "file:upload")
	grantAuth := app.AuthHeader("u-file", "t-1", []string{"member"})
	noAuth := app.AuthHeader("u-none", "t-1", []string{"member"})

	// 上传（image/png 在默认白名单；无订阅租户配额 -1 不限）
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	w := doMultipart(t, app, http.MethodPost, "/api/v1/files/upload", "file", "a.png", "image/png", png, grantAuth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	id := decodeID(t, w)

	// 列表可查
	w = app.Do(t, http.MethodGet, "/api/v1/files", "", grantAuth)
	assert.Contains(t, w.Body.String(), id)

	// 无 file:list → 403
	w = app.Do(t, http.MethodGet, "/api/v1/files", "", noAuth)
	assert.Equal(t, http.StatusForbidden, w.Code, "无 file:list 应 403")
}
