package bootstrap

import (
	"testing"

	"meteorx/internal/config"
)

// TestIsReleaseMode 验证生产模式判定的口径：仅 release（大小写不敏感、忽略首尾空白）为真，
// 其余（debug/dev/test 及空值）均为假。该判定决定 Redis 连接失败是否阻断启动。
func TestIsReleaseMode(t *testing.T) {
	cases := []struct {
		mode string
		want bool
	}{
		{"release", true},
		{"Release", true},
		{"  release  ", true},
		{"debug", false},
		{"test", false},
		{"production", false}, // 本项目生产标识统一用 release，不使用 production
		{"", false},
	}
	for _, c := range cases {
		if got := isReleaseMode(c.mode); got != c.want {
			t.Errorf("isReleaseMode(%q) = %v, want %v", c.mode, got, c.want)
		}
	}
}

// TestValidateJWTSecret 验证弱密钥校验按模式分级：非 release 仅告警放行，release 拒绝启动。
func TestValidateJWTSecret(t *testing.T) {
	weak := &config.Config{
		Server: config.ServerConfig{Mode: "release"},
		JWT:    config.JWTConfig{Secret: "change-this-to-a-strong-random-secret-key"},
	}
	if err := validateJWTSecret(weak); err == nil {
		t.Error("release mode with weak JWT secret should be rejected")
	}

	strong := &config.Config{
		Server: config.ServerConfig{Mode: "release"},
		JWT:    config.JWTConfig{Secret: "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="},
	}
	if err := validateJWTSecret(strong); err != nil {
		t.Errorf("release mode with strong JWT secret should pass, got %v", err)
	}

	// 非 release 模式即便弱密钥也只告警，不阻断本地调试
	debugWeak := &config.Config{
		Server: config.ServerConfig{Mode: "debug"},
		JWT:    config.JWTConfig{Secret: "123456"},
	}
	if err := validateJWTSecret(debugWeak); err != nil {
		t.Errorf("debug mode should not reject weak JWT secret, got %v", err)
	}
}
