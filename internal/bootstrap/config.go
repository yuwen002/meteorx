package bootstrap

import (
	"fmt"
	"strings"

	"meteorx/internal/config"
	"meteorx/pkg/logger"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// mustBindEnv 绑定环境变量
// 入参必须是 (viperKey, envVarName)；绑定失败返回 error 由调用方决定如何处理
func mustBindEnv(v *viper.Viper, input ...string) error {
	if len(input) != 2 {
		return fmt.Errorf("mustBindEnv requires exactly 2 arguments, got %d", len(input))
	}
	return v.BindEnv(input[0], input[1])
}

// LoadConfig 加载配置；出错时返回 error 供调用方决定是否继续
// 语义：以 config.yaml 为默认值基底，.env 中的环境变量覆盖对应项。
func LoadConfig() (*config.Config, error) {
	// 1. 首先尝试加载 .env 文件（存在则导入环境变量，缺失不报错）
	_ = godotenv.Load()

	// 2. 设置环境变量绑定
	viper.SetEnvPrefix("METEORX")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 统一 viper 实例
	v := viper.GetViper()

	// 手动绑定平铺的环境变量到嵌套结构
	envBindings := []struct{ key, env string }{
		{"server.port", "METEORX_APP_PORT"},
		{"server.mode", "METEORX_APP_MODE"},
		{"database.host", "METEORX_DB_HOST"},
		{"database.port", "METEORX_DB_PORT"},
		{"database.user", "METEORX_DB_USER"},
		{"database.password", "METEORX_DB_PASSWORD"},
		{"database.name", "METEORX_DB_NAME"},
		{"database.tls", "METEORX_DB_TLS"},
		{"database.debug", "METEORX_DB_DEBUG"},
		{"redis.host", "METEORX_REDIS_HOST"},
		{"redis.port", "METEORX_REDIS_PORT"},
		{"redis.password", "METEORX_REDIS_PASSWORD"},
		{"redis.db", "METEORX_REDIS_DB"},
		{"jwt.secret", "METEORX_JWT_SECRET"},
		{"jwt.expiration", "METEORX_JWT_EXPIRATION"},
		{"jwt.issuer", "METEORX_JWT_ISSUER"},
		{"client.base_url", "METEORX_CLIENT_BASE_URL"},
	}
	for _, b := range envBindings {
		if err := mustBindEnv(v, b.key, b.env); err != nil {
			return nil, fmt.Errorf("failed to bind env var %s to %s: %w", b.env, b.key, err)
		}
	}

	// 3. 始终读取 config.yaml 作为默认值，.env 中的变量对其做覆盖
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./internal/config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config.yaml: %w", err)
	}

	conf := &config.Config{}
	if err := viper.Unmarshal(conf); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	// 关键安全校验：release 模式必须使用强随机 JWT 密钥，拒绝默认/占位值上线
	if err := validateJWTSecret(conf); err != nil {
		return nil, err
	}

	return conf, nil
}

// validateJWTSecret release 模式下拒绝弱 JWT 密钥，防止使用默认/占位密钥上线；
// 非 release（debug/dev/test）仅告警提示，便于本地调试
func validateJWTSecret(conf *config.Config) error {
	if !strings.EqualFold(conf.Server.Mode, "release") {
		if isWeakJWTSecret(conf.JWT.Secret) {
			logger.Warnf("JWT secret 为默认/弱值，仅可用于本地调试；生产请通过 METEORX_JWT_SECRET 配置强随机密钥（>=32字节）")
		}
		return nil
	}

	if isWeakJWTSecret(conf.JWT.Secret) {
		return fmt.Errorf("release 模式下 JWT secret 未配置或过弱，拒绝启动：请通过环境变量 METEORX_JWT_SECRET 设置强随机密钥（>=32字节，生成示例: openssl rand -base64 64）")
	}
	return nil
}

// isWeakJWTSecret 判断密钥是否为空、命中常见占位/弱值关键字或长度不足 32
func isWeakJWTSecret(secret string) bool {
	s := strings.TrimSpace(secret)
	if s == "" {
		return true
	}
	lower := strings.ToLower(s)
	weakKeywords := []string{
		"change-this", "secret-key", "your-secret", "changeme",
		"123456", "abcdef", "password", "jwtsecret", "default", "todo",
	}
	for _, w := range weakKeywords {
		if strings.Contains(lower, w) {
			return true
		}
	}
	return len(s) < 32
}
