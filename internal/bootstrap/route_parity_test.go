package bootstrap_test

import (
	"net/http"
	"sort"
	"strings"
	"testing"

	"meteorx/internal/config"
	"meteorx/internal/routetable"
	"meteorx/internal/testkit"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// TestRouteTable_MatchesProductionRouter 是"手写镜像 vs 真实模块"的漂移护栏：
// 用 sqlite + miniredis 启动与生产完全一致的 bootstrap.InitRouter，
// 逐条比对 internal/routetable（供文档同步脚本与路由测试共用）的 API 路由集合。
//
// 过去 task 模块新增 /tasks/ai/*、batch/status、batch/assign 时镜像没跟上，
// 文档检查因此长期"假绿"；本测试会让这类漂移在单测阶段直接失败。
func TestRouteTable_MatchesProductionRouter(t *testing.T) {
	app := testkit.NewTestAppOpt(t, func(c *config.Config) {
		// 生产按 cfg.WS.Enabled 决定是否挂 /api/v1/ws，镜像无条件包含，这里对齐到开启态
		c.WS.Enabled = true
	})

	realRoutes := collectAPIRoutes(t, app.Router)
	mirrorRoutes := collectAPIRoutes(t, routetable.Build())

	var missingInMirror []string
	for key := range realRoutes {
		if !mirrorRoutes[key] {
			missingInMirror = append(missingInMirror, key)
		}
	}
	var extraInMirror []string
	for key := range mirrorRoutes {
		if !realRoutes[key] {
			extraInMirror = append(extraInMirror, key)
		}
	}
	sort.Strings(missingInMirror)
	sort.Strings(extraInMirror)

	require.Empty(t, missingInMirror,
		"internal/routetable 漏登记 %d 条生产路由：\n  %s",
		len(missingInMirror), strings.Join(missingInMirror, "\n  "))
	require.Empty(t, extraInMirror,
		"internal/routetable 多出 %d 条生产不存在的路由：\n  %s",
		len(extraInMirror), strings.Join(extraInMirror, "\n  "))
}

// collectAPIRoutes 提取参与文档契约的路由集合（根级健康检查/指标 + /api/v1/*）。
// 静态资源（/uploads/*）、NotFound 等不进入文档范围故排除；`*`（chi.Handle 通配）按 GET 归一。
func collectAPIRoutes(t *testing.T, r http.Handler) map[string]bool {
	t.Helper()
	routes := make(map[string]bool)
	require.NoError(t, chi.Walk(r.(chi.Routes), func(method, pattern string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if method == "" {
			return nil
		}
		pattern = normalizeRoutePath(pattern)
		if !isDocumentedPath(pattern) {
			return nil
		}
		if method == "*" || pattern == "/metrics" {
			// r.Handle 注册的端点（/metrics）会展开成全部 HTTP 方法，统一按 GET 对齐文档写法
			method = http.MethodGet
		}
		routes[strings.ToUpper(method)+" "+pattern] = true
		return nil
	}))
	return routes
}

// normalizeRoutePath 把 chi 的 :param 形式统一成 {param}，与文档侧写法一致。
func normalizeRoutePath(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if strings.HasPrefix(s, ":") {
			segs[i] = "{" + strings.TrimPrefix(s, ":") + "}"
		}
	}
	p = strings.Join(segs, "/")
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = p[:len(p)-1]
	}
	return p
}

// isDocumentedPath 判断路由是否属于文档契约范围。
func isDocumentedPath(p string) bool {
	switch {
	case strings.HasPrefix(p, "/api/v1/"), p == "/health", p == "/health/ready", p == "/metrics":
		return true
	default:
		return false
	}
}
