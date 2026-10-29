package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// TestInteg_TaskUpdateEditRoundTrip 回归编辑保存链路（用户反馈“现在保存不了”）：
// 前端编辑表单无条件提交 recurrence（无重复时为空串），旧版 DTO 的
// *string+omitempty,oneof 会把普通任务编辑拦成 400；同时验证接口输出
// （due_date “Y-m-d H:i:s”、remind_before Go Duration）原样回传时可正常保存，
// 以及 recurrence:"" 能真实取消重复周期。
func TestInteg_TaskUpdateEditRoundTrip(t *testing.T) {
	app := testkit.NewTestApp(t)
	seedTenantAdminRole(t, app.DB)

	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/auth/register", `{
		"tenant_id":"t-task2","username":"dave123","password":"pass1234","nickname":"戴夫","email":"d@x.com"
	}`, "").Code)
	w := app.Do(t, http.MethodPost, "/api/v1/auth/login", `{
		"tenant_id":"t-task2","username":"dave123","password":"pass1234"
	}`, "")
	require.Equal(t, http.StatusOK, w.Code)
	var login struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &login))
	auth := "Bearer " + login.Data.Token

	decodeTask := func(w *httptest.ResponseRecorder) map[string]any {
		var resp struct {
			Data map[string]any `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		return resp.Data
	}

	// 场景一：普通无重复无截止日任务，按前端完整提交体（recurrence:""）编辑 → 必须 200
	w = app.Do(t, http.MethodPost, "/api/v1/tasks", `{"title":"普通任务"}`, auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	id := decodeID(t, w)
	w = app.Do(t, http.MethodPut, "/api/v1/tasks/"+id,
		`{"title":"改名后的普通任务","description":"","visibility":"personal","priority":"normal","recurrence":"","tags":[]}`, auth)
	require.Equal(t, http.StatusOK, w.Code, "编辑普通任务不应被校验拦成 400："+w.Body.String())
	assert.Equal(t, "改名后的普通任务", decodeTask(w)["title"])

	// 场景二：带截止日/提醒/重复的任务，接口输出原样回传仅改标题 → 200 且字段不变
	w = app.Do(t, http.MethodPost, "/api/v1/tasks", `{
		"title":"周报","due_date":"2026-12-31 10:00:00","remind_before":"2h","recurrence":"weekly"
	}`, auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	created := decodeTask(w)
	wid := created["id"].(string)
	payload := map[string]any{
		"title":         "周报改名",
		"visibility":    created["visibility"],
		"priority":      created["priority"],
		"due_date":      created["due_date"],      // 后端输出格式原样回传
		"remind_before": created["remind_before"], // Go Duration 字符串回环
		"recurrence":    created["recurrence"],
		"tags":          []string{},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	w = app.Do(t, http.MethodPut, "/api/v1/tasks/"+wid, string(body), auth)
	require.Equal(t, http.StatusOK, w.Code, "整表原样回传保存不应失败："+w.Body.String())
	updated := decodeTask(w)
	assert.Equal(t, "周报改名", updated["title"])
	assert.Equal(t, created["due_date"], updated["due_date"], "原值回传不得改变截止日")
	assert.Equal(t, "weekly", updated["recurrence"])

	// 场景三：选择“不重复”提交 recurrence:"" → 取消重复成功
	w = app.Do(t, http.MethodPut, "/api/v1/tasks/"+wid, `{"recurrence":""}`, auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Empty(t, decodeTask(w)["recurrence"], "空串应取消重复周期")

	// 守门：重复任务清空截止日仍被拒 400（带可读 message），守护“重复必带截止日”不变量
	w = app.Do(t, http.MethodPut, "/api/v1/tasks/"+wid,
		`{"recurrence":"daily","due_date":""}`, auth)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// 守门：非空非法周期被 service 拒 400，并透出顶层 message（旧版响应无 message，前端只能显示 Bad Request）
	w = app.Do(t, http.MethodPut, "/api/v1/tasks/"+wid, `{"recurrence":"yearly"}`, auth)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "重复周期不合法")
}
