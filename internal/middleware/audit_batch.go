package middleware

import (
	"context"
	"sync"
	"time"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/service"
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
}

// NewAuditBatchProcessor 创建批量处理器
func NewAuditBatchProcessor(svc *service.AuditService, bufferSize int, flushInterval time.Duration) *AuditBatchProcessor {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}

	processor := &AuditBatchProcessor{
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

	// 批量写入（这里简化处理，逐个写入）
	// 生产环境可以实现真正的批量插入
	ctx := context.Background()
	for _, req := range logs {
		_, _ = p.svc.CreateLog(ctx, *req)
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

// InitAuditBatchProcessor 初始化全局批量处理器
func InitAuditBatchProcessor(svc *service.AuditService) {
	GlobalBatchProcessor = NewAuditBatchProcessor(svc, 100, 5*time.Second)
}

// StopAuditBatchProcessor 停止全局批量处理器
func StopAuditBatchProcessor() {
	if GlobalBatchProcessor != nil {
		GlobalBatchProcessor.Stop()
	}
}