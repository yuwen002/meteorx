package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Config 日志配置
type Config struct {
	Level      string // 日志级别: debug, info, warn, error
	Format     string // 输出格式: json, text
	Output     string // 输出目标: stdout, file, both
	FilePath   string // 日志文件路径（当 Output 为 file 或 both 时有效）
	MaxSize    int64  // 单个日志文件最大大小（字节）
	MaxBackups int    // 保留的旧日志文件数量
	MaxAge     int    // 保留天数

	// 异步日志配置
	Async      bool  // 是否启用异步日志
	BufferSize int   // 异步日志缓冲区大小（默认 1024）

	// 日志采样配置
	// Debug 日志采样率 (0.0-1.0)
	// 0.0 = 不记录任何 Debug 日志
	// 0.5 = 记录 50% 的 Debug 日志
	// 1.0 = 记录所有 Debug 日志
	SampleRate float64
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Level:      "info",
		Format:     "text",
		Output:     "stdout",
		FilePath:   "logs/meteorx.log",
		MaxSize:    100 * 1024 * 1024, // 100MB
		MaxBackups: 10,
		MaxAge:     30,
		Async:      false,
		BufferSize: 1024,
		SampleRate: 1.0, // 默认全部记录
	}
}

var (
	std          *slog.Logger
	cfg          Config
	closeOnce    sync.Once
)

// Init 初始化全局日志器
func Init(c ...Config) error {
	if len(c) > 0 {
		cfg = c[0]
	} else {
		cfg = DefaultConfig()
	}

	// 验证并修正配置
	if cfg.SampleRate < 0 || cfg.SampleRate > 1 {
		cfg.SampleRate = 1.0
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1024
	}
	if cfg.Level == "" {
		cfg.Level = "info"
	}

	level, err := parseLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("invalid log level %q: %w", cfg.Level, err)
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	writer := createWriter()

	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	default:
		handler = slog.NewTextHandler(writer, opts)
	}

	// 如果需要采样且级别为 Debug，包装采样处理器
	if cfg.SampleRate < 1.0 && level == slog.LevelDebug {
		handler = &samplingHandler{
			handler:    handler,
			sampleRate: cfg.SampleRate,
		}
	}

	std = slog.New(handler)
	slog.SetDefault(std)

	Info("logger initialized",
		"level", cfg.Level,
		"format", cfg.Format,
		"output", cfg.Output,
		"async", cfg.Async,
		"sample_rate", cfg.SampleRate,
	)

	return nil
}

// Close 关闭日志系统，刷新所有缓冲日志
func Close() {
	closeOnce.Do(func() {
		if asyncWriterInstance != nil {
			asyncWriterInstance.Close()
		}
		if rotWriter != nil {
			rotWriter.Close()
		}
	})
}

// IsAsync 返回当前是否启用了异步日志
func IsAsync() bool {
	return cfg.Async
}

// createWriter 创建日志输出 writer
func createWriter() io.Writer {
	var writer io.Writer

	switch cfg.Output {
	case "file":
		writer = createFileWriter()
	case "both":
		writer = io.MultiWriter(os.Stdout, createFileWriter())
	default:
		writer = os.Stdout
	}

	// 如果启用异步日志，包装为异步 writer
	if cfg.Async {
		asyncWriterInstance = newAsyncWriter(writer, cfg.BufferSize)
		return asyncWriterInstance
	}

	return writer
}

// ---------------------------------------------------------------------------
// 异步日志
// ---------------------------------------------------------------------------

var asyncWriterInstance *asyncWriter

// asyncWriter 异步日志 writer，通过缓冲 channel 实现非阻塞写入
type asyncWriter struct {
	ch         chan []byte
	underlying io.Writer
	wg         sync.WaitGroup
	closeCh    chan struct{}
}

func newAsyncWriter(w io.Writer, bufferSize int) *asyncWriter {
	aw := &asyncWriter{
		ch:         make(chan []byte, bufferSize),
		underlying: w,
		closeCh:    make(chan struct{}),
	}

	aw.wg.Add(1)
	go aw.loop()

	return aw
}

func (aw *asyncWriter) loop() {
	defer aw.wg.Done()

	for {
		select {
		case p := <-aw.ch:
			if _, err := aw.underlying.Write(p); err != nil {
				fmt.Fprintf(os.Stderr, "async log write error: %v\n", err)
			}
		case <-aw.closeCh:
			// 关闭前清空缓冲区中剩余的日志
			for {
				select {
				case p := <-aw.ch:
					if _, err := aw.underlying.Write(p); err != nil {
						fmt.Fprintf(os.Stderr, "async log flush error: %v\n", err)
					}
				default:
					return
				}
			}
		}
	}
}

func (aw *asyncWriter) Write(p []byte) (int, error) {
	// 复制数据，避免外部重用
	buf := make([]byte, len(p))
	copy(buf, p)

	select {
	case aw.ch <- buf:
	default:
		// 缓冲区满，丢弃日志，避免阻塞业务逻辑
		fmt.Fprintf(os.Stderr, "async log buffer full (%d), dropping log entry\n", len(aw.ch))
	}

	return len(p), nil
}

func (aw *asyncWriter) Close() {
	close(aw.closeCh)
	aw.wg.Wait()
}

// ---------------------------------------------------------------------------
// 日志轮转
// ---------------------------------------------------------------------------

// rotatingWriter 按文件大小自动轮转的日志 writer
type rotatingWriter struct {
	mu         sync.Mutex
	file       *os.File
	filePath   string
	maxSize    int64
	maxBackups int
	maxAge     int
	size       int64
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil && w.size+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	if w.file == nil {
		if err := w.openFile(); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	if err == nil {
		w.size += int64(n)
	}
	return n, err
}

func (w *rotatingWriter) openFile() error {
	dir := filepath.Dir(w.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	f, err := os.OpenFile(w.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	w.file = f
	info, err := f.Stat()
	if err == nil {
		w.size = info.Size()
	}
	return nil
}

func (w *rotatingWriter) rotate() error {
	if w.file != nil {
		w.file.Close()
		w.file = nil
	}

	// 重命名当前文件为备份
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.%s", w.filePath, timestamp)
	if err := os.Rename(w.filePath, backupPath); err != nil {
		return err
	}

	// 清理旧备份
	w.cleanOldBackups()

	// 打开新文件
	w.size = 0
	return w.openFile()
}

// cleanOldBackups 按修改时间排序，保留最新的 maxBackups 个备份文件
func (w *rotatingWriter) cleanOldBackups() {
	dir := filepath.Dir(w.filePath)
	base := filepath.Base(w.filePath)
	pattern := filepath.Join(dir, base+".*")

	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) <= w.maxBackups {
		return
	}

	// 收集每个文件的修改时间
	type backupFile struct {
		path    string
		modTime time.Time
	}

	backups := make([]backupFile, 0, len(matches))
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		backups = append(backups, backupFile{
			path:    match,
			modTime: info.ModTime(),
		})
	}

	// 按修改时间升序排列（最旧的在前）
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].modTime.Before(backups[j].modTime)
	})

	// 删除最旧的备份文件
	toDelete := len(backups) - w.maxBackups
	if toDelete > 0 {
		for i := 0; i < toDelete; i++ {
			if err := os.Remove(backups[i].path); err != nil {
				fmt.Fprintf(os.Stderr, "failed to remove old backup %s: %v\n", backups[i].path, err)
			}
		}
	}
}

func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

var rotWriter *rotatingWriter

// createFileWriter 创建文件 writer（带日志轮转）
func createFileWriter() io.Writer {
	rotWriter = &rotatingWriter{
		filePath:   cfg.FilePath,
		maxSize:    cfg.MaxSize,
		maxBackups: cfg.MaxBackups,
		maxAge:     cfg.MaxAge,
	}

	if err := rotWriter.openFile(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
		return os.Stdout
	}

	return rotWriter
}

// ---------------------------------------------------------------------------
// 日志采样
// ---------------------------------------------------------------------------

// samplingHandler 日志采样处理器，仅对 Debug 级别生效
// 使用计数采样策略：每 N 条记录 1 条（N = 1/sampleRate）
type samplingHandler struct {
	handler    slog.Handler
	sampleRate float64
	mu         sync.Mutex
	count      uint64
}

func (h *samplingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *samplingHandler) Handle(ctx context.Context, r slog.Record) error {
	// 仅对 Debug 级别进行采样
	if r.Level == slog.LevelDebug {
		h.mu.Lock()
		h.count++
		count := h.count
		h.mu.Unlock()

		// 基于计数的采样：每 1/sampleRate 条记录一条
		sampleInterval := uint64(1.0 / h.sampleRate)
		if count%sampleInterval != 0 {
			return nil // 跳过此条日志
		}
	}

	return h.handler.Handle(ctx, r)
}

func (h *samplingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &samplingHandler{
		handler:    h.handler.WithAttrs(attrs),
		sampleRate: h.sampleRate,
	}
}

func (h *samplingHandler) WithGroup(name string) slog.Handler {
	return &samplingHandler{
		handler:    h.handler.WithGroup(name),
		sampleRate: h.sampleRate,
	}
}

// ---------------------------------------------------------------------------
// 解析
// ---------------------------------------------------------------------------

// parseLevel 解析日志级别字符串
func parseLevel(level string) (slog.Level, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown level: %s", level)
	}
}

// ---------------------------------------------------------------------------
// 便捷日志函数
// ---------------------------------------------------------------------------

// Info 记录一条 info 级别日志
func Info(msg string, args ...any) {
	std.Info(msg, args...)
}

// Debug 记录一条 debug 级别日志
func Debug(msg string, args ...any) {
	std.Debug(msg, args...)
}

// Warn 记录一条 warn 级别日志
func Warn(msg string, args ...any) {
	std.Warn(msg, args...)
}

// Error 记录一条 error 级别日志
func Error(msg string, args ...any) {
	std.Error(msg, args...)
}

// Infof 格式化消息后记录 info 级别日志
func Infof(format string, args ...any) {
	std.Info(fmt.Sprintf(format, args...))
}

// Debugf 格式化消息后记录 debug 级别日志
func Debugf(format string, args ...any) {
	std.Debug(fmt.Sprintf(format, args...))
}

// Warnf 格式化消息后记录 warn 级别日志
func Warnf(format string, args ...any) {
	std.Warn(fmt.Sprintf(format, args...))
}

// Errorf 格式化消息后记录 error 级别日志
func Errorf(format string, args ...any) {
	std.Error(fmt.Sprintf(format, args...))
}

// Fatalf 记录致命错误并退出进程
func Fatalf(format string, args ...any) {
	std.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

// ---------------------------------------------------------------------------
// 上下文相关
// ---------------------------------------------------------------------------

// Ctx 创建一个带上下文信息的日志器（自动附加 request_id, user_id, tenant_id）
func Ctx(ctx context.Context) *slog.Logger {
	return std.With(
		slog.String("request_id", GetRequestID(ctx)),
		slog.String("user_id", GetUserID(ctx)),
		slog.String("tenant_id", GetTenantID(ctx)),
	)
}

// With 创建带额外属性的子日志器
func With(args ...any) *slog.Logger {
	return std.With(args...)
}

// 上下文键类型
type contextKey string

const (
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
	tenantIDKey  contextKey = "tenant_id"
	traceIDKey   contextKey = "trace_id"
)

// WithRequestID 将 request_id 存入 context
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// GetRequestID 从 context 获取 request_id
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithUserID 将 user_id 存入 context
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// GetUserID 从 context 获取 user_id
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}

// WithTenantID 将 tenant_id 存入 context
func WithTenantID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, tenantIDKey, id)
}

// GetTenantID 从 context 获取 tenant_id
func GetTenantID(ctx context.Context) string {
	if id, ok := ctx.Value(tenantIDKey).(string); ok {
		return id
	}
	return ""
}

// WithTraceID 将 trace_id 存入 context
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

// GetTraceID 从 context 获取 trace_id
func GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}
	return ""
}