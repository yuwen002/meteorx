package integration

import (
	"net/http"
	"testing"

	"meteorx/internal/middleware"
	auditmodel "meteorx/internal/modules/audit/model"
	"meteorx/internal/testkit"

	"github.com/stretchr/testify/require"
)

// TestInteg_Audit_PublicLoginFailureRecorded 公开的登录入口不在审计中间件覆盖范围内，
// 只能靠 handler 手动记录。本测试走完整 router 打一次「不存在的用户」登录，强制 flush 后
// 断言真实落库一条 auth/login/failure 高风险记录、且请求体未泄露密码——
// 证明公开凭据操作的审计覆盖端到端贯通（而非仅在 handler 单测的捕获钩子里）。
func TestInteg_Audit_PublicLoginFailureRecorded(t *testing.T) {
	app := testkit.NewTestApp(t)

	// ghost-user 不存在，登录必然失败，命中 handler 的失败手动审计分支
	app.Do(t, http.MethodPost, "/api/v1/auth/login",
		`{"tenant_id":"t-1","username":"ghost-user","password":"Sup3rSecret!"}`, "")

	// 公开登录不经过自动审计中间件，唯一来源是 handler 手动记录；
	// 停批处理器触发同步 flush，使缓冲中的记录立即落库。
	middleware.StopAuditBatchProcessor()

	var rec auditmodel.AuditLog
	require.NoError(t, app.DB.
		Where("module = ? AND action = ? AND result = ?", "auth", "login", "failure").
		First(&rec).Error, "公开的登录失败操作应被手动审计并真实落库")

	require.Equal(t, "/api/v1/auth/login", rec.Path)
	require.Equal(t, auditmodel.RiskHigh, rec.RiskLevel)
	// 手动审计不采集请求体，明文密码绝不入库
	require.NotContains(t, rec.RequestBody, "Sup3rSecret!")
}
