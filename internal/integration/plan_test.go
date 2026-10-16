package integration

import (
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_Plan_CRUDAndAssign 整体验证套餐模块：
// 后台套餐列表（已播种）、创建套餐、将套餐分配给租户（→ 生效订阅），以及非超管越权拦截。
func TestInteg_Plan_CRUDAndAssign(t *testing.T) {
	app := testkit.NewTestApp(t)
	seedTenantAdminRole(t, app.DB)
	sa := app.SuperadminHeader("sa", "t-1")
	member := app.AuthHeader("m", "t-1", []string{"member"})

	// 列表（initPlans 已播种默认套餐）
	w := app.Do(t, http.MethodGet, "/api/v1/admin/plans", "", sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 创建套餐
	createBody := `{"name":"专业版","code":"pro-integ","user_limit":100,` +
		`"storage_limit":1073741824,"api_rate_limit":60,"wiki_space_limit":50,"price":99,"status":1}`
	w = app.Do(t, http.MethodPost, "/api/v1/admin/plans", createBody, sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	planID := decodeID(t, w)

	// 非超管创建套餐 → 403
	w = app.Do(t, http.MethodPost, "/api/v1/admin/plans", createBody, member)
	assert.Equal(t, http.StatusForbidden, w.Code, "非超管创建套餐应 403")

	// 为租户分配套餐（端到端：套餐 → 生效订阅）
	tenantID := ensureTenant(t, app, "套餐租户", "plan-tenant", sa)
	w = app.Do(t, http.MethodPut, "/api/v1/admin/tenants/"+tenantID+"/plan",
		`{"plan_id":"`+planID+`"}`, sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 租户详情应反映已分配套餐（plan_name 为套餐名称）
	w = app.Do(t, http.MethodGet, "/api/v1/admin/tenants/"+tenantID+"/detail", "", sa)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "专业版")
}
