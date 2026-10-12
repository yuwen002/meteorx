package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"meteorx/internal/config"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// RateLimiterTestSuite 限流器测试套件
type RateLimiterTestSuite struct {
	suite.Suite
	limiter *RateLimiter
	ctx     context.Context
}

func (s *RateLimiterTestSuite) SetupTest() {
	cfg := config.RateLimitConfig{
		Enabled:   true,
		Requests:  5,
		Window:    1 * time.Minute,
		BurstSize: 2,
	}
	s.limiter = NewRateLimiter(nil, cfg)
	s.ctx = context.Background()
}

func TestRateLimiterSuite(t *testing.T) {
	suite.Run(t, new(RateLimiterTestSuite))
}

func (s *RateLimiterTestSuite) TestAllowDisabled() {
	cfg := config.RateLimitConfig{Enabled: false}
	limiter := NewRateLimiter(nil, cfg)

	allowed, err := limiter.Allow(s.ctx, "client1")
	s.NoError(err)
	s.True(allowed)
}

func (s *RateLimiterTestSuite) TestAllowWithoutRedis() {
	// 无 Redis 时，Allow 应返回 true（降级）
	allowed, err := s.limiter.Allow(s.ctx, "client1")
	s.NoError(err)
	s.True(allowed)
}

func (s *RateLimiterTestSuite) TestGetKey() {
	key := s.limiter.getKey("client1")
	s.Contains(key, "rate_limit:client1:")
}

func TestRateLimitConfig(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:   true,
		Requests:  100,
		Window:    1 * time.Minute,
		BurstSize: 10,
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 100, cfg.Requests)
	assert.Equal(t, 1*time.Minute, cfg.Window)
	assert.Equal(t, 10, cfg.BurstSize)
}

func TestNewRateLimiter(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:  true,
		Requests: 10,
		Window:   5 * time.Second,
	}

	limiter := NewRateLimiter(nil, cfg)
	assert.NotNil(t, limiter)
	assert.Equal(t, cfg, limiter.config)
}

// resolveClientIP 将带指定 RemoteAddr/XFF 的请求先经 mw（可为 nil）
// 再在末端 handler 内调用 GetClientIP，返回解析结果。
func resolveClientIP(mw func(http.Handler) http.Handler, remoteAddr, xff string) string {
	got := ""
	var h http.Handler = http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = GetClientIP(r)
	})
	if mw != nil {
		h = mw(h)
	}
	req, _ := http.NewRequest("GET", "/", nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	h.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		mw       func(http.Handler) http.Handler
		remote   string
		xff      string
		expected string
	}{
		{
			name:     "直连：无中间件时取 RemoteAddr",
			mw:       nil,
			remote:   "192.168.1.1:1234",
			expected: "192.168.1.1",
		},
		{
			name:     "未经可信代理解析时忽略伪造 XFF 头",
			mw:       nil,
			remote:   "192.168.1.1:1234",
			xff:      "10.0.0.1",
			expected: "192.168.1.1",
		},
		{
			name:     "单跳 nginx：左侧伪造条目被忽略，取最右 nginx 追加的对端",
			mw:       chimiddleware.ClientIPFromXFF(),
			remote:   "10.0.0.5:9876",
			xff:      "1.2.3.4, 203.0.113.7",
			expected: "203.0.113.7",
		},
		{
			name:     "多跳可信段：右起跳过可信地址取首个不可信项",
			mw:       chimiddleware.ClientIPFromXFF("198.51.100.0/24"),
			remote:   "10.0.0.5:9876",
			xff:      "203.0.113.9, 198.51.100.23",
			expected: "203.0.113.9",
		},
		{
			name:     "垃圾 XFF fail-closed：回退 RemoteAddr",
			mw:       chimiddleware.ClientIPFromXFF(),
			remote:   "192.168.1.9:5555",
			xff:      "not-an-ip",
			expected: "192.168.1.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, resolveClientIP(tt.mw, tt.remote, tt.xff))
		})
	}
}

func BenchmarkGetClientIP(b *testing.B) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:1234"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetClientIP(req)
	}
}