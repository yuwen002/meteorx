package timeutil

import (
	"testing"
	"time"
)

// TestStartOfDay_Shanghai 校验东八区下 Truncate(24h) 的 UTC 截断会得到当地 08:00，
// 而 StartOfDay 必须返回当地日历零点 00:00，用于修正历史日期分桶 bug。
func TestStartOfDay_Shanghai(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Skipf("时区数据不可用: %v", err)
	}
	in := time.Date(2026, 3, 15, 13, 45, 30, 0, loc)

	got := StartOfDay(in)
	want := time.Date(2026, 3, 15, 0, 0, 0, 0, loc)

	if !got.Equal(want) {
		t.Fatalf("StartOfDay = %s, want %s", got, want)
	}
	// 反证：旧写法 Truncate(24h) 会落在 UTC 午夜，即当地 08:00，与期望零点不符
	if in.Truncate(24 * time.Hour).Equal(want) {
		t.Fatal("Truncate(24h) 不应等于本地零点，用例前提失效")
	}
}

// TestStartOfDay_UTC UTC 时区下 StartOfDay 与 Truncate 行为一致。
func TestStartOfDay_UTC(t *testing.T) {
	in := time.Date(2026, 3, 15, 23, 59, 59, 0, time.UTC)
	got := StartOfDay(in)
	want := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("StartOfDay(UTC) = %s, want %s", got, want)
	}
}

// TestStartOfMonth 返回当月一号零点。
func TestStartOfMonth(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	in := time.Date(2026, 10, 5, 18, 30, 0, 0, loc)
	got := StartOfMonth(in)
	want := time.Date(2026, 10, 1, 0, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("StartOfMonth = %s, want %s", got, want)
	}
}

// TestStartOfWeek 以周一为起点，周日归属上一周。
func TestStartOfWeek(t *testing.T) {
	// 2026-10-05 是周一
	monday := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	if got := StartOfWeek(monday); !got.Equal(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("周一的周起点应为当天零点, got %s", got)
	}

	// 2026-10-11 是周日，应回退到 10-05 周一
	sunday := time.Date(2026, 10, 11, 23, 0, 0, 0, time.UTC)
	if got := StartOfWeek(sunday); !got.Equal(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("周日的周起点应为上一个周一, got %s", got)
	}
}

// TestStartOfDay_PreservesLocation 结果时区应与入参一致。
func TestStartOfDay_PreservesLocation(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("时区数据不可用: %v", err)
	}
	in := time.Date(2026, 7, 4, 9, 0, 0, 0, loc)
	got := StartOfDay(in)
	if got.Location().String() != loc.String() {
		t.Fatalf("StartOfDay 应保留原时区, got %s", got.Location())
	}
}
