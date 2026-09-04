package logger

import (
	"fmt"
	"log"
	"os"
)

type Logger struct {
	*log.Logger
}

// 包级默认 Logger，供全项目统一调用，避免各处自行实例化 log/fmt
var std = NewLogger("")

func NewLogger(prefix string) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, prefix, log.LstdFlags|log.Lshortfile),
	}
}

func (l *Logger) Info(msg string) {
	l.Println("INFO:", msg)
}

func (l *Logger) Error(msg string) {
	l.Println("ERROR:", msg)
}

func (l *Logger) Debug(msg string) {
	l.Println("DEBUG:", msg)
}

func (l *Logger) Warn(msg string) {
	l.Println("WARN:", msg)
}

func (l *Logger) Infof(format string, args ...any) {
	l.Info(fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...any) {
	l.Error(fmt.Sprintf(format, args...))
}

func (l *Logger) Debugf(format string, args ...any) {
	l.Debug(fmt.Sprintf(format, args...))
}

func (l *Logger) Warnf(format string, args ...any) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Fatalf 记录致命错误并退出进程（仅用于启动阶段的不可恢复错误）
func Fatalf(format string, args ...any) {
	std.Logger.Fatalf(format, args...)
}

func (l *Logger) Fatalf(format string, args ...any) {
	l.Logger.Fatalf(format, args...)
}

// 包级函数：项目统一通过 logger.Info/Error/Warn/... 记录日志

func Info(msg string) {
	std.Info(msg)
}

func Error(msg string) {
	std.Error(msg)
}

func Debug(msg string) {
	std.Debug(msg)
}

func Warn(msg string) {
	std.Warn(msg)
}

func Infof(format string, args ...any) {
	std.Infof(format, args...)
}

func Errorf(format string, args ...any) {
	std.Errorf(format, args...)
}

func Debugf(format string, args ...any) {
	std.Debugf(format, args...)
}

func Warnf(format string, args ...any) {
	std.Warnf(format, args...)
}
