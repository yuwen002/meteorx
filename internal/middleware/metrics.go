package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// MetricsCollector 指标收集器
type MetricsCollector struct {
	mu sync.RWMutex

	// 请求计数
	RequestCount   int64
	RequestDuration time.Duration

	// 按状态码分类的计数
	StatusCounts map[int]int64

	// 按路径分类的统计
	PathStats map[string]*PathStats

	// 启动时间
	StartTime time.Time
}

// PathStats 路径统计信息
type PathStats struct {
	Count    int64
	TotalTime time.Duration
	MinTime   time.Duration
	MaxTime   time.Duration
	LastError string
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		StatusCounts: make(map[int]int64),
		PathStats:    make(map[string]*PathStats),
		StartTime:    time.Now(),
	}
}

// Metrics 指标收集中间件
func (mc *MetricsCollector) Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter
		rw := NewResponseWriter(w)

		// 处理请求
		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// 更新指标
		mc.RecordMetrics(r.URL.Path, rw.StatusCode, duration)
	})
}

// RecordMetrics 记录指标
func (mc *MetricsCollector) RecordMetrics(path string, statusCode int, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.RequestCount++
	mc.RequestDuration += duration

	// 更新状态码计数
	mc.StatusCounts[statusCode]++

	// 更新路径统计
	stats, exists := mc.PathStats[path]
	if !exists {
		stats = &PathStats{
			MinTime: duration,
			MaxTime: duration,
		}
		mc.PathStats[path] = stats
	}

	stats.Count++
	stats.TotalTime += duration

	if duration < stats.MinTime {
		stats.MinTime = duration
	}
	if duration > stats.MaxTime {
		stats.MaxTime = duration
	}
}

// GetSummary 获取指标摘要
func (mc *MetricsCollector) GetSummary() map[string]any {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	uptime := time.Since(mc.StartTime)
	avgDuration := time.Duration(0)
	if mc.RequestCount > 0 {
		avgDuration = mc.RequestDuration / time.Duration(mc.RequestCount)
	}

	summary := map[string]any{
		"uptime_seconds":   uptime.Seconds(),
		"total_requests":   mc.RequestCount,
		"avg_response_ms":  avgDuration.Milliseconds(),
		"status_codes":     mc.StatusCounts,
		"requests_per_sec": float64(mc.RequestCount) / uptime.Seconds(),
	}

	return summary
}

// GetPathStats 获取路径统计
func (mc *MetricsCollector) GetPathStats() map[string]*PathStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// 返回副本
	result := make(map[string]*PathStats)
	for path, stats := range mc.PathStats {
		result[path] = &PathStats{
			Count:     stats.Count,
			TotalTime: stats.TotalTime,
			MinTime:   stats.MinTime,
			MaxTime:   stats.MaxTime,
			LastError: stats.LastError,
		}
	}

	return result
}

// MetricsHandler 创建指标查询 HTTP Handler
func (mc *MetricsCollector) MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		summary := mc.GetSummary()
		pathStats := mc.GetPathStats()

		// 简单 JSON 输出（响应流已部分写入，写入失败无法回退，显式忽略）
		_, _ = w.Write([]byte("{\n"))
		_, _ = w.Write([]byte("  \"summary\": {\n"))
		_, _ = w.Write([]byte("    \"uptime_seconds\": "))
		_, _ = fmt.Fprintf(w, "%.2f", summary["uptime_seconds"])
		_, _ = w.Write([]byte(",\n"))
		_, _ = w.Write([]byte("    \"total_requests\": "))
		_, _ = fmt.Fprintf(w, "%d", summary["total_requests"])
		_, _ = w.Write([]byte(",\n"))
		_, _ = w.Write([]byte("    \"avg_response_ms\": "))
		_, _ = fmt.Fprintf(w, "%d", summary["avg_response_ms"])
		_, _ = w.Write([]byte(",\n"))
		_, _ = w.Write([]byte("    \"requests_per_sec\": "))
		_, _ = fmt.Fprintf(w, "%.2f", summary["requests_per_sec"])
		_, _ = w.Write([]byte("\n  },\n"))

		_, _ = w.Write([]byte("  \"path_stats\": {\n"))
		i := 0
		for path, stats := range pathStats {
			_, _ = w.Write([]byte("    \""))
			_, _ = w.Write([]byte(path))
			_, _ = w.Write([]byte("\": {\n"))
			_, _ = w.Write([]byte("      \"count\": "))
			_, _ = fmt.Fprintf(w, "%d", stats.Count)
			_, _ = w.Write([]byte(",\n"))
			_, _ = w.Write([]byte("      \"avg_ms\": "))
			_, _ = fmt.Fprintf(w, "%d", stats.TotalTime/time.Duration(stats.Count)/time.Millisecond)
			_, _ = w.Write([]byte(",\n"))
			_, _ = w.Write([]byte("      \"min_ms\": "))
			_, _ = fmt.Fprintf(w, "%d", stats.MinTime/time.Millisecond)
			_, _ = w.Write([]byte(",\n"))
			_, _ = w.Write([]byte("      \"max_ms\": "))
			_, _ = fmt.Fprintf(w, "%d", stats.MaxTime/time.Millisecond)
			_, _ = w.Write([]byte("\n    }"))
			if i < len(pathStats)-1 {
				_, _ = w.Write([]byte(","))
			}
			_, _ = w.Write([]byte("\n"))
			i++
		}
		_, _ = w.Write([]byte("  }\n"))
		_, _ = w.Write([]byte("}\n"))
	})
}