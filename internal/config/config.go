package config

import "time"

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Log        LogConfig        `mapstructure:"log"`
	Security   SecurityConfig   `mapstructure:"security"`
	File       FileConfig       `mapstructure:"file"`
	Email      EmailConfig      `mapstructure:"email"`
	Client     ClientConfig     `mapstructure:"client"`
	IPLocation IPLocationConfig `mapstructure:"ip_location"`
}

type IPLocationConfig struct {
	Provider string `mapstructure:"provider"` // 解析方式：http-api / ip2region
	DBPath   string `mapstructure:"db_path"`  // ip2region.xdb 文件路径（仅 provider=ip2region 时需要）
	Timeout  int    `mapstructure:"timeout"`  // HTTP API 超时时间（秒），默认 3
}

type EmailConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	FromName string `mapstructure:"from_name"`
}

type ClientConfig struct {
	BaseURL string `mapstructure:"base_url"`
	// AllowedOrigins 额外允许跨域访问的前端来源列表；为空时仅允许 BaseURL
	AllowedOrigins []string `mapstructure:"allowed_origins"`
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
	LoginLockout   LoginLockoutConfig   `mapstructure:"login_lockout"`
	PasswordPolicy PasswordPolicyConfig `mapstructure:"password_policy"`
	RateLimit      RateLimitConfig      `mapstructure:"rate_limit"`
}

type LoginLockoutConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	MaxAttempts     int           `mapstructure:"max_attempts"`
	LockoutDuration time.Duration `mapstructure:"lockout_duration"`
	ResetAfter      time.Duration `mapstructure:"reset_after"`
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
	Enabled   bool          `mapstructure:"enabled"`
	Requests  int           `mapstructure:"requests"`
	Window    time.Duration `mapstructure:"window"`
	BurstSize int           `mapstructure:"burst_size"`
}

// FileConfig 文件上传配置
type FileConfig struct {
	UploadPath   string   `mapstructure:"upload_path"`   // 文件上传存储路径
	UploadURL    string   `mapstructure:"upload_url"`    // 文件访问URL前缀
	MaxFileSize  int64    `mapstructure:"max_file_size"` // 最大文件大小（字节）
	AllowedTypes []string `mapstructure:"allowed_types"` // 允许的文件MIME类型
	StorageType  string   `mapstructure:"storage_type"`  // 存储类型: local / oss / s3（预留）
	// 云存储配置（预留，未启用时为空即可）
	Cloud struct {
		Endpoint  string `mapstructure:"endpoint"`   // OSS/S3 endpoint
		AccessKey string `mapstructure:"access_key"` // AccessKey
		SecretKey string `mapstructure:"secret_key"` // SecretKey
		Bucket    string `mapstructure:"bucket"`     // Bucket 名称
		Region    string `mapstructure:"region"`     // Region
	} `mapstructure:"cloud"`
}
