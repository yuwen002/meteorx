package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// stubHorizon 配置中心提醒窗口桩：按预设值/错误驱动运行时覆盖与回退断言
type stubHorizon struct {
	d   time.Duration
	err error
}

func (s stubHorizon) GetDuration(context.Context, string) (time.Duration, error) { return s.d, s.err }

// TestReminderJob_EffectiveHorizon 提醒窗口回退链：配置中心 → 启动静态值
func TestReminderJob_EffectiveHorizon(t *testing.T) {
	ctx := context.Background()
	// db 传 nil：effectiveHorizon 不触达服务层，仅校验窗口解析逻辑
	job := NewTaskReminderJob(nil, ReminderJobOptions{Horizon: 2 * time.Hour})

	// 未注入 provider：使用启动时注入的静态窗口
	require.Equal(t, 2*time.Hour, job.effectiveHorizon(ctx))

	// 注入正值：运行时覆盖优先于静态配置，免重启生效
	job.SetHorizonProvider(stubHorizon{d: 6 * time.Hour})
	require.Equal(t, 6*time.Hour, job.effectiveHorizon(ctx), "配置中心正值应覆盖静态窗口")

	// 非正值：回落静态
	job.SetHorizonProvider(stubHorizon{d: 0})
	require.Equal(t, 2*time.Hour, job.effectiveHorizon(ctx), "非正值应回落静态配置")

	// 读取失败：fail-open 回落静态，不得因配置中心故障中断扫描
	job.SetHorizonProvider(stubHorizon{err: errors.New("db down")})
	require.Equal(t, 2*time.Hour, job.effectiveHorizon(ctx), "读取失败必须回落静态配置")
}

// TestReminderJob_HorizonDefaultZero 零值选项回落内置默认窗口
func TestReminderJob_HorizonDefaultZero(t *testing.T) {
	job := NewTaskReminderJob(nil, ReminderJobOptions{})
	require.Equal(t, 24*time.Hour, job.effectiveHorizon(context.Background()), "零值应回落内置默认 24h")
}
