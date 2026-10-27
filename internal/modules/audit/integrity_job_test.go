package audit

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/config"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"
	"meteorx/internal/modules/audit/service"
	"meteorx/pkg/idgen"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func newAnchorGuard(secret string) *service.IntegrityGuard {
	return service.NewIntegrityGuard(config.Config{Audit: config.AuditConfig{
		Integrity: config.AuditIntegrityConfig{Enabled: true, SecretKey: secret},
	}})
}

func newAnchorDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, repository.AutoMigrate(db))
	return db
}

// seedHashedLogAt 以显式（历史）created_at 写入一行带正确行摘要的审计日志。
// 摘要按该 created_at 计算并原样落库（repo.Create 保留非零 CreatedAt，sqlite 无损往返），
// 从而用小时级窗口边界彻底规避真实时钟 tick 竞争，使锚点封口测试确定性可重复。
func seedHashedLogAt(t *testing.T, db *gorm.DB, guard *service.IntegrityGuard, createdAt time.Time) {
	t.Helper()
	repo := repository.NewAuditLogRepository(db)
	log := &model.AuditLog{
		ID: idgen.New(), UserID: "u-1", Username: "alice", Module: "user",
		Action: model.ActionTypeCreate, Result: model.ResultSuccess,
		CreatedAt: createdAt,
	}
	log.RecordHash = guard.Compute(log)
	require.NoError(t, repo.Create(context.Background(), log))
}

func countAnchors(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&repository.AuditIntegrityAnchorPO{}).Count(&n).Error)
	return n
}

// TestIntegrityAnchorJob_ChainThenVerified 两窗口封口后重放校验应通过，且第二条锚点正确链向第一条。
func TestIntegrityAnchorJob_ChainThenVerified(t *testing.T) {
	db := newAnchorDB(t)
	guard := newAnchorGuard("pepper-A")
	base := time.Now()

	// 第一窗口：两条 3 小时前的行
	seedHashedLogAt(t, db, guard, base.Add(-3*time.Hour))
	seedHashedLogAt(t, db, guard, base.Add(-3*time.Hour+time.Second))

	job := NewIntegrityAnchorJob(db, config.Config{}, guard)
	job.grace = 2*time.Hour + time.Minute // 窗口右端≈base-2h1m，含 3h 前两条、不含 2h 前一条
	require.NoError(t, job.RunOnce(context.Background()))
	require.EqualValues(t, 1, countAnchors(t, db), "首个非空窗口应封口一条锚点")

	// 第二窗口：一条 2 小时前的行
	seedHashedLogAt(t, db, guard, base.Add(-2*time.Hour))
	job.grace = time.Hour // 窗口右端≈base-1h，左端衔接上一条右端
	require.NoError(t, job.RunOnce(context.Background()))
	require.EqualValues(t, 2, countAnchors(t, db))

	var anchors []repository.AuditIntegrityAnchorPO
	require.NoError(t, db.Order("window_end ASC").Find(&anchors).Error)
	require.Equal(t, "", anchors[0].PrevAnchorHash, "创世锚点无前置")
	require.Equal(t, anchors[0].AnchorHash, anchors[1].PrevAnchorHash, "第二条须链向第一条 AnchorHash")

	res, err := service.NewIntegrityService(guard, repository.NewIntegrityRepository(db)).
		Verify(context.Background())
	require.NoError(t, err)
	require.Equal(t, service.IntegrityVerdictVerified, res.Verdict)
	require.EqualValues(t, 2, res.AnchorsChecked)
	require.Zero(t, res.TamperedRows)
	require.Empty(t, res.AnomalyAnchorIDs)
}

// TestIntegrityAnchorJob_EmptyWindowNoAnchor 无新行（空窗口）时不封口，避免周期性空锚点噪声。
func TestIntegrityAnchorJob_EmptyWindowNoAnchor(t *testing.T) {
	db := newAnchorDB(t)
	guard := newAnchorGuard("pepper-A")

	job := NewIntegrityAnchorJob(db, config.Config{}, guard)
	job.grace = 0
	require.NoError(t, job.RunOnce(context.Background()))
	require.Zero(t, countAnchors(t, db), "空窗口不得封口")
}

// TestIntegrityAnchorJob_DetectsDeletedRow 删除一行后行级重算无法发现（存留行摘要仍自洽），
// 但锚点窗口行数/摘要须报异常。
func TestIntegrityAnchorJob_DetectsDeletedRow(t *testing.T) {
	db := newAnchorDB(t)
	guard := newAnchorGuard("pepper-A")
	base := time.Now()
	oldest := base.Add(-2 * time.Hour)
	seedHashedLogAt(t, db, guard, oldest)
	seedHashedLogAt(t, db, guard, base.Add(-2*time.Hour+time.Second))

	job := NewIntegrityAnchorJob(db, config.Config{}, guard)
	job.grace = time.Minute // 历史行远在窗口右端之前，稳定封口
	require.NoError(t, job.RunOnce(context.Background()))
	require.EqualValues(t, 1, countAnchors(t, db))

	// 删掉最早一行（攻击者删除历史留痕）
	require.NoError(t, db.Delete(&repository.AuditLogPO{}, "created_at = ?", oldest).Error)

	res, err := service.NewIntegrityService(guard, repository.NewIntegrityRepository(db)).
		Verify(context.Background())
	require.NoError(t, err)
	require.Equal(t, service.IntegrityVerdictTampered, res.Verdict)
	require.NotEmpty(t, res.AnomalyAnchorIDs, "删行须由锚点窗口检出")
}

// TestIntegrityAnchorJob_DetectsChainBreak 改写链中锚点的 PrevAnchorHash 应检出断链与锚点自证失败。
func TestIntegrityAnchorJob_DetectsChainBreak(t *testing.T) {
	db := newAnchorDB(t)
	guard := newAnchorGuard("pepper-A")
	base := time.Now()
	seedHashedLogAt(t, db, guard, base.Add(-3*time.Hour))

	job := NewIntegrityAnchorJob(db, config.Config{}, guard)
	job.grace = 2*time.Hour + time.Minute
	require.NoError(t, job.RunOnce(context.Background()))
	seedHashedLogAt(t, db, guard, base.Add(-2*time.Hour))
	job.grace = time.Hour
	require.NoError(t, job.RunOnce(context.Background()))
	require.EqualValues(t, 2, countAnchors(t, db))

	// 断链：把第二条锚点的 PrevAnchorHash 改成垃圾值
	var second repository.AuditIntegrityAnchorPO
	require.NoError(t, db.Order("window_end ASC").Last(&second).Error)
	require.NoError(t, db.Model(&repository.AuditIntegrityAnchorPO{}).
		Where("id = ?", second.ID).Update("prev_anchor_hash", "deadbeef").Error)

	res, err := service.NewIntegrityService(guard, repository.NewIntegrityRepository(db)).
		Verify(context.Background())
	require.NoError(t, err)
	require.Equal(t, service.IntegrityVerdictTampered, res.Verdict)
	require.Contains(t, res.AnomalyAnchorIDs, second.ID)
}

// TestIntegrityAnchorJob_Disabled 未配置 pepper 时：不封口、不写行摘要、校验返回 not_configured、不阻断写入。
func TestIntegrityAnchorJob_Disabled(t *testing.T) {
	db := newAnchorDB(t)
	disabled := service.NewIntegrityGuard(config.Config{})

	svc := service.NewAuditService(repository.NewAuditLogRepository(db), service.WithIntegrity(disabled))
	_, err := svc.CreateLog(context.Background(), dto.CreateAuditLogReq{
		UserID: "u-1", Module: "user", Action: model.ActionTypeCreate, Result: model.ResultSuccess,
	})
	require.NoError(t, err, "未配置 pepper 不得阻断写入")

	var rows []repository.AuditLogPO
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Empty(t, rows[0].RecordHash, "未启用时不写行摘要")

	job := NewIntegrityAnchorJob(db, config.Config{}, disabled)
	job.grace = 0
	require.NoError(t, job.RunOnce(context.Background()))
	require.Zero(t, countAnchors(t, db), "未启用时不得生成锚点")

	res, err := service.NewIntegrityService(disabled, repository.NewIntegrityRepository(db)).
		Verify(context.Background())
	require.NoError(t, err)
	require.Equal(t, service.IntegrityVerdictNotConfigured, res.Verdict)
}

// TestIntegrityAnchorJob_StartSkippedWhenDisabled 未启用时 Start 应立即返回、不起后台任务。
func TestIntegrityAnchorJob_StartSkippedWhenDisabled(t *testing.T) {
	db := newAnchorDB(t)
	disabled := service.NewIntegrityGuard(config.Config{})
	job := NewIntegrityAnchorJob(db, config.Config{}, disabled)

	done := make(chan struct{})
	go func() { job.Start(context.Background(), time.Hour); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("未启用完整性时 Start 应立即返回")
	}
}
