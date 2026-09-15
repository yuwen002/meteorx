package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitProvider 初始化 OTel TracerProvider，返回关闭函数用于优雅清理
// 支持两种导出模式：
//   - stdout: 将 Trace 输出到标准输出（开发调试用）
//   - otlp:   通过 gRPC 导出到 OTel Collector（生产环境）
//
// envMode 是部署环境（如 "debug", "release" 等），用于标记资源属性
func InitProvider(ctx context.Context, cfg *Config, envMode string) (func(), error) {
	if cfg == nil || !cfg.Enabled {
		// 未启用时使用 NoopTracerProvider（不做任何追踪）
		tp := trace.NewTracerProvider()
		otel.SetTracerProvider(tp)
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
			attribute.String("deployment.environment", envMode),
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

	// 6. 设置全局 TextMap 传播器（支持 W3C Trace Context 标准）
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	// 7. 返回关闭函数
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