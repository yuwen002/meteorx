package integration

import (
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_Audit_SuperadminAndAlertRule 整体验证审计模块（超管网关内）：
// 超管可看统计/日志、可建告警规则闭环；非超管一律 403。
func TestInteg_Audit_SuperadminAndAlertRule(t *testing.T) {
	app := testkit.NewTestApp(t)
	sa := app.SuperadminHeader("sa", "t-1")
	member := app.AuthHeader("m", "t-1", []string{"member"})

	// 统计/日志：超管 200，非超管 403
	assert.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, "/api/v1/audit/stats", "", sa).Code)
	assert.Equal(t, http.StatusForbidden, app.Do(t, http.MethodGet, "/api/v1/audit/stats", "", member).Code)
	assert.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, "/api/v1/audit/logs", "", sa).Code)
	assert.Equal(t, http.StatusForbidden, app.Do(t, http.MethodGet, "/api/v1/audit/logs", "", member).Code)

	// 告警规则 CRUD（超管）
	body := `{"name":"登录失败告警","enabled":true,"trigger_type":"login_failure","trigger_value":"5",` +
		`"notify_channels":["email"],"notify_targets":["a@x.com"],"cooldown_minutes":10}`
	w := app.Do(t, http.MethodPost, "/api/v1/audit/alert-rules", body, sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	id := decodeID(t, w)

	w = app.Do(t, http.MethodGet, "/api/v1/audit/alert-rules", "", sa)
	assert.Contains(t, w.Body.String(), id)
	w = app.Do(t, http.MethodGet, "/api/v1/audit/alert-rules/"+id, "", sa)
	assert.Equal(t, http.StatusOK, w.Code)
	w = app.Do(t, http.MethodDelete, "/api/v1/audit/alert-rules/"+id, "", sa)
	assert.Equal(t, http.StatusOK, w.Code)

	// 非超管创建告警规则 → 403
	w = app.Do(t, http.MethodPost, "/api/v1/audit/alert-rules", body, member)
	assert.Equal(t, http.StatusForbidden, w.Code, "非超管创建告警规则应 403")
}
