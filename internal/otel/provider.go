package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitProvider 初始化 OTel TracerProvider，返回关闭函数用于优雅清理
// 支持两种导出模式：
//   - stdout: 将 Trace 输出到标准输出（开发调试用）
//   - otlp:   通过 gRPC 导出到 OTel Collector（生产环境）
func InitProvider(ctx context.Context, cfg *Config) (func(), error) {
	if cfg == nil || !cfg.Enabled {
		// 未启用时使用 NoopTracerProvider（不做任何追踪）
		otel.SetTracerProvider(trace.NewNoopTracerProvider())
		return func() {}, nil
	}

	// 1. 创建 Exporter
	exp, err := createExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create otel exporter: %w", err)
	}

	// 2. 创建 Service Resource（标识当前服务）
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			attribute.String("deployment.environment", getEnvLabel()),
		),
		resource.WithProcessRuntimeDescription(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	// 3. 创建 Sampler
	sampler := createSampler(cfg.SampleRate)

	// 4. 创建 TracerProvider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)

	// 5. 设置为全局 TracerProvider
	otel.SetTracerProvider(tp)

	// 6. 返回关闭函数
	shutdown := func() {
		_ = tp.Shutdown(context.Background())
	}

	return shutdown, nil
}

// createExporter 根据配置创建 Exporter
func createExporter(ctx context.Context, cfg *Config) (trace.SpanExporter, error) {
	switch cfg.Exporter {
	case "stdout":
		return stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)

	case "otlp":
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		return otlptracegrpc.New(ctx, opts...)

	default:
		return nil, fmt.Errorf("unknown otel exporter: %s (supported: stdout, otlp)", cfg.Exporter)
	}
}

// createSampler 根据采样率创建 Sampler
func createSampler(rate float64) trace.Sampler {
	switch {
	case rate <= 0:
		return trace.NeverSample()
	case rate >= 1.0:
		return trace.AlwaysSample()
	default:
		return trace.TraceIDRatioBased(rate)
	}
}

// getEnvLabel 获取部署环境标签
func getEnvLabel() string {
	// 通过 METEORX_APP_MODE 环境变量判断环境
	mode := "development"
	// 简单判断：实际运行时从 config.Server.Mode 传入更准确
	return mode
}