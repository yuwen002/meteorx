package integration

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

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

// TestInteg_Audit_AlertChainFiresOnProductionWiring 端到端锁住生产接线：
// 只经真实路由（bootstrap.InitRouter → audit.WireAlertEvaluation）与审计中间件，
// 不用任何测试自建装配——一次 critical 请求（DELETE → 风险评级 critical）应自动
// 命中规则、POST 到 webhook 目标并把告警以 notified 落库。接线行被删时 hook 永不
// 命中、告警表永远为空（audit 包 wire_test 锁的是函数本身，盖不住 router 里的调用行）。
func TestInteg_Audit_AlertChainFiresOnProductionWiring(t *testing.T) {
	var hookHits int32
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hookHits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	app := testkit.NewTestApp(t)
	sa := app.SuperadminHeader("sa", "t-1")

	body := `{"name":"严重风险告警","enabled":true,"trigger_type":"risk_level","trigger_value":"critical",` +
		`"notify_channels":["webhook"],"notify_targets":["` + hook.URL + `"],"cooldown_minutes":30}`
	w := app.Do(t, http.MethodPost, "/api/v1/audit/alert-rules", body, sa)
	require.Equal(t, http.StatusOK, w.Code)
	ruleID := decodeID(t, w)

	// DELETE 被中间件风险评级判为 critical；目标用户不存在无关紧要——
	// 告警评估的是审计事件本身，失败操作同样应触发。
	app.Do(t, http.MethodDelete, "/api/v1/users/nonexistent", "", sa)

	// 审计中间件异步缓冲、批量处理器每 5s flush，轮询直等真实链路跑通
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&hookHits) >= 1
	}, 15*time.Second, 200*time.Millisecond, "生产接线下一条告警都未送达")

	// 告警按 rule_id 过滤只断言自建规则产出的那条：bootstrap 会预置默认告警规则（仓储层
	// seed），同一条 critical 日志会同时命中多条规则，其余规则因渠道不可达 notified=false，
	// 不限定 rule_id 时 items[0] 是同秒多行中哪条并不确定（旧写法间歇性误红）
	//
	// notified 回写与 hook 命中是两个动作（POST 在 MarkAlertNotified 之前发生），回写可能
	// 晚于计数一个调度窗口，故轮询直等 UPDATE 真实落库
	var out struct {
		Data struct {
			Items []struct {
				RiskLevel string `json:"risk_level"`
				Notified  bool   `json:"notified"`
			} `json:"items"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.Eventually(t, func() bool {
		w := app.Do(t, http.MethodGet, "/api/v1/audit/alerts?page=1&page_size=10&rule_id="+ruleID, "", sa)
		if w.Code != http.StatusOK {
			return false
		}
		testkit.DecodeData(t, w, &out.Data)
		return out.Data.Total == 1 && len(out.Data.Items) == 1 &&
			out.Data.Items[0].RiskLevel == "critical" && out.Data.Items[0].Notified
	}, 10*time.Second, 200*time.Millisecond, "送达后 notified 必须经 UPDATE 真实落库")
}
