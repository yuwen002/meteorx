package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/pkg/security"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// TestSetupMiddleware_ClientIPFromXFF 端到端锁定 P1-2 装配点：
// SetupMiddleware 必须安装 chimiddleware.ClientIPFromXFF，把 XFF 解析结果写入
// 请求 context，令 security.GetClientIP 读到可信客户端 IP 而不是伪造最左条目。
// 若有人误改回已弃用的 RealIP 或漏挂中间件，本测试会立即失败。
func TestSetupMiddleware_ClientIPFromXFF(t *testing.T) {
	cases := []struct {
		name         string
		trusted      []string
		remoteAddr   string
		xff          string
		expectedIP   string
	}{
		{
			// 未配置可信代理：nginx 单跳部署下最右条目才是 nginx 追加的真实对端，
			// 攻击者塞入的最左伪造条目应被忽略。
			name:       "无 trusted 代理：单跳 nginx 部署下取最右条目",
			trusted:    nil,
			remoteAddr: "10.0.0.5:9876",
			xff:        "1.2.3.4, 203.0.113.7",
			expectedIP: "203.0.113.7",
		},
		{
			// 配置可信代理段：右起跳过可信地址，首个不可信条目即真实客户端。
			name:       "配置 trusted 代理：右起跳过可信段取首个不可信 IP",
			trusted:    []string{"198.51.100.0/24"},
			remoteAddr: "10.0.0.5:9876",
			xff:        "203.0.113.9, 198.51.100.23",
			expectedIP: "203.0.113.9",
		},
		{
			// 完全无 XFF：使用 RemoteAddr 兜底。
			name:       "无 XFF：回退 RemoteAddr",
			trusted:    []string{"127.0.0.1/32"},
			remoteAddr: "192.168.1.42:5555",
			xff:        "",
			expectedIP: "192.168.1.42",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewMux()
			SetupMiddleware(r, nil, tc.trusted)

			var got string
			r.Get("/", func(_ http.ResponseWriter, req *http.Request) {
				got = security.GetClientIP(req)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			r.ServeHTTP(httptest.NewRecorder(), req)

			assert.Equal(t, tc.expectedIP, got)
		})
	}
}
