package bootstrap

import (
	"fmt"
	"log"
	"meteorx/internal/config"
	"strings"

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
func LoadConfig() (*config.Config, error) {
	// 1. 首先尝试加载 .env 文件
	envErr := godotenv.Load()

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
	}
	for _, b := range envBindings {
		if err := mustBindEnv(v, b.key, b.env); err != nil {
			return nil, fmt.Errorf("failed to bind env var %s to %s: %w", b.env, b.key, err)
		}
	}

	// 3. 检查是否需要读取 YAML 文件
	if envErr == nil {
		log.Println("Using configuration from .env file")
	} else {
		log.Println("No .env file found, loading from YAML")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./internal/config")
		viper.AddConfigPath(".")

		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config.yaml: %w", err)
		}
	}

	conf := &config.Config{}
	if err := viper.Unmarshal(conf); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return conf, nil
}