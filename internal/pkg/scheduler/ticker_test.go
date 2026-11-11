package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestTicker_Basic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var count int32
	Ticker(ctx, "test-basic", 50*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	time.Sleep(120 * time.Millisecond)
	cancel()

	c := atomic.LoadInt32(&count)
	if c < 1 {
		t.Fatalf("expected at least 1 tick, got %d", c)
	}
}

func TestTicker_RunImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var count int32
	Ticker(ctx, "test-imm", 10*time.Second, func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	}, WithRunImmediately())

	time.Sleep(20 * time.Millisecond)
	cancel()

	c := atomic.LoadInt32(&count)
	if c != 1 {
		t.Fatalf("expected exactly 1 immediate run, got %d", c)
	}
}

func TestTicker_PanicRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var count int32
	var recovered int32
	Ticker(ctx, "test-panic", 30*time.Millisecond, func(ctx context.Context) error {
		n := atomic.AddInt32(&count, 1)
		if n == 1 {
			panic("boom")
		}
		return nil
	}, WithPanicHandler(func(name string, r any) {
		atomic.AddInt32(&recovered, 1)
	}))

	time.Sleep(100 * time.Millisecond)
	cancel()

	if atomic.LoadInt32(&recovered) != 1 {
		t.Fatalf("expected panic handler called once")
	}
	c := atomic.LoadInt32(&count)
	if c < 2 {
		t.Fatalf("expected ticker to continue after panic, got %d runs", c)
	}
}

func TestTicker_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var count int32
	Ticker(ctx, "test-cancel", 10*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	time.Sleep(25 * time.Millisecond)
	cancel()
	time.Sleep(25 * time.Millisecond)

	after := atomic.LoadInt32(&count)
	time.Sleep(30 * time.Millisecond)
	final := atomic.LoadInt32(&count)

	if after != final {
		t.Fatalf("ticker continued after cancel: before=%d after=%d", after, final)
	}
}

func TestTicker_ErrorLogging(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var count int32
	Ticker(ctx, "test-err", 20*time.Millisecond, func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return errors.New("test error")
	})

	time.Sleep(50 * time.Millisecond)
	cancel()

	c := atomic.LoadInt32(&count)
	if c < 1 {
		t.Fatalf("expected at least 1 tick, got %d", c)
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{90 * time.Minute, "1.5h"},
		{90 * time.Second, "1.5m"},
		{30 * time.Second, "30s"},
	}
	for _, tc := range cases {
		got := FormatDuration(tc.in)
		if got != tc.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}