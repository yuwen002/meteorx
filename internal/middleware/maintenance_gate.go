// Package middleware 维护模式门禁中间件
// 消费平台运行时配置中心的 maintenance_mode 开关：维护期间除平台超管外，
// 所有已登录用户的业务 API 请求返回 503 与可配置提示文案，实现全站软停机。
package middleware

import (
	"context"
	"net/http"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
)

// MaintenanceStateProvider 维护模式状态查询能力（由 settings 配置服务提供）。
// 实现方应在内部故障时回退「未开启」，避免基础设施故障把全站误锁死。
type MaintenanceStateProvider interface {
	MaintenanceState(ctx context.Context) (enabled bool, message string)
}

// MaintenanceGate 维护模式门禁：挂载在认证之后，依据运行时配置拦截业务请求。
//   - 未开启维护：直接放行（判定走 settings 服务的 Redis 整表缓存，非每请求查库）
//   - 维护中且为平台超管：放行，保证管理员仍能登录后台关闭维护（闭环）
//   - 维护中其他用户：503 + 配置的提示文案
//
// 公开分组路由（登录/注册/公开配置）不在本门禁作用域内，前端得以展示维护公告。
func MaintenanceGate(provider MaintenanceStateProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			enabled, message := provider.MaintenanceState(r.Context())
			if !enabled {
				next.ServeHTTP(w, r)
				return
			}
			// 超管豁免：维护期间保留平台后台可用性
			if contextx.HasRole(r.Context(), "superadmin") {
				next.ServeHTTP(w, r)
				return
			}
			if message == "" {
				message = "系统维护中，请稍后再试"
			}
			response.FailWithStatus(w, http.StatusServiceUnavailable, message)
		})
	}
}
