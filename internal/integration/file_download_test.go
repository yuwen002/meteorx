package integration

import (
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_File_DownloadAndIsolation 整体验证文件模块下载链路（AutoRequirePermission）：
//   - 上传 → 同租户下载成功（200）
//   - 跨租户下载 → 404（tenant 作用域隔离）
//   - 无 file:download 权限 → 403
func TestInteg_File_DownloadAndIsolation(t *testing.T) {
	app := testkit.NewTestApp(t)

	// 上传者：file:upload + file:list + file:download
	grantPermissions(t, app.DB, "u-fdl", "t-file", "file:upload", "file:list", "file:download")
	uploader := app.AuthHeader("u-fdl", "t-file", []string{"member"})

	// 上传文本文件
	content := []byte("hello meteorx download")
	w := doMultipart(t, app, http.MethodPost, "/api/v1/files/upload", "file", "note.txt", "text/plain", content, uploader)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	fileID := decodeID(t, w)

	// 同租户下载 → 200
	w = app.Do(t, http.MethodGet, "/api/v1/files/"+fileID+"/download", "", uploader)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, string(content), w.Body.String())

	// 跨租户下载 → 404（即使对方也有 file:download 权限，文件不归属其租户）
	grantPermissions(t, app.DB, "u-fdl-other", "t-other", "file:download")
	other := app.AuthHeader("u-fdl-other", "t-other", []string{"member"})
	assert.Equal(t, http.StatusNotFound, app.Do(t, http.MethodGet, "/api/v1/files/"+fileID+"/download", "", other).Code,
		"跨租户下载应 404")

	// 无 file:download 权限 → 403
	grantPermissions(t, app.DB, "u-fdl-noperm", "t-file", "file:list")
	noPerm := app.AuthHeader("u-fdl-noperm", "t-file", []string{"member"})
	assert.Equal(t, http.StatusForbidden, app.Do(t, http.MethodGet, "/api/v1/files/"+fileID+"/download", "", noPerm).Code,
		"无 file:download 应 403")
}
