// Package worker 提供导出任务的异步执行池。
// poller 周期性从数据库拉取 pending 任务推入队列，N 个 worker 消费队列，
// 按资源类型分派到对应 exporter，流式写出 CSV 文件并回写任务状态。
package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"meteorx/internal/modules/export/model"
	"meteorx/internal/modules/export/repository"
	"meteorx/internal/modules/export/service"
	"meteorx/internal/modules/export/service/exporter"
	"meteorx/pkg/logger"
)

// pollInterval poller 扫描 pending 任务的间隔
const pollInterval = 2 * time.Second

// pollBatchSize 单轮最多拉取的 pending 任务数
const pollBatchSize = 100

// Pool 导出任务工作池
type Pool struct {
	repo     repository.ExportRepository
	registry *exporter.Registry
	cfg      service.Config
	queue    chan string
}

// NewPool 创建工作池
func NewPool(repo repository.ExportRepository, registry *exporter.Registry, cfg service.Config) *Pool {
	return &Pool{
		repo:     repo,
		registry: registry,
		cfg:      cfg,
		queue:    make(chan string, 64),
	}
}

// Start 启动 poller 与 worker 协程，阻塞直到 ctx 取消。
func (p *Pool) Start(ctx context.Context) {
	if err := os.MkdirAll(p.cfg.Dir, 0o750); err != nil {
		logger.Errorf("[Export] 无法创建导出目录 %s: %v", p.cfg.Dir, err)
		return
	}
	workers := p.cfg.WorkerCount
	if workers <= 0 {
		workers = 3
	}
	for i := 0; i < workers; i++ {
		go p.worker(ctx)
	}
	logger.Infof("[Export] worker 池启动，worker 数=%d, 目录=%s", workers, p.cfg.Dir)
	p.poll(ctx)
}

// poll 周期性拉取 pending 任务并入队，ctx 取消时返回。
func (p *Pool) poll(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("[Export] worker 池已停止")
			return
		case <-ticker.C:
			jobs, err := p.repo.ListPending(ctx, pollBatchSize)
			if err != nil {
				logger.Errorf("[Export] 拉取待处理任务失败: %v", err)
				continue
			}
			for _, j := range jobs {
				select {
				case p.queue <- j.ID:
				case <-ctx.Done():
					return
				default:
					// 队列已满，本轮跳过，下一轮重试
				}
			}
		}
	}
}

// worker 消费队列直到 ctx 取消。
func (p *Pool) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-p.queue:
			p.processJob(ctx, id)
		}
	}
}

// processJob 处理单个导出任务：抢占 → 分派 exporter → 流式写 CSV → 回写状态。
func (p *Pool) processJob(ctx context.Context, id string) {
	job, err := p.repo.GetByID(ctx, id)
	if err != nil {
		if !errors.Is(err, repository.ErrExportJobNotFound) {
			logger.Errorf("[Export] 读取任务 %s 失败: %v", id, err)
		}
		return
	}
	// 条件抢占：仅 pending→running 成功者继续，避免多 worker/多副本重复执行
	ok, err := p.repo.MarkRunning(ctx, id, time.Now())
	if err != nil {
		logger.Errorf("[Export] 抢占任务 %s 失败: %v", id, err)
		return
	}
	if !ok {
		return
	}

	exp, found := p.registry.Get(job.Resource)
	if !found {
		p.markFailed(ctx, id, fmt.Sprintf("未注册的导出资源类型: %s", job.Resource))
		return
	}

	var filters map[string]any
	if job.FiltersJSON != "" {
		if err := json.Unmarshal([]byte(job.FiltersJSON), &filters); err != nil {
			p.markFailed(ctx, id, "筛选条件解析失败: "+err.Error())
			return
		}
	}

	filePath := filepath.Join(p.cfg.Dir, id+"."+model.FormatExt(job.Format))
	rows, err := p.runExport(ctx, exp, job, filePath, filters)
	if err != nil {
		_ = os.Remove(filePath) // 清理半成品文件
		// 完整错误只进服务端日志，落库的 error_msg 截断后回显，避免内部细节原样泄给前端
		p.markFailed(ctx, id, truncateErrMsg(err.Error()))
		return
	}
	if err := p.repo.MarkCompleted(ctx, id, filePath, rows, time.Now()); err != nil {
		logger.Errorf("[Export] 回写任务 %s 完成状态失败: %v", id, err)
	}
}

// runExport 打开文件、按格式构造 RowSink、调用 exporter 流式写出，返回数据行数。
// CSV 额外写入 UTF-8 BOM 以保证 Excel 打开中文不乱码；其余格式直写。
func (p *Pool) runExport(ctx context.Context, exp exporter.Exporter, job *model.ExportJob, filePath string, filters map[string]any) (int64, error) {
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o640)
	if err != nil {
		return 0, fmt.Errorf("创建导出文件失败: %w", err)
	}
	defer func() { _ = f.Close() }()

	var sink exporter.RowSink
	if job.Format == model.FormatCSV || job.Format == "" {
		// 写入 UTF-8 BOM，保证 Excel 打开中文不乱码
		if _, err := f.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
			return 0, fmt.Errorf("写入 BOM 失败: %w", err)
		}
		sink = exporter.NewCSVSink(f)
	} else {
		sink, err = exporter.NewSink(job.Format, f)
		if err != nil {
			return 0, err
		}
	}

	rows, err := exp.Stream(ctx, job.TenantID, filters, sink)
	if err != nil {
		return 0, err
	}
	if err := sink.Close(); err != nil {
		return 0, fmt.Errorf("收尾导出文件失败: %w", err)
	}
	if err := f.Sync(); err != nil {
		return 0, fmt.Errorf("同步文件失败: %w", err)
	}
	return rows, nil
}

// markFailed 回写失败状态并记录日志
func (p *Pool) markFailed(ctx context.Context, id, msg string) {
	if err := p.repo.MarkFailed(ctx, id, msg, time.Now()); err != nil {
		logger.Errorf("[Export] 回写任务 %s 失败状态出错: %v", id, err)
	}
	logger.Warnf("[Export] 任务 %s 导出失败: %s", id, msg)
}

// maxErrMsgLen error_msg 落库/回显的最大长度，防止将驱动层长错误（含路径/堆砌信息）原样展示。
const maxErrMsgLen = 200

// truncateErrMsg 截断错误摘要，保留前 maxErrMsgLen 个字符（按 rune 避免乱码）。
func truncateErrMsg(msg string) string {
	r := []rune(msg)
	if len(r) <= maxErrMsgLen {
		return msg
	}
	return string(r[:maxErrMsgLen]) + "..."
}
