package integration

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"meteorx/internal/modules/export/model"
	"meteorx/internal/modules/export/repository"
	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_Export_CreateAndPerm 整体验证导出模块（受保护 + AutoRequirePermission）：
// 有 export:list/create/read 的用户可创建任务并查询；无权限用户 → 403。
func TestInteg_Export_CreateAndPerm(t *testing.T) {
	app := testkit.NewTestApp(t)
	grantPermissions(t, app.DB, "u-export", "t-1", "export:list", "export:create", "export:read")
	grantAuth := app.AuthHeader("u-export", "t-1", []string{"member"})
	noAuth := app.AuthHeader("u-none", "t-1", []string{"member"})

	// 创建导出任务
	w := app.Do(t, http.MethodPost, "/api/v1/exports", `{"resource":"user","format":"csv"}`, grantAuth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	id := decodeID(t, w)

	// 列表/详情可查
	w = app.Do(t, http.MethodGet, "/api/v1/exports", "", grantAuth)
	assert.Contains(t, w.Body.String(), id)
	w = app.Do(t, http.MethodGet, "/api/v1/exports/"+id, "", grantAuth)
	assert.Equal(t, http.StatusOK, w.Code)

	// 无 export 权限 → 403
	w = app.Do(t, http.MethodGet, "/api/v1/exports", "", noAuth)
	assert.Equal(t, http.StatusForbidden, w.Code, "无 export:list 应 403")
}

// TestInteg_Export_DeleteRemovesFile 端到端锁定删除契约：文件与记录一同消失。
// 删除动作需在 service 层按“先文件后记录”的顺序执行，旧写法在 handler 里
// `_ = os.Remove(...)` 丢弃错误，文件占用失败时记录已删、磁盘永久残留。
func TestInteg_Export_DeleteRemovesFile(t *testing.T) {
	app := testkit.NewTestApp(t)
	grantPermissions(t, app.DB, "u-export", "t-1", "export:list", "export:create", "export:delete")
	grantAuth := app.AuthHeader("u-export", "t-1", []string{"member"})

	filePath := filepath.Join(t.TempDir(), "exp1.csv")
	require.NoError(t, os.WriteFile(filePath, []byte("id,name\n1,a\n"), 0o640))
	require.NoError(t, repository.NewExportRepository(app.DB).Create(t.Context(), &model.ExportJob{
		ID: "exp-del-1", TenantID: "t-1", UserID: "u-export",
		Resource:  model.ResourceUser,
		Status:    model.ExportStatusCompleted,
		FilePath:  filePath,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}))

	w := app.Do(t, http.MethodDelete, "/api/v1/exports/exp-del-1", "", grantAuth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	_, statErr := os.Stat(filePath)
	assert.True(t, os.IsNotExist(statErr), "删除后导出文件应不在磁盘")
	_, err := repository.NewExportRepository(app.DB).GetByID(t.Context(), "exp-del-1")
	assert.ErrorIs(t, err, repository.ErrExportJobNotFound, "删除后记录不应存在")
}

// TestInteg_Export_DeleteFileInUseKeepsRow 文件删不掉时返回 409 且保留记录，
// 保证调用方可重试，不会留下无记录引用的孤儿文件。
func TestInteg_Export_DeleteFileInUseKeepsRow(t *testing.T) {
	app := testkit.NewTestApp(t)
	grantPermissions(t, app.DB, "u-export", "t-1", "export:list", "export:create", "export:delete")
	grantAuth := app.AuthHeader("u-export", "t-1", []string{"member"})

	// 非空目录作为 FilePath：os.Remove 在 Windows/Linux 均失败且不是 ErrNotExist
	blocked := filepath.Join(t.TempDir(), "busy.csv")
	require.NoError(t, os.Mkdir(blocked, 0o755))
	inner := filepath.Join(blocked, "hold")
	require.NoError(t, os.WriteFile(inner, []byte("x"), 0o640))
	repo := repository.NewExportRepository(app.DB)
	require.NoError(t, repo.Create(t.Context(), &model.ExportJob{
		ID: "exp-busy-1", TenantID: "t-1", UserID: "u-export",
		Resource:  model.ResourceUser,
		Status:    model.ExportStatusCompleted,
		FilePath:  blocked,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}))

	w := app.Do(t, http.MethodDelete, "/api/v1/exports/exp-busy-1", "", grantAuth)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	_, err := repo.GetByID(t.Context(), "exp-busy-1")
	assert.NoError(t, err, "文件未删成功时必须保留记录供重试")
	_, statErr := os.Stat(blocked)
	assert.NoError(t, statErr, "占位目录不应被误删")
}
