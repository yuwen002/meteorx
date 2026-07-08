package config

import "time"

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
	Security SecurityConfig `mapstructure:"security"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	TLS      bool   `mapstructure:"tls"`
	Debug    bool   `mapstructure:"debug"` // 开启后输出 SQL 日志
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	Expiration string `mapstructure:"expiration"`
	Issuer     string `mapstructure:"issuer"`
}

// GetExpiration 解析字符串为 time.Duration
func (j JWTConfig) GetExpiration() time.Duration {
	d, err := time.ParseDuration(j.Expiration)
	if err != nil {
		return 24 * time.Hour // 默认 24 小时
	}
	return d
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type SecurityConfig struct {
	LoginLockout     LoginLockoutConfig     `mapstructure:"login_lockout"`
	PasswordPolicy   PasswordPolicyConfig   `mapstructure:"password_policy"`
	RateLimit        RateLimitConfig        `mapstructure:"rate_limit"`
}

type LoginLockoutConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	MaxAttempts      int           `mapstructure:"max_attempts"`
	LockoutDuration  time.Duration `mapstructure:"lockout_duration"`
	ResetAfter       time.Duration `mapstructure:"reset_after"`
}

type PasswordPolicyConfig struct {
	Enabled          bool `mapstructure:"enabled"`
	MinLength        int  `mapstructure:"min_length"`
	MaxLength        int  `mapstructure:"max_length"`
	RequireUppercase bool `mapstructure:"require_uppercase"`
	RequireLowercase bool `mapstructure:"require_lowercase"`
	RequireDigit     bool `mapstructure:"require_digit"`
	RequireSpecial   bool `mapstructure:"require_special"`
}

type RateLimitConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	Requests   int           `mapstructure:"requests"`
	Window     time.Duration `mapstructure:"window"`
	BurstSize  int           `mapstructure:"burst_size"`
}