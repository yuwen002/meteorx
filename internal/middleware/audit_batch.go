package middleware

import (
	"context"
	"log"
	"sync"
	"time"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/service"
	"meteorx/pkg/iplocation"
)

// AuditBatchProcessor 批量审计日志处理器
type AuditBatchProcessor struct {
	svc           *service.AuditService
	buffer        []*dto.CreateAuditLogReq
	bufferSize    int
	flushInterval time.Duration
	mutex         sync.Mutex
	flushing      bool
	ticker        *time.Ticker
	stopCh        chan struct{}
	wg            sync.WaitGroup
	ctx           context.Context
}

// auditFlushTimeout 单次批量落库写入的最大时长，防止数据库无响应时阻塞关闭流程。
const auditFlushTimeout = 10 * time.Second

// NewAuditBatchProcessor 创建批量处理器
func NewAuditBatchProcessor(ctx context.Context, svc *service.AuditService, bufferSize int, flushInterval time.Duration) *AuditBatchProcessor {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}

	processor := &AuditBatchProcessor{
		ctx:           ctx,
		svc:           svc,
		buffer:        make([]*dto.CreateAuditLogReq, 0, bufferSize),
		bufferSize:    bufferSize,
		flushInterval: flushInterval,
		ticker:        time.NewTicker(flushInterval),
		stopCh:        make(chan struct{}),
	}

	// 启动定时刷新协程
	processor.wg.Add(1)
	go processor.flushLoop()

	return processor
}

// Add 添加日志到缓冲区
// 当累积到 bufferSize 时触发异步 flush。
// 为避免多个 goroutine 并发 flush，加 flushing 标记做协调。
func (p *AuditBatchProcessor) Add(req *dto.CreateAuditLogReq) {
	p.mutex.Lock()
	p.buffer = append(p.buffer, req)
	shouldFlush := len(p.buffer) >= p.bufferSize && !p.flushing
	if shouldFlush {
		p.flushing = true
	}
	p.mutex.Unlock()

	if shouldFlush {
		go p.flush()
	}
}

// flushLoop 定时刷新循环
func (p *AuditBatchProcessor) flushLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ticker.C:
			p.flush()
		case <-p.stopCh:
			p.flush()
			return
		}
	}
}

// flush 刷新缓冲区到数据库
func (p *AuditBatchProcessor) flush() {
	p.mutex.Lock()
	if len(p.buffer) == 0 {
		p.mutex.Unlock()
		return
	}

	// 复制缓冲区并清空
	logs := make([]*dto.CreateAuditLogReq, len(p.buffer))
	copy(logs, p.buffer)
	p.buffer = p.buffer[:0]
	p.flushing = false
	p.mutex.Unlock()

	// 使用真正的批量插入（性能优化）
	reqs := make([]dto.CreateAuditLogReq, len(logs))
	for i, log := range logs {
		reqs[i] = *log
	}

	// 使用脱离取消的独立上下文 + 超时：bootstrap 的关闭顺序是先 cancel() 再
	// StopAuditBatchProcessor()，若沿用已被取消的 app ctx，收尾刷新会整批写入失败，
	// 缓冲区中最多 bufferSize 条审计日志会被静默丢弃；超时则避免极端下阻塞关闭流程。
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(p.ctx), auditFlushTimeout)
	defer cancel()

	if err := p.svc.BatchCreateLogs(writeCtx, reqs); err != nil {
		// 记录错误但不阻塞
		log.Printf("[audit] batch create logs failed: %v", err)
	}
}

// Stop 停止处理器
func (p *AuditBatchProcessor) Stop() {
	close(p.stopCh)
	p.ticker.Stop()
	p.wg.Wait()
}

// GlobalBatchProcessor 全局批量处理器实例
var GlobalBatchProcessor *AuditBatchProcessor

// globalAuditSvc 全局审计服务实例（供 RecordAuditLog 回退使用）
var globalAuditSvc *service.AuditService

// globalIPLocator 全局 IP 地理位置解析器（供 RecordAuditLog 使用）
var globalIPLocator iplocation.IPLocator

// InitAuditBatchProcessor 初始化全局批量处理器
func InitAuditBatchProcessor(ctx context.Context, svc *service.AuditService, ipLocator iplocation.IPLocator) {
	GlobalBatchProcessor = NewAuditBatchProcessor(ctx, svc, 100, 5*time.Second)
	globalAuditSvc = svc
	globalIPLocator = ipLocator
}

// StopAuditBatchProcessor 停止全局批量处理器
func StopAuditBatchProcessor() {
	if GlobalBatchProcessor != nil {
		GlobalBatchProcessor.Stop()
	}
}

// RecordAuditLog 全局审计日志记录函数，供非中间件场景手动记录（如登录/登出）。
// 如果 req.IPLocation 为空，则尝试用全局 IP 定位器解析。
// 优先使用批量处理器，回退到直接写入。
func RecordAuditLog(ctx context.Context, req *dto.CreateAuditLogReq) {
	// 测试捕获钩子：非 nil 时直接交给它，不触碰批量/服务写入。
	// 生产代码中 testAuditCapture 恒为 nil。
	if testAuditCapture != nil {
		testAuditCapture(req)
		return
	}

	if req.IPLocation == "" && globalIPLocator != nil && req.ClientIP != "" {
		if loc, err := globalIPLocator.Locate(ctx, req.ClientIP); err == nil && loc != nil {
			req.IPLocation = loc.FullText
		}
	}

	if GlobalBatchProcessor != nil {
		GlobalBatchProcessor.Add(req)
	} else if globalAuditSvc != nil {
		// 无批量处理器时直接写入
		_, _ = globalAuditSvc.CreateLog(ctx, *req)
	}
}

// testAuditCapture 仅供测试注入的审计记录捕获钩子；非 nil 时 RecordAuditLog 将记录直接交给它。
var testAuditCapture func(req *dto.CreateAuditLogReq)

// SetAuditCaptureForTest 注入审计捕获钩子，返回还原函数。仅应在测试中使用。
func SetAuditCaptureForTest(capture func(req *dto.CreateAuditLogReq)) func() {
	prev := testAuditCapture
	testAuditCapture = capture
	return func() { testAuditCapture = prev }
}
