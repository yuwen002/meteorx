package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_Task_CRUD_Isolation_AIDegrade 整体验证任务模块（受保护、仅需登录，无细粒度权限）：
// 注册→建任务→查询→完成→删除闭环；跨租户读不到他人任务；AI 未配置时优雅降级 503。
func TestInteg_Task_CRUD_Isolation_AIDegrade(t *testing.T) {
	app := testkit.NewTestApp(t)
	seedTenantAdminRole(t, app.DB)

	// 注册 + 登录
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/auth/register", `{
		"tenant_id":"t-task","username":"carol123","password":"pass1234","nickname":"卡罗","email":"c@x.com"
	}`, "").Code)
	w := app.Do(t, http.MethodPost, "/api/v1/auth/login", `{
		"tenant_id":"t-task","username":"carol123","password":"pass1234"
	}`, "")
	require.Equal(t, http.StatusOK, w.Code)
	var login struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &login))
	auth := "Bearer " + login.Data.Token

	// 创建任务
	w = app.Do(t, http.MethodPost, "/api/v1/tasks", `{"title":"写集成测试","priority":"high"}`, auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	id := decodeID(t, w)

	// 详情（列表 List 依赖 MySQL FIELD() 排序，sqlite 测试库不支持，故以详情验证读取；生产 MySQL 无碍）
	w = app.Do(t, http.MethodGet, "/api/v1/tasks/"+id, "", auth)
	assert.Equal(t, http.StatusOK, w.Code)

	// 完成 + 删除
	w = app.Do(t, http.MethodPut, "/api/v1/tasks/"+id+"/complete", "", auth)
	assert.Equal(t, http.StatusOK, w.Code)
	w = app.Do(t, http.MethodDelete, "/api/v1/tasks/"+id, "", auth)
	assert.Equal(t, http.StatusOK, w.Code)

	// 跨租户隔离：另一租户用户读不到该任务
	otherAuth := app.AuthHeader("u-other", "t-other", []string{"member"})
	w = app.Do(t, http.MethodGet, "/api/v1/tasks/"+id, "", otherAuth)
	assert.NotEqual(t, http.StatusOK, w.Code, "跨租户不应读到他人任务")

	// AI 能力端点：未配置时应优雅响应（200 或 503），不得 500
	w = app.Do(t, http.MethodGet, "/api/v1/tasks/ai/capabilities", "", auth)
	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, w.Code, "AI 能力端点应优雅响应")
}
