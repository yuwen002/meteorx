package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"
)

type TickerOption func(*tickerConfig)

type tickerConfig struct {
	runImmediately bool
	logger         *log.Logger
	panicHandler   func(name string, r any)
}

func WithRunImmediately() TickerOption {
	return func(c *tickerConfig) { c.runImmediately = true }
}

func WithLogger(l *log.Logger) TickerOption {
	return func(c *tickerConfig) { c.logger = l }
}

func WithPanicHandler(h func(name string, r any)) TickerOption {
	return func(c *tickerConfig) { c.panicHandler = h }
}

func Ticker(ctx context.Context, name string, interval time.Duration, fn func(ctx context.Context) error, opts ...TickerOption) {
	cfg := &tickerConfig{
		logger: log.Default(),
		panicHandler: func(name string, r any) {
			log.Printf("[scheduler] panic recovered in %s: %v", name, r)
		},
	}
	for _, o := range opts {
		o(cfg)
	}

	go runTicker(ctx, name, interval, fn, cfg)
	cfg.logger.Printf("[%s] 定时任务已启动，间隔 %s", name, interval)
}

func runTicker(ctx context.Context, name string, interval time.Duration, fn func(ctx context.Context) error, cfg *tickerConfig) {
	safeRun := func() {
		defer func() {
			if r := recover(); r != nil {
				cfg.panicHandler(name, r)
			}
		}()
		if err := fn(ctx); err != nil {
			cfg.logger.Printf("[%s] 执行失败: %v", name, err)
		}
	}

	if cfg.runImmediately {
		safeRun()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			cfg.logger.Printf("[%s] 收到退出信号，定时任务停止", name)
			return
		case <-ticker.C:
			safeRun()
		}
	}
}

func FormatDuration(d time.Duration) string {
	if d >= time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	if d >= time.Minute {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return d.String()
}