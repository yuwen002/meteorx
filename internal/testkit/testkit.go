// Package testkit 提供“整体项”集成测试的复用基座：
// 用真实 miniredis + sqlite + 完整 config 启动 bootstrap.InitRouter，
// 挂载全部业务模块的真实路由，从而能在单一测试里跨模块驱动端到端 HTTP 请求。
//
// 仅被 *_test.go 引用，不会进入生产二进制。
package testkit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"meteorx/internal/bootstrap"
	"meteorx/internal/cache"
	"meteorx/internal/common/jwt"
	"meteorx/internal/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestApp 封装完整路由 + 依赖，供整体集成测试复用。
type TestApp struct {
	Router      *chi.Mux
	DB          *gorm.DB
	TokenHelper *jwt.TokenHelper
	Cfg         *config.Config

	// MR 是内嵌的 miniredis 实例，供测试断言 Redis 键状态（如登出黑名单）。
	MR   *miniredis.Miniredis
	stop context.CancelFunc
}

// NewTestApp 构造一个挂载全部模块真实路由的测试应用。
func NewTestApp(t *testing.T) *TestApp {
	t.Helper()
	return NewTestAppOpt(t)
}

// NewTestAppOpt 同 NewTestApp，但允许在 InitRouter 前对 cfg 做定向调整
// （如开启审计完整性 pepper），供需要验证特定配置下生产接线的集成测试复用。
func NewTestAppOpt(t *testing.T, tune ...func(*config.Config)) *TestApp {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb, err := cache.NewRedis(mr.Addr(), "", 0)
	require.NoError(t, err)

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, bootstrap.AutoMigrate(db), "全量迁移失败")

	cfg := &config.Config{
		Server: config.ServerConfig{Mode: "test"},
		JWT: config.JWTConfig{
			Secret:     "test-jwt-secret-which-is-long-enough-32",
			Expiration: "1h",
			Issuer:     "meteorx-test",
		},
		OAuth: config.OAuthConfig{
			SecretKey: "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=", // base64(32 bytes)，用于 client_secret 加密
		},
		Client: config.ClientConfig{BaseURL: "http://localhost:3000"},
		Security: config.SecurityConfig{
			PasswordPolicy: config.PasswordPolicyConfig{MinLength: 6},
			RateLimit:      config.RateLimitConfig{Enabled: false},
			LoginLockout:   config.LoginLockoutConfig{Enabled: false},
		},
		File: config.FileConfig{UploadPath: t.TempDir()},
	}
	for _, f := range tune {
		f(cfg)
	}

	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)

	router := bootstrap.InitRouter(ctx, db, cfg, rdb)

	return &TestApp{
		Router:      router,
		DB:          db,
		TokenHelper: jwt.NewTokenHelper(cfg.JWT),
		Cfg:         cfg,
		MR:          mr,
		stop:        stop,
	}
}

// AuthHeader 为指定身份签发 JWT 并返回 "Bearer <token>"。
func (a *TestApp) AuthHeader(userID, tenantID string, roles []string) string {
	tok, err := a.TokenHelper.GenerateToken(userID, tenantID, roles)
	if err != nil {
		panic(err)
	}
	return "Bearer " + tok
}

// SuperadminHeader 签发平台超级管理员令牌。
func (a *TestApp) SuperadminHeader(userID, tenantID string) string {
	return a.AuthHeader(userID, tenantID, []string{jwt.RoleSuperadmin})
}

// Do 以给定方法/路径/JSON 体和可选鉴权头驱动一次真实 HTTP 请求。
func (a *TestApp) Do(t *testing.T, method, path, body, auth string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	return w
}

// DecodeData 将响应体中的 data 字段反序列化到 out（便于断言业务数据）。
func DecodeData(t *testing.T, w *httptest.ResponseRecorder, out any) {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env), "响应非标准结构: %s", w.Body.String())
	require.NoError(t, json.Unmarshal(env.Data, out), "data 解析失败: %s", env.Data)
}
