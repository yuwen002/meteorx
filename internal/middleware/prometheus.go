package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// PrometheusMetrics Prometheus 格式指标收集器
type PrometheusMetrics struct {
	mu sync.RWMutex

	// HTTP 请求总数（按状态码和路径分类）
	httpRequestsTotal map[string]int64

	// HTTP 请求持续时间（按路径分类）
	httpRequestDuration map[string]time.Duration

	// 启动时间
	startTime time.Time
}

// NewPrometheusMetrics 创建 Prometheus 指标收集器
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		httpRequestsTotal:   make(map[string]int64),
		httpRequestDuration: make(map[string]time.Duration),
		startTime:           time.Now(),
	}
}

// Middleware Prometheus 指标中间件
func (pm *PrometheusMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter
		rw := NewResponseWriter(w)

		// 处理请求
		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// 记录指标
		pm.Record(r.URL.Path, rw.StatusCode, duration)
	})
}

// Record 记录指标
func (pm *PrometheusMetrics) Record(path string, statusCode int, duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 记录请求总数
	label := fmt.Sprintf("%d:%s", statusCode, path)
	pm.httpRequestsTotal[label]++

	// 记录请求持续时间
	pm.httpRequestDuration[path] += duration
}

// Handler 返回 Prometheus 格式的指标 HTTP Handler
func (pm *PrometheusMetrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		pm.mu.RLock()
		defer pm.mu.RUnlock()

		uptime := time.Since(pm.startTime).Seconds()

		// 写入指标
		fmt.Fprintf(w, "# HELP meteorx_uptime_seconds 服务运行时间（秒）\n")
		fmt.Fprintf(w, "# TYPE meteorx_uptime_seconds gauge\n")
		fmt.Fprintf(w, "meteorx_uptime_seconds %.2f\n", uptime)

		fmt.Fprintf(w, "# HELP meteorx_http_requests_total HTTP 请求总数\n")
		fmt.Fprintf(w, "# TYPE meteorx_http_requests_total counter\n")
		for label, count := range pm.httpRequestsTotal {
			fmt.Fprintf(w, "meteorx_http_requests_total{label=\"%s\"} %d\n", label, count)
		}

		fmt.Fprintf(w, "# HELP meteorx_http_request_duration_seconds HTTP 请求总耗时（秒）\n")
		fmt.Fprintf(w, "# TYPE meteorx_http_request_duration_seconds counter\n")
		for path, duration := range pm.httpRequestDuration {
			fmt.Fprintf(w, "meteorx_http_request_duration_seconds{path=\"%s\"} %.3f\n", path, duration.Seconds())
		}
	})
}