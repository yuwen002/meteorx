package bootstrap

import (
	"log"
	"meteorx/internal/config"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// mustBindEnv 绑定环境变量，失败时记录 fatal 错误
func mustBindEnv(v *viper.Viper, input ...string) {
	if len(input) != 2 {
		log.Fatalf("mustBindEnv requires exactly 2 arguments, got %d", len(input))
		return
	}
	if err := v.BindEnv(input[0], input[1]); err != nil {
		log.Fatalf("failed to bind env var %s to %s: %v", input[1], input[0], err)
	}
}

func LoadConfig() *config.Config {
	// 1. 首先尝试加载 .env 文件
	envErr := godotenv.Load()

	// 2. 设置环境变量绑定
	viper.SetEnvPrefix("METEORX")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 统一 viper 实例
	v := viper.GetViper()

	// 手动绑定平铺的环境变量到嵌套结构
	mustBindEnv(v, "server.port", "METEORX_APP_PORT")
	mustBindEnv(v, "server.mode", "METEORX_APP_MODE")
	mustBindEnv(v, "database.host", "METEORX_DB_HOST")
	mustBindEnv(v, "database.port", "METEORX_DB_PORT")
	mustBindEnv(v, "database.user", "METEORX_DB_USER")
	mustBindEnv(v, "database.password", "METEORX_DB_PASSWORD")
	mustBindEnv(v, "database.name", "METEORX_DB_NAME")
	mustBindEnv(v, "database.tls", "METEORX_DB_TLS")
	mustBindEnv(v, "database.debug", "METEORX_DB_DEBUG")
	mustBindEnv(v, "redis.host", "METEORX_REDIS_HOST")
	mustBindEnv(v, "redis.port", "METEORX_REDIS_PORT")
	mustBindEnv(v, "redis.password", "METEORX_REDIS_PASSWORD")
	mustBindEnv(v, "redis.db", "METEORX_REDIS_DB")
	mustBindEnv(v, "jwt.secret", "METEORX_JWT_SECRET")
	mustBindEnv(v, "jwt.expiration", "METEORX_JWT_EXPIRATION")
	mustBindEnv(v, "jwt.issuer", "METEORX_JWT_ISSUER")

	// 3. 检查是否需要读取 YAML 文件
	// 如果 .env 文件加载成功，则跳过 YAML 读取
	if envErr == nil {
		log.Println("Using configuration from .env file")
	} else {
		log.Println("No .env file found, loading from YAML")
		// 3. 设置并读取 YAML 配置文件（仅在没有 .env 时使用）
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./internal/config") // 你的 yaml 存放地
		viper.AddConfigPath(".")

		if err := viper.ReadInConfig(); err != nil {
			log.Fatalf("Failed to read config.yaml: %v", err)
		}
	}

	conf := &config.Config{}
	if err := viper.Unmarshal(conf); err != nil {
		log.Fatalf("Unable to decode config: %v", err)
	}

	return conf
}
