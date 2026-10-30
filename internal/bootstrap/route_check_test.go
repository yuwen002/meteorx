package bootstrap

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"

	"meteorx/internal/middleware"
	"meteorx/internal/routetable"

	"github.com/go-chi/chi/v5"
)

func TestAllRoutes_DerivePermissionCode(t *testing.T) {
	// 路由来源统一为 internal/routetable（与生产 InitRouter 由对齐测试强校验），
	// 避免本测试再维护一份手写镜像导致覆盖范围悄悄缩水。
	r := routetable.Build()

	type routeEntry struct {
		method  string
		pattern string
	}
	var entries []routeEntry

	err := chi.Walk(r, func(method, pattern string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if method == "" {
			return nil
		}
		entries = append(entries, routeEntry{method: method, pattern: pattern})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].pattern == entries[j].pattern {
			return entries[i].method < entries[j].method
		}
		return entries[i].pattern < entries[j].pattern
	})

	apiCount := 0
	rootCount := 0
	var noPermRoutes []string
	var permResults []string

	for _, e := range entries {
		if strings.HasPrefix(e.pattern, "/api/v1/") {
			apiCount++
			code := middleware.DerivePermissionCode(e.method, e.pattern)
			if code == "" {
				noPermRoutes = append(noPermRoutes, fmt.Sprintf("%s %s", e.method, e.pattern))
			}
			permResults = append(permResults, fmt.Sprintf("%-6s %-55s → %s", e.method, e.pattern, code))
		} else {
			rootCount++
		}
	}

	fmt.Fprintf(os.Stderr, "\n===== 路由总数统计 =====\n")
	fmt.Fprintf(os.Stderr, "API 路由 (/api/v1/*): %d\n", apiCount)
	fmt.Fprintf(os.Stderr, "根路由: %d\n", rootCount)
	fmt.Fprintf(os.Stderr, "总计: %d\n\n", apiCount+rootCount)

	fmt.Fprintf(os.Stderr, "===== 权限码推导结果 =====\n")
	for _, line := range permResults {
		fmt.Fprintf(os.Stderr, "%s\n", line)
	}

	if len(noPermRoutes) > 0 {
		t.Errorf("有 %d 个 /api/v1 路由无法自动推导权限码：\n", len(noPermRoutes))
		for _, r := range noPermRoutes {
			t.Errorf("  %s", r)
		}
	}
}
