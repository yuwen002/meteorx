package audit

import (
	"context"
	"time"

	"meteorx/internal/config"
	"meteorx/internal/modules/audit/repository"
	"meteorx/internal/modules/audit/service"
	"meteorx/pkg/idgen"
	"meteorx/pkg/logger"

	"gorm.io/gorm"
)

// IntegrityAnchorJob 审计完整性锚点链定时任务。
// 因审计写入走异步批量 + 多副本并发 flush，严格逐行线性链会强制写串行化、不可行；
// 故按周期把「窗口内全部行摘要」折叠为一个 WindowDigest，并以 PrevAnchorHash→AnchorHash 链向上一条锚点，
// 形成 append-only 时间线：窗口内增删改/重序破坏 WindowDigest，断链破坏后续所有 AnchorHash。
// 未启用完整性（无 pepper）时整卡关闭，不生成锚点。
type IntegrityAnchorJob struct {
	guard *service.IntegrityGuard
	irepo repository.IntegrityRepository
	grace time.Duration
}

// NewIntegrityAnchorJob 创建锚点链任务实例。guard 须与写入侧共用同一实例（摘要/折叠同一 pepper）。
// 封口安全边界取 guard.AnchorGrace()，与 rehash 全量重建共用同一口径。
func NewIntegrityAnchorJob(db *gorm.DB, cfg config.Config, guard *service.IntegrityGuard) *IntegrityAnchorJob {
	return &IntegrityAnchorJob{
		guard: guard,
		irepo: repository.NewIntegrityRepository(db),
		grace: guard.AnchorGrace(),
	}
}

// Start 启动锚点链生成任务（后台 goroutine），interval 为封口周期。
// 未启用完整性时不启动，零后台开销。
func (j *IntegrityAnchorJob) Start(ctx context.Context, interval time.Duration) {
	if !j.guard.Enabled() {
		logger.Info("[IntegrityAnchorJob] 审计完整性校验未启用，锚点链任务已跳过")
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := j.runOnce(ctx); err != nil {
					logger.Errorf("[IntegrityAnchorJob] 生成完整性锚点失败: %v", err)
				}
			}
		}
	}()
	logger.Infof("[IntegrityAnchorJob] 审计完整性锚点链任务已启动，封口间隔 %s，安全边界 %s", interval, j.grace)
}

// runOnce 生成一条锚点：取链尾窗口右端为下界、now-grace 为上界折叠窗口行摘要并链式落库。
// 无新行（空窗口）时不封口，留待后续窗口累积，避免周期性空锚点噪声。
// 与密钥轮换 rehash 共用 integrityAnchorMu，避免 rehash 重建期间被并发追加锚点。
func (j *IntegrityAnchorJob) runOnce(ctx context.Context) error {
	if !j.guard.Enabled() {
		return nil
	}
	integrityAnchorMu.Lock()
	defer integrityAnchorMu.Unlock()
	latest, err := j.irepo.LatestAnchor(ctx)
	if err != nil {
		return err
	}
	windowStart := time.Time{}
	prevAnchorHash := ""
	if latest != nil {
		windowStart = latest.WindowEnd
		prevAnchorHash = latest.AnchorHash
	}

	windowEnd := time.Now().Add(-j.grace)
	// 创世窗口尚无下界，windowStart 为零值；非创世须确保窗口向前推进
	if !windowStart.IsZero() && !windowEnd.After(windowStart) {
		return nil
	}

	entries, err := j.irepo.ListRecordHashesInRange(ctx, windowStart, windowEnd)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	// 创世窗口以最早行的 created_at 作为落库下界（避免写入零值时间，且与校验重放口径一致）
	if windowStart.IsZero() {
		windowStart = entries[0].CreatedAt
	}

	hashes := make([]string, len(entries))
	for i, e := range entries {
		hashes[i] = e.RecordHash
	}
	windowDigest := j.guard.FoldRecordHashes(hashes)
	anchorHash := j.guard.AnchorHashOf(prevAnchorHash, windowDigest, int64(len(entries)), windowStart, windowEnd)

	anchor := &repository.IntegrityAnchor{
		ID:             idgen.New(),
		WindowStart:    windowStart,
		WindowEnd:      windowEnd,
		RowCount:       int64(len(entries)),
		WindowDigest:   windowDigest,
		AnchorHash:     anchorHash,
		PrevAnchorHash: prevAnchorHash,
	}
	if err := j.irepo.CreateAnchor(ctx, anchor); err != nil {
		return err
	}
	logger.Infof("[IntegrityAnchorJob] 已生成锚点 %s：窗口 [%s, %s)，行数 %d",
		anchor.ID, windowStart.Format(time.RFC3339), windowEnd.Format(time.RFC3339), anchor.RowCount)
	return nil
}

// RunOnce 手动执行一轮封口（供测试或运维触发）。
func (j *IntegrityAnchorJob) RunOnce(ctx context.Context) error {
	return j.runOnce(ctx)
}
