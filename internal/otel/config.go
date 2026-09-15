package otel

// Config OTel 可观测性配置
type Config struct {
	// Enabled 是否启用 OTel
	Enabled bool `mapstructure:"enabled"`

	// Exporter 导出方式：stdout（开发调试） / otlp（生产环境）
	Exporter string `mapstructure:"exporter"`

	// Endpoint OTLP gRPC 端点地址（exporter=otlp 时生效）
	// 例如：otel-collector:4317
	Endpoint string `mapstructure:"endpoint"`

	// Insecure OTLP gRPC 是否跳过 TLS（开发环境通常设为 true）
	Insecure bool `mapstructure:"insecure"`

	// SampleRate Trace 采样率（0.0 ~ 1.0），1.0 表示 100% 采样
	// 生产环境建议设为 0.1 以控制成本
	SampleRate float64 `mapstructure:"sample_rate"`

	// ServiceName 服务名称，用于在 Trace/Metric 中标识当前服务
	ServiceName string `mapstructure:"service_name"`

	// ServiceVersion 服务版本号，用于标识部署版本
	ServiceVersion string `mapstructure:"service_version"`
}

// DefaultConfig 返回 OTel 默认配置
func DefaultConfig() *Config {
	return &Config{
		Enabled:        true,
		Exporter:       "stdout",
		Endpoint:       "localhost:4317",
		Insecure:       true,
		SampleRate:     1.0,
		ServiceName:    "meteorx",
		ServiceVersion: "1.0.0",
	}
}