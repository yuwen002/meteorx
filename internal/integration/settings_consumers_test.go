package integration

import (
	"fmt"
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/require"
)

// setSettingRaw 测试辅助：绕过服务层直改配置行并清掉整表缓存，
// 模拟"运维直接动库"路径，验证消费方读取的是最新状态而非脏缓存。
func setSettingRaw(t *testing.T, app *testkit.TestApp, key, value string) {
	t.Helper()
	require.NoError(t, app.DB.Exec(
		"UPDATE system_settings SET value = ? WHERE config_key = ?", value, key).Error)
	app.MR.Del("settings:all")
}

// TestInteg_SettingsRegistrationGate 端到端验证注册开关消费者：
// 关闭 registration_enabled 后公开注册返回 403，重新打开（走管理接口，验证写路径自动失效缓存）后立即恢复放行。
func TestInteg_SettingsRegistrationGate(t *testing.T) {
	app := testkit.NewTestApp(t)
	seedTenantAdminRole(t, app.DB)

	body := func(name string) string {
		return fmt.Sprintf(`{"tenant_id":"t-gate","username":%q,"password":"pass1234","nickname":"游测","email":"%s@example.com"}`, name, name)
	}

	// 默认放行：种子值 registration_enabled=true，注册成功
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/auth/register", body("gate_u1"), "").Code)

	// 租户自助开户同受开关约束：默认放行
	tenBody := func(domain string) string {
		return fmt.Sprintf(`{"name":"Gate 租户 %s","domain":%q,"contact_email":"%s@example.com","admin_user":{"username":"boss_%s","password":"pass1234","nickname":"开户管理员"}}`, domain, domain, domain, domain)
	}
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/tenants/register", tenBody("gate-t1"), "").Code)

	// 关闭注册（直改库+清缓存，且首次注册已让服务读过配置，确认无脏缓存放行）
	setSettingRaw(t, app, "registration_enabled", "false")
	w := app.Do(t, http.MethodPost, "/api/v1/auth/register", body("gate_u2"), "")
	require.Equal(t, http.StatusForbidden, w.Code, "开关关闭后注册应 403: %s", w.Body.String())

	// 自助开户同步 403（堵住绕过前端直接开户的缺口）
	w = app.Do(t, http.MethodPost, "/api/v1/tenants/register", tenBody("gate-t2"), "")
	require.Equal(t, http.StatusForbidden, w.Code, "开关关闭后自助开户应 403: %s", w.Body.String())

	// 超管通过管理接口重新打开（UpdateBulk 应自动失效缓存，无需手动清）
	sAdmin := app.SuperadminHeader("u-super", "")
	w = app.Do(t, http.MethodPut, "/api/v1/admin/settings",
		`{"settings":[{"key":"registration_enabled","value":"true"}]}`, sAdmin)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/auth/register", body("gate_u3"), "").Code,
		"管理接口改回后注册应立即恢复放行")
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/tenants/register", tenBody("gate-t3"), "").Code,
		"管理接口改回后自助开户应立即恢复放行")
}

// TestInteg_SettingsMaintenanceMode 端到端验证维护模式消费者：
// 开启后普通用户业务请求 503、平台超管豁免、公开配置接口不受限；关闭后立即恢复。
func TestInteg_SettingsMaintenanceMode(t *testing.T) {
	app := testkit.NewTestApp(t)
	seedTenantAdminRole(t, app.DB)

	// 注册并登录一个普通用户
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/auth/register",
		`{"tenant_id":"t-maint","username":"maint_u1","password":"pass1234","nickname":"维护测试","email":"m1@example.com"}`, "").Code)
	w := app.Do(t, http.MethodPost, "/api/v1/auth/login",
		`{"tenant_id":"t-maint","username":"maint_u1","password":"pass1234"}`, "")
	require.Equal(t, http.StatusOK, w.Code)
	userToken := "Bearer " + decodeToken(t, w)

	// 基线：未开启维护时业务接口可用
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, "/api/v1/profile", "", userToken).Code)

	// 超管开启维护模式并设置公告文案（走管理接口，验证完整写链路）
	sAdmin := app.SuperadminHeader("u-super", "")
	w = app.Do(t, http.MethodPut, "/api/v1/admin/settings",
		`{"settings":[{"key":"maintenance_mode","value":"true"},{"key":"maintenance_message","value":"今晚 22:00 升级"}]}`, sAdmin)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 普通用户业务请求被 503 拦截并携带公告文案
	w = app.Do(t, http.MethodGet, "/api/v1/profile", "", userToken)
	require.Equal(t, http.StatusServiceUnavailable, w.Code, "维护中普通用户应 503: %s", w.Body.String())
	require.Contains(t, w.Body.String(), "今晚 22:00 升级")

	// 超管豁免：仍可访问业务接口（保证能进后台关闭维护）
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, "/api/v1/admin/settings", "", sAdmin).Code,
		"超管应不受维护模式限制")

	// 公开配置接口在 Public 分组，维护期间仍可读取（前端展示公告依赖）
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, "/api/v1/settings/public", "", "").Code)

	// 关闭维护后普通用户立即恢复
	w = app.Do(t, http.MethodPut, "/api/v1/admin/settings",
		`{"settings":[{"key":"maintenance_mode","value":"false"}]}`, sAdmin)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, "/api/v1/profile", "", userToken).Code,
		"关闭维护后业务接口应恢复")
}

// TestInteg_SettingsAdminEndpoints 管理/公开接口权限与数据形状端到端验证：
// 超管可见全部登记键；非超管被特权组拦截；公开接口只暴露 is_public 键且类型化。
func TestInteg_SettingsAdminEndpoints(t *testing.T) {
	app := testkit.NewTestApp(t)
	seedTenantAdminRole(t, app.DB)

	// 非超管（租户管理员）访问管理配置：特权组直接 403
	tenantAdmin := app.AuthHeader("u-tadmin", "t-x", []string{"tenant_admin"})
	require.Equal(t, http.StatusForbidden,
		app.Do(t, http.MethodGet, "/api/v1/admin/settings", "", tenantAdmin).Code)

	// 超管列表：三个登记键均在，source 为 db（启动已播种）
	sAdmin := app.SuperadminHeader("u-super", "")
	w := app.Do(t, http.MethodGet, "/api/v1/admin/settings", "", sAdmin)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var list []struct {
		Key    string `json:"key"`
		Source string `json:"source"`
	}
	testkit.DecodeData(t, w, &list)
	require.Len(t, list, 3)
	for _, item := range list {
		require.Equal(t, "db", item.Source, "启动播种后列表值应来源于库: %s", item.Key)
	}

	// 公开接口：仅 3 个 public 键，布尔为真 JSON 类型
	pub := app.Do(t, http.MethodGet, "/api/v1/settings/public", "", "")
	require.Equal(t, http.StatusOK, pub.Code)
	var public map[string]any
	testkit.DecodeData(t, pub, &public)
	require.Equal(t, true, public["registration_enabled"])
	require.Equal(t, false, public["maintenance_mode"])
	require.Contains(t, public, "maintenance_message")
}
