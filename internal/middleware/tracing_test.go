package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/pkg/logger"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// traceIDPattern W3C 规定的 32 位小写十六进制 trace id
var traceIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

// installTracer 安装一个真实采样、无导出的 TracerProvider 与 W3C 传播器，
// 使 TracingMiddleware 能生成有效的 SpanContext；返回清理函数还原全局状态。
func installTracer() func() {
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return func() { _ = tp.Shutdown(context.Background()) }
}

// TestTracingMiddleware_InjectsRealTraceID 验证追踪中间件把真实 OTel Trace ID
// 同时注入 contextx（供审计落库）与 logger（供结构化日志），并回填响应头，三处同源。
func TestTracingMiddleware_InjectsRealTraceID(t *testing.T) {
	defer installTracer()()

	var ctxTraceID, logTraceID string
	handler := TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxTraceID = contextx.GetTraceID(r.Context())
		logTraceID = logger.GetTraceID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))

	// 下游 context 中的 trace id 必须是有效的 32 位十六进制
	require.Regexp(t, traceIDPattern, ctxTraceID, "contextx 应携带真实 OTel trace id")
	// 审计（contextx）与日志（logger）必须拿到同一个值
	require.Equal(t, ctxTraceID, logTraceID, "logger 与 contextx 的 trace id 必须同源")
	// 响应头回填的 X-Trace-ID 与注入值一致，客户端可据此跳查链路
	require.Equal(t, ctxTraceID, rec.Header().Get("X-Trace-ID"))
}

// TestTracingMiddleware_ContinuesPropagatedTrace 上游若传播了 W3C traceparent，
// 中间件必须延续其 trace id（而非另起），保证跨服务链路可拼接。
func TestTracingMiddleware_ContinuesPropagatedTrace(t *testing.T) {
	defer installTracer()()

	const upstreamTraceID = "0af7651916cd43dd8448eb211c80319c"

	var gotTraceID string
	handler := TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = contextx.GetTraceID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("traceparent", "00-"+upstreamTraceID+"-00f067aa0ba902b7-01")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	require.Equal(t, upstreamTraceID, gotTraceID, "应延续上游传播的 trace id")
}
