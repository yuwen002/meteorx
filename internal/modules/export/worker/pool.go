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
	"meteorx/pkg/uuid"
)

// pollInterval poller 扫描 pending 任务的间隔
const pollInterval = 2 * time.Second

// pollBatchSize 单轮最多拉取的 pending 任务数
const pollBatchSize = 100

// terminalWriteTimeout 终态回写（completed/failed）所用独立上下文的超时。
// 该超时与入参 ctx 解耦：优雅关闭时 ctx 已被取消，若沿用则回写 SQL 直接失败，任务永久停在 running。
const terminalWriteTimeout = 5 * time.Second

const (
	// exportLeaseTTL 执行租约时长。优雅关闭已由 detachedWriteCtx 保证回写终态，本租约防的是
	// kill -9 / OOM / 容器驱逐这类无任何收尾机会的死法：当时无人写终态，任务会永久停在
	// running（poll 只拉 pending 不会重驱，还长期占用租户队列额度），租约到期后由回收扫描重驱。
	exportLeaseTTL = 60 * time.Second
	// heartbeatInterval 续租间隔，取 TTL/3：留出两次失败容错（GC 停顿、DB 抖动）而不被误判为死。
	heartbeatInterval = exportLeaseTTL / 3
	// maxExportAttempts 单个任务最多被执行次数：若某导出本身就会把进程撞死（如数据量导致 OOM），
	// 无限重驱会形成崩溃循环、反复拖垮整个服务，达上限后直接置失败。
	maxExportAttempts = 3
	// leaseWriteTimeout 单次续租写库超时，避免 DB 无响应时 goroutine 久留。
	leaseWriteTimeout = 5 * time.Second
)

// detachedWriteCtx 派生一个不随父 ctx 取消、但带超时的上下文，用于抢占后的终态回写。
func detachedWriteCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), terminalWriteTimeout)
}

// Pool 导出任务工作池
type Pool struct {
	repo     repository.ExportRepository
	registry *exporter.Registry
	cfg      service.Config
	queue    chan string

	// 以下三个时长/次数参数默认取常量，列为字段便于测试缩短（否则需真等 60s 租约）
	pollEvery     time.Duration
	leaseTTL      time.Duration
	heartbeatRate time.Duration
	maxAttempts   int
}

// NewPool 创建工作池
func NewPool(repo repository.ExportRepository, registry *exporter.Registry, cfg service.Config) *Pool {
	return &Pool{
		repo:     repo,
		registry: registry,
		cfg:      cfg,
		queue:    make(chan string, 64),

		pollEvery:     pollInterval,
		leaseTTL:      exportLeaseTTL,
		heartbeatRate: heartbeatInterval,
		maxAttempts:   maxExportAttempts,
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
	// 启动即回收一次：上一副本被强杀时残留的 running 任务不必等首个 ticker 周期
	p.reclaim(ctx)
	for i := 0; i < workers; i++ {
		go p.worker(ctx)
	}
	logger.Infof("[Export] worker 池启动，worker 数=%d, 目录=%s, 租约=%s", workers, p.cfg.Dir, p.leaseTTL)
	p.poll(ctx)
}

// reclaim 回收租约过期的 running 任务（执行进程被强杀/OOM，没有任何收尾机会）。
// 多副本安全：只动租约已过期的行，持有效租约的是其他存活 worker 正在跑的任务。
func (p *Pool) reclaim(ctx context.Context) {
	requeued, aborted, err := p.repo.ReclaimExpiredLeases(ctx, time.Now(), p.leaseTTL, p.maxAttempts)
	if err != nil {
		logger.Errorf("[Export] 回收失联导出任务失败: %v", err)
		return
	}
	if requeued > 0 || aborted > 0 {
		logger.Warnf("[Export] 回收失联导出任务：重新排队 %d 个，超执行次数上限置失败 %d 个", requeued, aborted)
	}
}

// poll 周期性拉取 pending 任务并入队，ctx 取消时返回。
func (p *Pool) poll(ctx context.Context) {
	ticker := time.NewTicker(p.pollEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("[Export] worker 池已停止")
			return
		case <-ticker.C:
			// 先回收再拉取：回队的任务本轮就能重新被调度，不必额外等一个周期
			p.reclaim(ctx)
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
	// 条件抢占：仅 pending→running 成功者继续，避免多 worker/多副本重复执行；
	// 同时写入执行租约与归属令牌，使本进程硬死亡时该任务后续可被回收重驱。
	claimedAt := time.Now()
	token := uuid.Generate()
	ok, err := p.repo.MarkRunning(ctx, id, repository.Claim{
		StartedAt:      claimedAt,
		LeaseExpiresAt: claimedAt.Add(p.leaseTTL),
		Token:          token,
	})
	if err != nil {
		logger.Errorf("[Export] 抢占任务 %s 失败: %v", id, err)
		return
	}
	if !ok {
		return
	}
	// 执行期间持续续租，向其他副本证明“我还活着”；processJob 返回即停止。
	hbCtx, stopHB := context.WithCancel(ctx)
	defer stopHB()
	go p.heartbeat(hbCtx, id, token)

	exp, found := p.registry.Get(job.Resource)
	if !found {
		p.markFailed(ctx, id, token, fmt.Sprintf("未注册的导出资源类型: %s", job.Resource))
		return
	}

	var filters map[string]any
	if job.FiltersJSON != "" {
		if err := json.Unmarshal([]byte(job.FiltersJSON), &filters); err != nil {
			p.markFailed(ctx, id, token, "筛选条件解析失败: "+err.Error())
			return
		}
	}

	filePath := filepath.Join(p.cfg.Dir, id+"."+model.FormatExt(job.Format))
	rows, err := p.runExport(ctx, exp, job, filePath, filters)
	if err != nil {
		_ = os.Remove(filePath) // 清理半成品文件
		// 完整错误只进服务端日志，落库的 error_msg 截断后回显，避免内部细节原样泄给前端
		p.markFailed(ctx, id, token, userSafeExportError(err))
		return
	}
	// 终态回写用独立上下文：此时 ctx 可能因优雅关闭已取消，若失败会使任务永久停在 running
	wctx, wcancel := detachedWriteCtx(ctx)
	defer wcancel()
	if err := p.repo.MarkCompleted(wctx, id, token, filePath, rows, time.Now()); err != nil {
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

// heartbeat 周期性续租，直到任务结束（stopHB）或上下文取消。
// 不做续租则长导出会被误判为已死而被其他副本抢去重跑；反之进程真死时心跳自然停，租约到期即触发回收。
func (p *Pool) heartbeat(ctx context.Context, id, token string) {
	ticker := time.NewTicker(p.heartbeatRate)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wctx, cancel := context.WithTimeout(ctx, leaseWriteTimeout)
			ok, err := p.repo.Heartbeat(wctx, id, token, time.Now().Add(p.leaseTTL))
			cancel()
			if err != nil {
				logger.Warnf("[Export] 任务 %s 续租失败（租约将于 %s 后到期）: %v", id, p.leaseTTL, err)
				continue
			}
			if !ok {
				// 任务已不在 running 或令牌已换（本执行成僵尸）：继续按旧令牌续租只会空转
				return
			}
		}
	}
}

// markFailed 回写失败状态并记录日志。
// 使用脱离取消的独立上下文：调用时常发生在 ctx 已被取消的异常路径上（如优雅关闭），
// 若沿用原 ctx 会使回写失败，任务永久停在 running 并持续占用租户队列额度。
func (p *Pool) markFailed(ctx context.Context, id, token, msg string) {
	wctx, wcancel := detachedWriteCtx(ctx)
	defer wcancel()
	if err := p.repo.MarkFailed(wctx, id, token, msg, time.Now()); err != nil {
		logger.Errorf("[Export] 回写任务 %s 失败状态出错: %v", id, err)
	}
	logger.Warnf("[Export] 任务 %s 导出失败: %s", id, msg)
}

// userSafeExportError 将中断类错误映为用户可读文案，其余仍走截断。
// 服务重启导致的 context.Canceled/DeadlineExceeded 若直接回显，给用户看到的是“context canceled”这类无意义内部错误。
func userSafeExportError(err error) string {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "导出被中断（服务重启或超时），请重新发起"
	}
	return truncateErrMsg(err.Error())
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
