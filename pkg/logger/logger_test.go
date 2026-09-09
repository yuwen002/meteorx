package logger

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试辅助：重置全局状态
func resetGlobals() {
	std = nil
	cfg = Config{}
	closeOnce = sync.Once{}
	asyncWriterInstance = nil
	rotWriter = nil
}

// ---------------------------------------------------------------------------
// Init 测试
// ---------------------------------------------------------------------------

func TestInit_DefaultConfig(t *testing.T) {
	resetGlobals()

	err := Init()
	require.NoError(t, err)
	require.NotNil(t, std)

	assert.Equal(t, "info", cfg.Level)
	assert.Equal(t, "text", cfg.Format)
	assert.Equal(t, "stdout", cfg.Output)
	assert.False(t, cfg.Async)
}

func TestInit_JSONFormat(t *testing.T) {
	resetGlobals()

	err := Init(Config{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	})
	require.NoError(t, err)
	assert.NotNil(t, std)
}

func TestInit_InvalidLevel(t *testing.T) {
	resetGlobals()

	err := Init(Config{
		Level: "invalid",
	})
	assert.Error(t, err)
}

func TestInit_InvalidSampleRate(t *testing.T) {
	resetGlobals()

	err := Init(Config{
		Level:      "debug",
		SampleRate: 2.0, // 无效值，应修正为 1.0
	})
	require.NoError(t, err)
	assert.Equal(t, 1.0, cfg.SampleRate)

	// 负值
	resetGlobals()
	err = Init(Config{
		Level:      "debug",
		SampleRate: -0.5,
	})
	require.NoError(t, err)
	assert.Equal(t, 1.0, cfg.SampleRate)
}

func TestInit_FileOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	resetGlobals()
	err := Init(Config{
		Level:    "info",
		Format:   "text",
		Output:   "file",
		FilePath: logPath,
	})
	require.NoError(t, err)

	Info("test file log")
	Close()

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test file log")
}

func TestInit_BothOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	resetGlobals()
	err := Init(Config{
		Level:    "info",
		Format:   "text",
		Output:   "both",
		FilePath: logPath,
	})
	require.NoError(t, err)

	Info("test both output")
	Close()

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test both output")
}

// ---------------------------------------------------------------------------
// 日志级别测试
// ---------------------------------------------------------------------------

func TestLogLevels(t *testing.T) {
	resetGlobals()
	err := Init(Config{Level: "debug"})
	require.NoError(t, err)

	// 这些函数应该不会 panic
	Info("info message", "key", "value")
	Debug("debug message", "count", 1)
	Warn("warn message", "reason", "test")
	Error("error message", "err", "something")
	Infof("formatted %s", "info")
	Debugf("formatted %s", "debug")
	Warnf("formatted %s", "warn")
	Errorf("formatted %s", "error")
}

// ---------------------------------------------------------------------------
// 上下文测试
// ---------------------------------------------------------------------------

func TestContextFunctions(t *testing.T) {
	ctx := context.Background()

	// WithRequestID / GetRequestID
	ctx = WithRequestID(ctx, "req-123")
	assert.Equal(t, "req-123", GetRequestID(ctx))
	assert.Equal(t, "", GetRequestID(context.Background()))

	// WithUserID / GetUserID
	ctx = WithUserID(ctx, "user-456")
	assert.Equal(t, "user-456", GetUserID(ctx))
	assert.Equal(t, "", GetUserID(context.Background()))

	// WithTenantID / GetTenantID
	ctx = WithTenantID(ctx, "tenant-789")
	assert.Equal(t, "tenant-789", GetTenantID(ctx))
	assert.Equal(t, "", GetTenantID(context.Background()))

	// WithTraceID / GetTraceID
	ctx = WithTraceID(ctx, "trace-abc")
	assert.Equal(t, "trace-abc", GetTraceID(ctx))
	assert.Equal(t, "", GetTraceID(context.Background()))
}

func TestCtxLogger(t *testing.T) {
	resetGlobals()
	err := Init(Config{Level: "debug"})
	require.NoError(t, err)

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-001")
	ctx = WithUserID(ctx, "user-001")
	ctx = WithTenantID(ctx, "tenant-001")

	// Ctx 应该返回一个带上下文的日志器，不会 panic
	logger := Ctx(ctx)
	require.NotNil(t, logger)
	logger.Info("context-aware log")
}

// ---------------------------------------------------------------------------
// 采样测试
// ---------------------------------------------------------------------------

func TestSamplingHandler(t *testing.T) {
	// 使用缓冲区捕获日志输出
	var buf bytes.Buffer

	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	samplingHandler := &samplingHandler{
		handler:    handler,
		sampleRate: 0.5, // 50% 采样率
	}

	logger := slog.New(samplingHandler)

	// 写入 100 条 Debug 日志，统计实际写入数量
	written := 0
	for i := 0; i < 100; i++ {
		before := buf.Len()
		logger.Debug("sampled log", "index", i)
		if buf.Len() > before {
			written++
		}
	}

	// 50% 采样率下，100 条日志大约写入 50 条左右（允许一定偏差）
	t.Logf("Sampled: wrote %d out of 100 debug logs (rate: 0.5)", written)
	assert.Greater(t, written, 0)
	assert.Less(t, written, 100)
}

func TestSamplingHandler_InfoNotSampled(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	samplingHandler := &samplingHandler{
		handler:    handler,
		sampleRate: 0.1, // 10% 采样率
	}

	logger := slog.New(samplingHandler)

	// Info 级别的日志不应该被采样
	for i := 0; i < 10; i++ {
		logger.Info("info log", "index", i)
	}

	// 应该所有 Info 日志都被记录
	output := buf.String()
	assert.Equal(t, 10, strings.Count(output, "info log"))
}

// ---------------------------------------------------------------------------
// 异步日志测试
// ---------------------------------------------------------------------------

func TestAsyncWriter(t *testing.T) {
	var buf bytes.Buffer

	aw := newAsyncWriter(&buf, 100)

	// 写入多条日志
	for i := 0; i < 10; i++ {
		n, err := aw.Write([]byte("test message\n"))
		assert.NoError(t, err)
		assert.Equal(t, 13, n)
	}

	// 等待异步写入完成
	aw.Close()

	// 应该收到所有日志
	assert.Equal(t, 10, strings.Count(buf.String(), "test message"))
}

func TestAsyncWriter_BufferFull(t *testing.T) {
	var buf bytes.Buffer

	// 使用小缓冲区
	aw := newAsyncWriter(&buf, 2)

	// 写入大量数据，缓冲区满时不应阻塞
	for i := 0; i < 100; i++ {
		aw.Write([]byte("x\n"))
	}

	// 等待写入完成
	aw.Close()

	// 至少应该写入了一些（缓冲区满时丢弃，但不会全部丢失）
	output := buf.String()
	t.Logf("Async writer wrote %d lines out of 100 (buffer size: 2)", strings.Count(output, "x\n"))
	assert.Greater(t, strings.Count(output, "x\n"), 0)
}

// ---------------------------------------------------------------------------
// 日志轮转测试
// ---------------------------------------------------------------------------

func TestRotatingWriter_Rotate(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "rotate.log")

	rw := &rotatingWriter{
		filePath:   logPath,
		maxSize:    50, // 很小的文件大小，便于触发轮转
		maxBackups: 3,
	}

	err := rw.openFile()
	require.NoError(t, err)

	// 写入超过 maxSize 的数据以触发轮转
	line := []byte("this is a test log line that will be used to trigger rotation\n")
	for i := 0; i < 20; i++ {
		_, err := rw.Write(line)
		require.NoError(t, err)
	}

	rw.Close()

	// 验证备份文件存在
	files, err := filepath.Glob(logPath + ".*")
	require.NoError(t, err)
	t.Logf("Backup files: %d, files: %v", len(files), files)

	// 应该有备份文件
	assert.GreaterOrEqual(t, len(files), 1)

	// 验证当前文件存在
	_, err = os.Stat(logPath)
	assert.NoError(t, err)
}

func TestRotatingWriter_CleanOldBackups(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "clean.log")

	// 创建一些旧的备份文件
	for i := 0; i < 5; i++ {
		backupPath := fmt.Sprintf("%s.20250101-%06d", logPath, i)
		os.WriteFile(backupPath, []byte("old backup"), 0644)
		// 设置不同的修改时间
		os.Chtimes(backupPath, time.Now(), time.Now().Add(-time.Duration(5-i)*time.Hour))
	}

	rw := &rotatingWriter{
		filePath:   logPath,
		maxSize:    1000,
		maxBackups: 3, // 只保留 3 个备份
	}

	// 写入并触发轮转（会调用 cleanOldBackups）
	rw.openFile()
	rw.Write([]byte("test data\n"))
	rw.rotate()
	rw.Close()

	// 验证只保留了 3 个备份 + 当前文件
	matches, _ := filepath.Glob(logPath + ".*")
	assert.LessOrEqual(t, len(matches), 3)
}

// ---------------------------------------------------------------------------
// Close 测试
// ---------------------------------------------------------------------------

func TestClose_Multiple(t *testing.T) {
	resetGlobals()
	err := Init(Config{
		Level:  "info",
		Output: "stdout",
	})
	require.NoError(t, err)

	// 多次调用 Close 不应 panic
	Close()
	Close()
	Close()
}

// ---------------------------------------------------------------------------
// IsAsync 测试
// ---------------------------------------------------------------------------

func TestIsAsync(t *testing.T) {
	resetGlobals()
	err := Init(Config{Level: "info"})
	require.NoError(t, err)
	assert.False(t, IsAsync())

	Close()

	resetGlobals()
	err = Init(Config{
		Level: "info",
		Async: true,
	})
	require.NoError(t, err)
	assert.True(t, IsAsync())

	Close()
}

// ---------------------------------------------------------------------------
// 并发安全测试
// ---------------------------------------------------------------------------

func TestConcurrentLogging(t *testing.T) {
	resetGlobals()
	err := Init(Config{Level: "debug"})
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				Info("concurrent log", "goroutine", id, "index", j)
				Debug("debug log", "goroutine", id, "index", j)
			}
		}(i)
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// 边界情况测试
// ---------------------------------------------------------------------------

func TestEmptyConfig(t *testing.T) {
	resetGlobals()
	err := Init(Config{})
	require.NoError(t, err)
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		want     slog.Level
		wantErr  bool
	}{
		{"debug", slog.LevelDebug, false},
		{"info", slog.LevelInfo, false},
		{"warn", slog.LevelWarn, false},
		{"warning", slog.LevelWarn, false},
		{"error", slog.LevelError, false},
		{"unknown", slog.LevelInfo, true},
		{"", slog.LevelInfo, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseLevel(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestWith(t *testing.T) {
	resetGlobals()
	err := Init(Config{Level: "debug"})
	require.NoError(t, err)

	// With 创建带额外属性的子日志器
	logger := With("component", "test")
	require.NotNil(t, logger)
	logger.Info("test with attrs")
}