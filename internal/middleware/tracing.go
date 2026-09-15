package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// tracer 是当前包级别的 Tracer 实例，用于创建 Span
var tracer = otel.Tracer("meteorx/middleware")

// TracingMiddleware OTel 分布式追踪中间件
//
// 功能：
//   - 从 HTTP 请求头中提取父 Trace 上下文（支持 W3C Trace Context）
//   - 根据 Chi 路由模式创建具有语义名称的 Span
//   - 自动记录 HTTP 方法、URL、状态码等标准属性
//   - 将 Trace ID 注入响应头（方便调试）
//
// 使用方式：在 Chi Router 中作为全局中间件注册。
//
//	r.Use(middleware.TracingMiddleware)
func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 从请求头中提取传播的 Trace 上下文（如果存在）
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// 2. 获取 Chi 路由模式（如 GET /api/v1/tenants/{id}）
		routePattern := getRoutePattern(r)
		spanName := r.Method + " " + routePattern

		// 3. 创建 Span，设置标准属性
		opts := []trace.SpanStartOption{
			trace.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", routePattern),
				attribute.String("url.full", r.URL.String()),
				attribute.String("url.scheme", r.URL.Scheme),
				attribute.String("server.address", r.Host),
				attribute.String("user_agent.original", r.UserAgent()),
				attribute.String("http.target", r.URL.RequestURI()),
			),
			trace.WithSpanKind(trace.SpanKindServer),
		}

		// 4. 启动 Span
		ctx, span := tracer.Start(ctx, spanName, opts...)
		defer span.End()

		// 5. 将 Trace ID 写入响应头（便于前端/调试工具关联）
		spanContext := span.SpanContext()
		if spanContext.HasTraceID() {
			w.Header().Set("X-Trace-ID", spanContext.TraceID().String())
		}

		// 6. 包装 ResponseWriter 以捕获状态码
		rw := NewResponseWriter(w)

		// 7. 将带有 Span 的 Context 传递下去
		next.ServeHTTP(rw, r.WithContext(ctx))

		// 8. 记录响应状态码，并根据状态码设置 Span 状态
		span.SetAttributes(attribute.Int("http.response.status_code", rw.StatusCode))

		if rw.StatusCode >= 500 {
			span.SetAttributes(attribute.String("error.type", "server_error"))
			span.SetStatus(codes.Error, http.StatusText(rw.StatusCode))
		} else if rw.StatusCode >= 400 {
			span.SetAttributes(attribute.String("error.type", "client_error"))
		}
	})
}

// TracingHTTPHandler 包装一个 http.Handler 使其支持 OTel 分布式追踪。
// 适用于不能直接使用中间件的地方（如路由组级别的细粒度控制）。
//
//	name: 操作名称（如 "UploadFile"），会出现在 Trace 中
func TracingHTTPHandler(name string, handler http.Handler) http.Handler {
	return otelhttp.NewHandler(handler, name)
}

// getRoutePattern 从 Chi 上下文中提取路由模式。
// 如果无法获取，则回退为原始 URL 路径。
//
//	示例返回: "GET /api/v1/tenants/{id}"
func getRoutePattern(r *http.Request) string {
	// 优先从 Chi 路由上下文获取匹配的模式
	routeCtx := chi.RouteContext(r.Context())
	if routeCtx != nil {
		if pattern := routeCtx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	// 回退：使用 URL 路径
	return r.URL.Path
}