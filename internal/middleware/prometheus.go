package middleware

import (
	"fmt"
	"net/http"
	"runtime"
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

		// === Go 运行时指标 ===
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		fmt.Fprintf(w, "# HELP go_memstats_alloc_bytes 已分配堆内存（字节）\n")
		fmt.Fprintf(w, "# TYPE go_memstats_alloc_bytes gauge\n")
		fmt.Fprintf(w, "go_memstats_alloc_bytes{job=\"meteorx\"} %d\n", m.Alloc)

		fmt.Fprintf(w, "# HELP go_memstats_sys_bytes 从系统申请的总内存（字节）\n")
		fmt.Fprintf(w, "# TYPE go_memstats_sys_bytes gauge\n")
		fmt.Fprintf(w, "go_memstats_sys_bytes{job=\"meteorx\"} %d\n", m.Sys)

		fmt.Fprintf(w, "# HELP go_memstats_heap_inuse_bytes 正在使用的堆内存（字节）\n")
		fmt.Fprintf(w, "# TYPE go_memstats_heap_inuse_bytes gauge\n")
		fmt.Fprintf(w, "go_memstats_heap_inuse_bytes{job=\"meteorx\"} %d\n", m.HeapInuse)

		fmt.Fprintf(w, "# HELP go_memstats_stack_inuse_bytes 正在使用的栈内存（字节）\n")
		fmt.Fprintf(w, "# TYPE go_memstats_stack_inuse_bytes gauge\n")
		fmt.Fprintf(w, "go_memstats_stack_inuse_bytes{job=\"meteorx\"} %d\n", m.StackInuse)

		fmt.Fprintf(w, "# HELP go_memstats_gc_sys_bytes GC 元数据使用内存（字节）\n")
		fmt.Fprintf(w, "# TYPE go_memstats_gc_sys_bytes gauge\n")
		fmt.Fprintf(w, "go_memstats_gc_sys_bytes{job=\"meteorx\"} %d\n", m.GCSys)

		fmt.Fprintf(w, "# HELP go_memstats_next_gc_bytes 下次 GC 触发阈值（字节）\n")
		fmt.Fprintf(w, "# TYPE go_memstats_next_gc_bytes gauge\n")
		fmt.Fprintf(w, "go_memstats_next_gc_bytes{job=\"meteorx\"} %d\n", m.NextGC)

		fmt.Fprintf(w, "# HELP go_memstats_last_gc_time_seconds 上次 GC 时间（Unix 时间戳）\n")
		fmt.Fprintf(w, "# TYPE go_memstats_last_gc_time_seconds gauge\n")
		fmt.Fprintf(w, "go_memstats_last_gc_time_seconds{job=\"meteorx\"} %.2f\n", float64(m.LastGC)/1e9)

		fmt.Fprintf(w, "# HELP go_memstats_num_gc_total GC 次数\n")
		fmt.Fprintf(w, "# TYPE go_memstats_num_gc_total counter\n")
		fmt.Fprintf(w, "go_memstats_num_gc_total{job=\"meteorx\"} %d\n", m.NumGC)

		fmt.Fprintf(w, "# HELP go_memstats_gc_cpu_fraction GC 占 CPU 比例\n")
		fmt.Fprintf(w, "# TYPE go_memstats_gc_cpu_fraction gauge\n")
		fmt.Fprintf(w, "go_memstats_gc_cpu_fraction{job=\"meteorx\"} %.4f\n", m.GCCPUFraction)

		fmt.Fprintf(w, "# HELP go_goroutines 当前 goroutine 数量\n")
		fmt.Fprintf(w, "# TYPE go_goroutines gauge\n")
		fmt.Fprintf(w, "go_goroutines{job=\"meteorx\"} %d\n", runtime.NumGoroutine())

		var numCPU = runtime.NumCPU()
		fmt.Fprintf(w, "# HELP go_threads 当前线程数量\n")
		fmt.Fprintf(w, "# TYPE go_threads gauge\n")
		fmt.Fprintf(w, "go_threads{job=\"meteorx\"} %d\n", numCPU)

		// 进程级指标
		fmt.Fprintf(w, "# HELP process_start_time_seconds 进程启动时间（Unix 时间戳）\n")
		fmt.Fprintf(w, "# TYPE process_start_time_seconds gauge\n")
		fmt.Fprintf(w, "process_start_time_seconds{job=\"meteorx\"} %.2f\n", float64(pm.startTime.Unix()))
	})
}