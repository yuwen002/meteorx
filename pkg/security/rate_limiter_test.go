package security

import (
	"context"
	"net/http"
	"testing"
	"time"

	"meteorx/internal/config"

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

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name:       "X-Forwarded-For",
			remoteAddr: "192.168.1.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1"},
			expected:   "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "192.168.1.1:1234",
			headers:    map[string]string{"X-Real-IP": "10.0.0.2"},
			expected:   "10.0.0.2",
		},
		{
			name:       "RemoteAddr",
			remoteAddr: "192.168.1.1:1234",
			headers:    map[string]string{},
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			ip := GetClientIP(req)
			assert.Equal(t, tt.expected, ip)
		})
	}
}

func BenchmarkGetClientIP(b *testing.B) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetClientIP(req)
	}
}