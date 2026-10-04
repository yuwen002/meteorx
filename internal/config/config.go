// Package config 定义应用配置结构体及加载逻辑，支持 YAML、环境变量和默认值三层合并。
package config

import (
	"strings"
	"time"
)

// Config 应用总配置，聚合各子模块配置项。
type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Redis        RedisConfig        `mapstructure:"redis"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	Log          LogConfig          `mapstructure:"log"`
	Security     SecurityConfig     `mapstructure:"security"`
	File         FileConfig         `mapstructure:"file"`
	Email        EmailConfig        `mapstructure:"email"`
	Client       ClientConfig       `mapstructure:"client"`
	IPLocation   IPLocationConfig   `mapstructure:"ip_location"`
	OAuth        OAuthConfig        `mapstructure:"oauth"`
	Auth         AuthConfig         `mapstructure:"auth"`
	WS           WSConfig           `mapstructure:"ws"`
	Notify       NotifyConfig       `mapstructure:"notify"`
	OTel         OTelConfig         `mapstructure:"otel"`
	Search       SearchConfig       `mapstructure:"search"`
	TaskReminder TaskReminderConfig `mapstructure:"task_reminder"`
}

// TaskReminderConfig 任务到期提醒定时任务配置。
// 未配置或为零值时由对应 Getter 回落内置默认，保持向后兼容。
type TaskReminderConfig struct {
	Horizon         time.Duration `mapstructure:"horizon"`          // 提醒窗口（提前量），默认 24h
	Interval        time.Duration `mapstructure:"interval"`         // 扫描间隔，默认 30m
	OverdueCooldown time.Duration `mapstructure:"overdue_cooldown"` // 逾期任务重复提醒的最小间隔，默认 24h
	BatchLimit      int           `mapstructure:"batch_limit"`      // 单次扫描发送提醒的最大任务数，默认 200
}

// GetHorizon 返回提醒窗口，未配置时默认 24 小时
func (c TaskReminderConfig) GetHorizon() time.Duration {
	if c.Horizon > 0 {
		return c.Horizon
	}
	return 24 * time.Hour
}

// GetInterval 返回扫描间隔，未配置时默认 30 分钟
func (c TaskReminderConfig) GetInterval() time.Duration {
	if c.Interval > 0 {
		return c.Interval
	}
	return 30 * time.Minute
}

// GetOverdueCooldown 返回逾期重复提醒冷却期，未配置时默认 24 小时
func (c TaskReminderConfig) GetOverdueCooldown() time.Duration {
	if c.OverdueCooldown > 0 {
		return c.OverdueCooldown
	}
	return 24 * time.Hour
}

// GetBatchLimit 返回单次扫描最大任务数，未配置时默认 200
func (c TaskReminderConfig) GetBatchLimit() int {
	if c.BatchLimit > 0 {
		return c.BatchLimit
	}
	return 200
}

// SearchConfig 全文检索引擎配置
type SearchConfig struct {
	// Provider 搜索引擎类型：meilisearch / none
	// 设为 none 或留空时降级为数据库 LIKE 搜索（向后兼容）
	Provider string `mapstructure:"provider"`
	// Host 搜索引擎服务地址（如 http://127.0.0.1:7700）
	Host string `mapstructure:"host"`
	// APIKey MeiliSearch 主密钥（用于索引管理，非搜索专用 key）
	APIKey string `mapstructure:"api_key"`
	// IndexPrefix 索引前缀（多环境隔离，如 "meteorx_dev_"），默认 "meteorx_"
	IndexPrefix string `mapstructure:"index_prefix"`
}

// WSConfig WebSocket 配置
type WSConfig struct {
	Enabled        bool `mapstructure:"enabled"`
	MaxConnPerUser int  `mapstructure:"max_conn_per_user"`
}

// NotifyConfig 多渠道通知配置
type NotifyConfig struct {
	// Webhook 通知渠道
	Webhook WebhookNotifyConfig `mapstructure:"webhook"`
}

// WebhookNotifyConfig Webhook 通知配置
type WebhookNotifyConfig struct {
	Enabled  bool     `mapstructure:"enabled"`
	Kind     string   `mapstructure:"kind"`      // generic / dingtalk / wechat / feishu
	URL      string   `mapstructure:"url"`       // Webhook URL
	Secret   string   `mapstructure:"secret"`    // 签名密钥
	OnEvents []string `mapstructure:"on_events"` // 触发事件列表：announcement/alert/cancel_request/subscription_expiry
}

// OTelConfig OpenTelemetry 可观测性配置
type OTelConfig struct {
	Enabled        bool    `mapstructure:"enabled"`
	Exporter       string  `mapstructure:"exporter"`    // stdout / otlp
	Endpoint       string  `mapstructure:"endpoint"`    // OTLP gRPC 端点（exporter=otlp 时生效）
	Insecure       bool    `mapstructure:"insecure"`    // 跳过 TLS（开发环境）
	SampleRate     float64 `mapstructure:"sample_rate"` // Trace 采样率 0.0~1.0
	ServiceName    string  `mapstructure:"service_name"`
	ServiceVersion string  `mapstructure:"service_version"`
}

// OAuthConfig OAuth2 登录配置
type OAuthConfig struct {
	Google          OAuthProviderConfig `mapstructure:"google"`
	GitHub          OAuthProviderConfig `mapstructure:"github"`
	RefreshTokenTTL time.Duration       `mapstructure:"refresh_token_ttl"`
	StateTTL        time.Duration       `mapstructure:"state_ttl"`
}

// GetRefreshTokenTTL 返回刷新令牌有效期，未配置时默认 7 天
func (c OAuthConfig) GetRefreshTokenTTL() time.Duration {
	if c.RefreshTokenTTL > 0 {
		return c.RefreshTokenTTL
	}
	return 7 * 24 * time.Hour
}

// GetStateTTL 返回 CSRF state 有效期，未配置时默认 10 分钟
func (c OAuthConfig) GetStateTTL() time.Duration {
	if c.StateTTL > 0 {
		return c.StateTTL
	}
	return 10 * time.Minute
}

// AuthConfig 认证模块通用配置
type AuthConfig struct {
	APITokenMaxTTL time.Duration `mapstructure:"api_token_max_ttl"`
}

// GetAPITokenMaxTTL 返回 API Token 最长有效期，未配置时默认 2160h（90 天）
func (c AuthConfig) GetAPITokenMaxTTL() time.Duration {
	if c.APITokenMaxTTL > 0 {
		return c.APITokenMaxTTL
	}
	return 2160 * time.Hour
}

// OAuthProviderConfig OAuth2 提供商配置
type OAuthProviderConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
}

// IPLocationConfig IP 归属地解析配置。
type IPLocationConfig struct {
	Provider string `mapstructure:"provider"` // 解析方式：http-api / ip2region
	DBPath   string `mapstructure:"db_path"`  // ip2region.xdb 文件路径（仅 provider=ip2region 时需要）
	Timeout  int    `mapstructure:"timeout"`  // HTTP API 超时时间（秒），默认 3
}

// EmailConfig SMTP 邮件发送配置。
type EmailConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	FromName string `mapstructure:"from_name"`
}

// ClientConfig 前端客户端配置（站点地址与跨域白名单）。
type ClientConfig struct {
	BaseURL string `mapstructure:"base_url"`
	// AllowedOrigins 额外允许跨域访问的前端来源列表；为空时仅允许 BaseURL
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// ServerConfig HTTP 服务器配置。
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
	// TestBypass 允许固定调试 Token "123456789" 以超级管理员身份直登。
	// 仅当 app.mode != release 且显式开启时才生效；默认关闭，生产切勿开启。
	TestBypass bool `mapstructure:"test_bypass"`
}

// DatabaseReplicaConfig 从库（只读副本）配置
type DatabaseReplicaConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	TLS      bool   `mapstructure:"tls"`
	// Weight 负载均衡权重（默认 1），权重越高分配到的查询越多
	Weight int `mapstructure:"weight"`
}

// DatabaseConfig 数据库连接配置，支持读写分离从库列表。
type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	TLS      bool   `mapstructure:"tls"`
	Debug    bool   `mapstructure:"debug"` // 开启后输出 SQL 日志
	// Timezone 业务时区名称（如 Asia/Shanghai），启动时据此强制 time.Local，
	// 使 time.Now()/时间解析/DSN loc=Local 与容器 TZ 环境变量解耦；为空时默认 Asia/Shanghai。
	Timezone string `mapstructure:"timezone"`
	// Replicas 只读从库列表（启用读写分离时配置）
	// 配置后 GORM 自动将 SELECT 查询路由到从库，INSERT/UPDATE/DELETE 仍走主库
	Replicas []DatabaseReplicaConfig `mapstructure:"replicas"`
}

// GetTimezone 返回业务时区名称，未配置时默认 Asia/Shanghai。
func (c DatabaseConfig) GetTimezone() string {
	if strings.TrimSpace(c.Timezone) == "" {
		return "Asia/Shanghai"
	}
	return strings.TrimSpace(c.Timezone)
}

// RedisConfig Redis 连接配置。
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig JWT 令牌签发配置。
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

// LogConfig 日志输出配置。
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// SecurityConfig 安全相关配置（登录锁定、密码策略、限流）。
type SecurityConfig struct {
	LoginLockout   LoginLockoutConfig   `mapstructure:"login_lockout"`
	PasswordPolicy PasswordPolicyConfig `mapstructure:"password_policy"`
	RateLimit      RateLimitConfig      `mapstructure:"rate_limit"`
}

// LoginLockoutConfig 登录失败锁定配置。
type LoginLockoutConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	MaxAttempts     int           `mapstructure:"max_attempts"`
	LockoutDuration time.Duration `mapstructure:"lockout_duration"`
	ResetAfter      time.Duration `mapstructure:"reset_after"`
}

// PasswordPolicyConfig 密码强度策略配置。
type PasswordPolicyConfig struct {
	Enabled          bool `mapstructure:"enabled"`
	MinLength        int  `mapstructure:"min_length"`
	MaxLength        int  `mapstructure:"max_length"`
	RequireUppercase bool `mapstructure:"require_uppercase"`
	RequireLowercase bool `mapstructure:"require_lowercase"`
	RequireDigit     bool `mapstructure:"require_digit"`
	RequireSpecial   bool `mapstructure:"require_special"`
}

// RateLimitConfig 请求频率限制配置。
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
	// SignKey 上传资源访问 URL 的独立签名密钥（不再复用 jwt.secret，避免更换登录密钥导致存量公开链接失效）。
	// 支持逗号分隔传入多把密钥以实现平滑轮换：第一把为当前签发密钥（新链接使用），
	// 其余仅用于校验存量链接（宽限期后可移除）；为空时回退 jwt.secret 以兼容未配置的旧部署。
	SignKey string `mapstructure:"sign_key"`
	// 云存储配置（预留，未启用时为空即可）
	Cloud struct {
		Endpoint  string `mapstructure:"endpoint"`   // OSS/S3 endpoint
		AccessKey string `mapstructure:"access_key"` // AccessKey
		SecretKey string `mapstructure:"secret_key"` // SecretKey
		Bucket    string `mapstructure:"bucket"`     // Bucket 名称
		Region    string `mapstructure:"region"`     // Region
	} `mapstructure:"cloud"`
}

// UploadSignKey 返回当前用于签发上传资源短时效访问 URL 的密钥。
// 轮换配置下取密钥链首项；未配置 file.sign_key 时回退 jwt.secret（兼容旧部署）。
func (f FileConfig) UploadSignKey(jwtSecret string) string {
	keys := f.UploadSignKeys(jwtSecret)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

// UploadSignKeys 返回完整签名密钥链（首项签发 + 全部校验存量链接）。
// 语义：sign_key 按逗号分隔（形如 "new-key,old-key"），空项被剔除。
func (f FileConfig) UploadSignKeys(jwtSecret string) []string {
	if strings.TrimSpace(f.SignKey) == "" {
		if strings.TrimSpace(jwtSecret) == "" {
			return nil
		}
		return []string{jwtSecret}
	}
	var keys []string
	for _, part := range strings.Split(f.SignKey, ",") {
		if k := strings.TrimSpace(part); k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}
