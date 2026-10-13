// Package dbmonitor 提供数据库慢查询采集能力：以 GORM Logger 中间件形式捕获超过阈值的 SQL，
// 在内存环形缓冲区中保留最近若干条，供平台管理员看板实时查看，无需依赖外部慢查询日志。
package dbmonitor

import (
	"context"
	"sync"
	"time"

	"meteorx/internal/common/contextx"

	"gorm.io/gorm/logger"
)

// Entry 单次慢查询记录。
type Entry struct {
	SQL        string    `json:"sql"`         // 执行的 SQL 语句
	DurationMs int64     `json:"duration_ms"` // 耗时（毫秒）
	Rows       int64     `json:"rows"`        // 影响/返回行数
	Error      string    `json:"error,omitempty"`
	TraceID    string    `json:"trace_id,omitempty"` // 关联链路追踪 ID
	Time       time.Time `json:"time"`               // 查询发生时间（UTC）
}

// Stats 慢查询整体统计快照。
type Stats struct {
	ThresholdMs   int64   `json:"threshold_ms"`   // 慢查询判定阈值（毫秒）
	TotalQueries  int64   `json:"total_queries"`  // 采样期间捕获的 SQL 总数
	SlowQueries   int64   `json:"slow_queries"`   // 超过阈值的慢查询数
	MaxDurationMs int64   `json:"max_duration_ms"` // 观测到的最大耗时
	AvgSlowMs     float64 `json:"avg_slow_ms"`     // 慢查询平均耗时
	Capacity      int     `json:"capacity"`        // 缓冲区容量
}

// Snapshot 慢查询看板响应体：统计 + 最近慢查询列表（时间倒序）。
type Snapshot struct {
	Stats   Stats   `json:"stats"`
	Entries []Entry `json:"entries"`
}

// store 线程安全的环形缓冲区，保留最近 max 条慢查询并累计统计。
type store struct {
	mu        sync.Mutex
	entries   []Entry
	max       int
	threshold time.Duration

	totalQueries int64
	slowQueries  int64
	maxDuration  int64
	sumSlowMs    int64
}

func newStore(capacity int, threshold time.Duration) *store {
	if capacity <= 0 {
		capacity = 200
	}
	return &store{
		entries:   make([]Entry, 0, capacity),
		max:       capacity,
		threshold: threshold,
	}
}

// Init 设置全局慢查询缓冲区容量与阈值（幂等，容量保留既有记录）。
func Init(threshold time.Duration, capacity int) {
	Default.mu.Lock()
	defer Default.mu.Unlock()
	Default.threshold = threshold
	if capacity > 0 {
		Default.max = capacity
	}
}

// add 记录一条慢查询。
func (s *store) add(e Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.totalQueries++
	s.slowQueries++
	if e.DurationMs > s.maxDuration {
		s.maxDuration = e.DurationMs
	}
	s.sumSlowMs += e.DurationMs
	// 时间倒序：新记录插入头部，超出容量则丢弃尾部
	s.entries = append(s.entries, Entry{})
	copy(s.entries[1:], s.entries[:len(s.entries)-1])
	s.entries[0] = e
	if len(s.entries) > s.max {
		s.entries = s.entries[:s.max]
	}
}

// countTotal 累计非慢查询计数（仅统计总量，不落明细）。
func (s *store) countNormal() {
	s.mu.Lock()
	s.totalQueries++
	s.mu.Unlock()
}

// snapshot 返回当前统计与最近慢查询列表的拷贝。
func (s *store) snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := make([]Entry, len(s.entries))
	copy(entries, s.entries)
	var avg float64
	if s.slowQueries > 0 {
		avg = float64(s.sumSlowMs) / float64(s.slowQueries)
	}
	return Snapshot{
		Stats: Stats{
			ThresholdMs:   int64(s.threshold.Milliseconds()),
			TotalQueries:  s.totalQueries,
			SlowQueries:   s.slowQueries,
			MaxDurationMs: s.maxDuration,
			AvgSlowMs:     avg,
			Capacity:      s.max,
		},
		Entries: entries,
	}
}

// reset 清空缓冲区与统计（用于管理员手动重置）。
func (s *store) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = s.entries[:0]
	s.totalQueries = 0
	s.slowQueries = 0
	s.maxDuration = 0
	s.sumSlowMs = 0
}

// Default 全局慢查询缓冲区，供采集器写入、看板读取。
var Default = newStore(200, 500*time.Millisecond)

// Snapshot 返回全局缓冲区快照。
func GetSnapshot() Snapshot { return Default.snapshot() }

// Reset 清空全局缓冲区。
func Reset() { Default.reset() }

// Logger 包装 GORM 内置 logger，在 Trace 时捕获超过阈值的慢查询到全局缓冲区。
type Logger struct {
	inner     logger.Interface
	threshold time.Duration
}

// NewLogger 创建采集器，threshold 为慢查询判定阈值，inner 为被包装的原始日志器。
func NewLogger(threshold time.Duration, inner logger.Interface) *Logger {
	return &Logger{inner: inner, threshold: threshold}
}

func (l *Logger) LogMode(level logger.LogLevel) logger.Interface {
	l.inner = l.inner.LogMode(level)
	return l
}

func (l *Logger) Info(ctx context.Context, msg string, args ...interface{}) {
	l.inner.Info(ctx, msg, args...)
}

func (l *Logger) Warn(ctx context.Context, msg string, args ...interface{}) {
	l.inner.Warn(ctx, msg, args...)
}

func (l *Logger) Error(ctx context.Context, msg string, args ...interface{}) {
	l.inner.Error(ctx, msg, args...)
}

// Trace 计算耗时，超过阈值则记入慢查询缓冲区，随后委托原始 logger。
func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	if l.threshold > 0 && elapsed >= l.threshold {
		sql, rows := fc()
		e := Entry{
			SQL:        sql,
			DurationMs: elapsed.Milliseconds(),
			Rows:       rows,
			Time:       begin.UTC(),
			TraceID:    contextx.GetTraceID(ctx),
		}
		if err != nil {
			e.Error = err.Error()
		}
		Default.add(e)
	} else {
		Default.countNormal()
	}
	l.inner.Trace(ctx, begin, fc, err)
}

// 确保实现 GORM logger 接口。
var _ logger.Interface = (*Logger)(nil)
