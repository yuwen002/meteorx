package dbmonitor

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

// infLogger 是被包装的桩 logger，用于验证委托调用。
type infLogger struct{ traceCalls int }

func (f *infLogger) LogMode(logger.LogLevel) logger.Interface { return f }
func (f *infLogger) Info(context.Context, string, ...interface{})  {}
func (f *infLogger) Warn(context.Context, string, ...interface{})  {}
func (f *infLogger) Error(context.Context, string, ...interface{}) {}
func (f *infLogger) Trace(context.Context, time.Time, func() (string, int64), error) {
	f.traceCalls++
}

func TestStoreAddAndSnapshot(t *testing.T) {
	s := newStore(3, 500*time.Millisecond)
	s.add(Entry{SQL: "a", DurationMs: 600})
	s.add(Entry{SQL: "b", DurationMs: 900})
	s.add(Entry{SQL: "c", DurationMs: 700})
	// 超出容量，最早的 d 应把 a 挤出
	s.add(Entry{SQL: "d", DurationMs: 500})

	snap := s.snapshot()
	if snap.Stats.TotalQueries != 4 || snap.Stats.SlowQueries != 4 {
		t.Fatalf("stats count mismatch: %+v", snap.Stats)
	}
	if snap.Stats.MaxDurationMs != 900 {
		t.Errorf("max duration want 900 got %d", snap.Stats.MaxDurationMs)
	}
	// 环形缓冲保留最近 3 条，头部为最新 d
	if len(snap.Entries) != 3 {
		t.Fatalf("entries len want 3 got %d", len(snap.Entries))
	}
	if snap.Entries[0].SQL != "d" {
		t.Errorf("head entry want d got %s", snap.Entries[0].SQL)
	}
}

func TestStoreCountNormal(t *testing.T) {
	s := newStore(10, 500*time.Millisecond)
	s.countNormal()
	s.countNormal()
	snap := s.snapshot()
	if snap.Stats.TotalQueries != 2 || snap.Stats.SlowQueries != 0 {
		t.Fatalf("normal count mismatch: %+v", snap.Stats)
	}
}

func TestStoreReset(t *testing.T) {
	s := newStore(10, 500*time.Millisecond)
	s.add(Entry{SQL: "a", DurationMs: 600})
	s.reset()
	snap := s.snapshot()
	if snap.Stats.TotalQueries != 0 || len(snap.Entries) != 0 {
		t.Fatalf("reset should clear store: %+v", snap)
	}
}

func TestLoggerTraceCapturesSlow(t *testing.T) {
	defer Reset()
	Reset()
	inner := &infLogger{}
	l := NewLogger(10*time.Millisecond, inner)

	// 慢查询：begin 早于当前足够多，elapsed 超过阈值
	begin := time.Now().Add(-100 * time.Millisecond)
	l.Trace(context.Background(), begin, func() (string, int64) { return "SELECT slow", 1 }, nil)

	snap := GetSnapshot()
	if snap.Stats.SlowQueries != 1 {
		t.Fatalf("want 1 slow query captured got %d", snap.Stats.SlowQueries)
	}
	if len(snap.Entries) == 0 || snap.Entries[0].SQL != "SELECT slow" {
		t.Fatalf("slow entry not recorded: %+v", snap.Entries)
	}
	if inner.traceCalls != 1 {
		t.Errorf("inner logger should be delegated once, got %d", inner.traceCalls)
	}
}

func TestLoggerTraceSkipsFast(t *testing.T) {
	defer Reset()
	Reset()
	inner := &infLogger{}
	l := NewLogger(1*time.Second, inner)

	// 快查询：elapsed 远小于阈值，仅计入总量
	l.Trace(context.Background(), time.Now(), func() (string, int64) { return "SELECT fast", 1 }, errors.New("boom"))

	snap := GetSnapshot()
	if snap.Stats.TotalQueries != 1 || snap.Stats.SlowQueries != 0 {
		t.Fatalf("fast query should only count total: %+v", snap.Stats)
	}
	if inner.traceCalls != 1 {
		t.Errorf("inner logger should be delegated once, got %d", inner.traceCalls)
	}
}
