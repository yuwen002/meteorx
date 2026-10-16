package integration

import (
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_TaskAI_Degradation 整体验证任务 AI 辅助层（受保护 /tasks 分组，仅需登录）：
//   - capabilities 探测端点恒可用（200，返回能力描述）
//   - 未配置 AI 凭据时，parse/summarize 等生成端点优雅降级为 503，绝不影响主流程
//   - 未登录 → 401
func TestInteg_TaskAI_Degradation(t *testing.T) {
	app := testkit.NewTestApp(t)
	seedTenantAdminRole(t, app.DB)

	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/auth/register",
		`{"tenant_id":"t-ai","username":"aiuser123","password":"pass1234","nickname":"AI","email":"ai@x.com"}`, "").Code)
	w := app.Do(t, http.MethodPost, "/api/v1/auth/login",
		`{"tenant_id":"t-ai","username":"aiuser123","password":"pass1234"}`, "")
	require.Equal(t, http.StatusOK, w.Code)
	auth := "Bearer " + decodeToken(t, w)

	// 未登录访问 AI 端点 → 401
	assert.Equal(t, http.StatusUnauthorized, app.Do(t, http.MethodGet, "/api/v1/tasks/ai/capabilities", "", "").Code)

	// capabilities 探测端点：恒 200（testkit 未配置 AI，应回传不可用描述）
	w = app.Do(t, http.MethodGet, "/api/v1/tasks/ai/capabilities", "", auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 未配置 AI：生成类端点优雅降级为 503（不得 500/panic）
	assert.Equal(t, http.StatusServiceUnavailable,
		app.Do(t, http.MethodPost, "/api/v1/tasks/ai/parse", `{"text":"明天下午三点开会"}`, auth).Code,
		"AI 未配置，parse 应降级 503")
	assert.Equal(t, http.StatusServiceUnavailable,
		app.Do(t, http.MethodPost, "/api/v1/tasks/ai/summary", `{}`, auth).Code,
		"AI 未配置，summary 应降级 503")
}
