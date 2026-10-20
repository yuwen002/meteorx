package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"meteorx/internal/modules/export/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// newExportTestDB 创建内存 SQLite 并迁移 export_jobs 表。
// 用具名 shared-cache DSN 而非裸 ":memory:"：后者连接池每新一条连接都是全新空库，
// 而 worker 心跳与主线程断言会并发访问库，随机出现 no such table。
func newExportTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, AutoMigrate(db))
	return db
}

func newJob(id, status string, expiresAt time.Time) *model.ExportJob {
	return &model.ExportJob{
		ID:        id,
		TenantID:  "t1",
		UserID:    "u1",
		Resource:  model.ResourceUser,
		Status:    status,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
}

// resetStatus 测试辅助：直接改库以构造“被回收后回队”等状态，绕过生产端的 pending→running CAS；
// 不把测试专用方法放进 ExportRepository 接口。
func resetStatus(t *testing.T, db *gorm.DB, id, status string) {
	t.Helper()
	require.NoError(t, db.Model(&ExportJobPO{}).Where("id = ?", id).Update("status", status).Error)
}

func TestExportRepo_Create_GetByID(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newJob("job1", model.ExportStatusPending, time.Now().Add(time.Hour))))

	got, err := repo.GetByID(ctx, "job1")
	require.NoError(t, err)
	require.Equal(t, "job1", got.ID)
	require.Equal(t, model.ExportStatusPending, got.Status)

	_, err = repo.GetByID(ctx, "missing")
	require.True(t, errors.Is(err, ErrExportJobNotFound))
}

// claimOf 构造一次抢占凭据；令牌由调用方指定以便后续验证归属。
func claimOf(started, lease time.Time, token string) Claim {
	return Claim{StartedAt: started, LeaseExpiresAt: lease, Token: token}
}

func TestExportRepo_MarkRunning_Preemption(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newJob("job1", model.ExportStatusPending, time.Now().Add(time.Hour))))

	lease := time.Now().Add(time.Minute)
	ok, err := repo.MarkRunning(ctx, "job1", claimOf(time.Now(), lease, "tok-a"))
	require.NoError(t, err)
	require.True(t, ok, "首次抢占 pending→running 应成功")

	ok, err = repo.MarkRunning(ctx, "job1", claimOf(time.Now(), lease, "tok-b"))
	require.NoError(t, err)
	require.False(t, ok, "已 running 的任务不应被重复抢占")
}

// TestExportRepo_MarkRunning_SetsLeaseAndAttempts 抢占必须同时写入租约与递增执行次数：
// 租约是后续回收“硬崩溃残留 running”的唯一依据，执行次数是重试封顶的依据。
func TestExportRepo_MarkRunning_SetsLeaseAndAttempts(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newJob("job1", model.ExportStatusPending, time.Now().Add(time.Hour))))

	started := time.Now().Truncate(time.Millisecond)
	lease := started.Add(time.Minute)
	ok, err := repo.MarkRunning(ctx, "job1", claimOf(started, lease, "tok-1"))
	require.NoError(t, err)
	require.True(t, ok)

	got, err := repo.GetByID(ctx, "job1")
	require.NoError(t, err)
	require.NotNil(t, got.LeaseExpiresAt, "抢占后必须持有租约")
	require.True(t, got.LeaseExpiresAt.Truncate(time.Millisecond).Equal(lease), "租约应写入传入值")
	require.Equal(t, "tok-1", got.LeaseToken, "归属令牌应落库")
	require.Equal(t, 1, got.Attempts, "首次执行计入 1 次")

	// 回队后再次抢占：次数累加、令牌换新
	require.NoError(t, repo.Create(ctx, newJob("job2", model.ExportStatusPending, time.Now().Add(time.Hour))))
	_, err = repo.MarkRunning(ctx, "job2", claimOf(started, lease, "tok-2"))
	require.NoError(t, err)
	resetStatus(t, db, "job2", model.ExportStatusPending) // 模拟被回收回队
	_, err = repo.MarkRunning(ctx, "job2", claimOf(started, lease, "tok-3"))
	require.NoError(t, err)
	got2, err := repo.GetByID(ctx, "job2")
	require.NoError(t, err)
	require.Equal(t, 2, got2.Attempts, "重复抢占（失败重驱）应累加执行次数")
	require.Equal(t, "tok-3", got2.LeaseToken, "重驱后应持有新令牌")
}

// TestExportRepo_Heartbeat_ExtendsOnlyRunningLease 心跳只能续期正在执行的任务，
// 否则会把已被回收/已完成的任务租约复活，使其逃离回收。
func TestExportRepo_Heartbeat_ExtendsOnlyRunningLease(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newJob("job1", model.ExportStatusPending, time.Now().Add(time.Hour))))

	old := time.Now().Add(-time.Minute).Truncate(time.Millisecond)
	ok, err := repo.Heartbeat(ctx, "job1", "tok-1", time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.False(t, ok, "pending 任务无租约可续")

	_, err = repo.MarkRunning(ctx, "job1", claimOf(old, old, "tok-1"))
	require.NoError(t, err)

	// 错令牌不得给他人续租：本执行已被回收重驱时，按 id 续租会把新持有者的租约拉到未来、使其逃离回收
	ok, err = repo.Heartbeat(ctx, "job1", "tok-other", time.Now().Add(10*time.Minute))
	require.NoError(t, err)
	require.False(t, ok, "令牌不匹配时不能续租")

	newLease := time.Now().Add(5 * time.Minute).Truncate(time.Millisecond)
	ok, err = repo.Heartbeat(ctx, "job1", "tok-1", newLease)
	require.NoError(t, err)
	require.True(t, ok, "持有效令牌应能续租")

	got, err := repo.GetByID(ctx, "job1")
	require.NoError(t, err)
	require.NotNil(t, got.LeaseExpiresAt)
	require.True(t, got.LeaseExpiresAt.Truncate(time.Millisecond).Equal(newLease))

	// 已跑完的任务不再续租
	require.NoError(t, repo.MarkCompleted(ctx, "job1", "tok-1", "/tmp/job1.csv", 1, time.Now()))
	ok, err = repo.Heartbeat(ctx, "job1", "tok-1", time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.False(t, ok)
	got, err = repo.GetByID(ctx, "job1")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusCompleted, got.Status, "心跳不得改变终态")
}

// TestExportRepo_ZombieWritesIgnoredAfterReclaim 回收重驱后旧执行的回写必须空转：
// 心跳断供（进程暂停、DB 超时）超过租约时，同进程的 poll 或其他副本会把任务重驱，
// 若旧执行仍可按 id 回写，则会覆写新执行的租约与结果。
func TestExportRepo_ZombieWritesIgnoredAfterReclaim(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	require.NoError(t, repo.Create(ctx, newJob("job1", model.ExportStatusPending, now.Add(time.Hour))))
	_, err := repo.MarkRunning(ctx, "job1", claimOf(now.Add(-2*time.Hour), now.Add(-time.Hour), "tok-zombie"))
	require.NoError(t, err)

	requeued, _, err := repo.ReclaimExpiredLeases(ctx, now, time.Minute, 3)
	require.NoError(t, err)
	require.Equal(t, int64(1), requeued)
	_, err = repo.MarkRunning(ctx, "job1", claimOf(now, now.Add(time.Hour), "tok-new"))
	require.NoError(t, err)

	// 僵尸的三种回写均应空转
	ok, err := repo.Heartbeat(ctx, "job1", "tok-zombie", now.Add(99*time.Hour))
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, repo.MarkCompleted(ctx, "job1", "tok-zombie", "/tmp/zombie.csv", 999, now))
	require.NoError(t, repo.MarkFailed(ctx, "job1", "tok-zombie", "僵尸回写", now))

	got, err := repo.GetByID(ctx, "job1")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusRunning, got.Status, "新执行状态不能被旧执行覆写")
	require.Equal(t, "tok-new", got.LeaseToken)
	require.Empty(t, got.FilePath, "旧执行的结果不得写入")
	require.Empty(t, got.ErrorMsg)
	require.NotNil(t, got.LeaseExpiresAt)
	require.False(t, got.LeaseExpiresAt.After(now.Add(9*time.Hour)), "旧心跳不得把新执行租约拉到99小时后")

	// 当前持有者的回写仍正常生效
	require.NoError(t, repo.MarkCompleted(ctx, "job1", "tok-new", "/tmp/new.csv", 7, now))
	got, err = repo.GetByID(ctx, "job1")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusCompleted, got.Status)
	require.Equal(t, int64(7), got.Rows)
}

// TestExportRepo_ReclaimExpiredLeases_RequeuesStaleAndKeepsLive 租约过期的 running
// （进程被 kill -9 / OOM，无任何收尾机会）必须回队；而持有效租约的 running 是其他副本
// 正在执行的活任务，绝对不能抢。
func TestExportRepo_ReclaimExpiredLeases_RequeuesStaleAndKeepsLive(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	for _, id := range []string{"stale", "live"} {
		require.NoError(t, repo.Create(ctx, newJob(id, model.ExportStatusPending, now.Add(time.Hour))))
		_, err := repo.MarkRunning(ctx, id, claimOf(now.Add(-2*time.Minute), now.Add(-time.Minute), "tok-"+id))
		require.NoError(t, err)
	}
	// live：租约未到（心跳正常），即使 started_at 特别早也不能抢
	resetStatus(t, db, "live", model.ExportStatusPending)
	_, err := repo.MarkRunning(ctx, "live", claimOf(now, now.Add(2*time.Hour), "tok-live-2"))
	require.NoError(t, err)

	requeued, aborted, err := repo.ReclaimExpiredLeases(ctx, now, time.Minute, 3)
	require.NoError(t, err)
	require.Equal(t, int64(1), requeued)
	require.Equal(t, int64(0), aborted)

	got, err := repo.GetByID(ctx, "stale")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusPending, got.Status, "租约过期的任务应回队重驱")
	require.Nil(t, got.LeaseExpiresAt, "回队后不应残留旧租约")
	require.Nil(t, got.StartedAt)

	live, err := repo.GetByID(ctx, "live")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusRunning, live.Status, "持有效租约的任务属其他存活 worker，不得被抢")
}

// TestExportRepo_ReclaimExpiredLeases_AbortsAfterMaxAttempts 反复把进程撞死的任务
// （如数据量导致 OOM）不能无限重驱，达上限后置为失败，否则会不断重启整个服务。
func TestExportRepo_ReclaimExpiredLeases_AbortsAfterMaxAttempts(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	require.NoError(t, repo.Create(ctx, newJob("bomb", model.ExportStatusPending, now.Add(time.Hour))))
	// 模拟已反复崩溃 3 次（attempts=3）且当前这一次的租约已过期
	expired := now.Add(-time.Minute)
	for i := 0; i < 3; i++ {
		_, err := repo.MarkRunning(ctx, "bomb", claimOf(now.Add(-time.Hour), expired, fmt.Sprintf("tok-%d", i)))
		require.NoError(t, err)
		resetStatus(t, db, "bomb", model.ExportStatusPending)
	}
	_, err := repo.MarkRunning(ctx, "bomb", claimOf(now.Add(-time.Hour), expired, "tok-last"))
	require.NoError(t, err)
	bomb, err := repo.GetByID(ctx, "bomb")
	require.NoError(t, err)
	require.Equal(t, 4, bomb.Attempts)

	requeued, aborted, err := repo.ReclaimExpiredLeases(ctx, now, time.Minute, 3)
	require.NoError(t, err)
	require.Equal(t, int64(0), requeued, "达上限后不应再回队")
	require.Equal(t, int64(1), aborted)

	got, err := repo.GetByID(ctx, "bomb")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusFailed, got.Status)
	require.Equal(t, ErrMsgInterruptExhausted, got.ErrorMsg)
	require.NotNil(t, got.CompletedAt, "失败为终态应写入完成时间")
}

// TestExportRepo_ReclaimExpiredLeases_LegacyNullLease 升级前遗留的 running 行没有租约列
// （AutoMigrate 补列为 NULL），若只按租约判定则永远得不到回收；以 started_at 旧于 TTL 作为回退判定。
func TestExportRepo_ReclaimExpiredLeases_LegacyNullLease(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	require.NoError(t, repo.Create(ctx, newJob("legacy", model.ExportStatusPending, now.Add(time.Hour))))
	_, err := repo.MarkRunning(ctx, "legacy", claimOf(now.Add(-2*time.Hour), now.Add(-time.Hour), "tok-legacy"))
	require.NoError(t, err)
	// 抹掉租约，还原本次变更前的历史数据态
	require.NoError(t, db.Model(&ExportJobPO{}).Where("id = ?", "legacy").
		UpdateColumn("lease_expires_at", nil).Error)

	requeued, aborted, err := repo.ReclaimExpiredLeases(ctx, now, time.Minute, 3)
	require.NoError(t, err)
	require.Equal(t, int64(1), requeued)
	require.Equal(t, int64(0), aborted)
	got, err := repo.GetByID(ctx, "legacy")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusPending, got.Status, "无租约但 started_at 已旧的 running 行应被回收")
}

// TestExportRepo_ReclaimExpiredLeases_NoOpWhenNothingStale 无 stale 行时不得误动任何任务。
func TestExportRepo_ReclaimExpiredLeases_NoOpWhenNothingStale(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	require.NoError(t, repo.Create(ctx, newJob("p1", model.ExportStatusPending, now.Add(time.Hour))))
	require.NoError(t, repo.Create(ctx, newJob("c1", model.ExportStatusCompleted, now.Add(time.Hour))))

	requeued, aborted, err := repo.ReclaimExpiredLeases(ctx, now, time.Minute, 3)
	require.NoError(t, err)
	require.Equal(t, int64(0), requeued)
	require.Equal(t, int64(0), aborted)
}

func TestExportRepo_MarkCompleted_Failed(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newJob("job1", model.ExportStatusPending, time.Now().Add(time.Hour))))
	_, err := repo.MarkRunning(ctx, "job1", claimOf(time.Now(), time.Now().Add(time.Minute), "tok-1"))
	require.NoError(t, err)

	require.NoError(t, repo.MarkCompleted(ctx, "job1", "tok-1", "/tmp/job1.csv", 42, time.Now()))
	got, err := repo.GetByID(ctx, "job1")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusCompleted, got.Status)
	require.Equal(t, int64(42), got.Rows)
	require.Equal(t, "/tmp/job1.csv", got.FilePath)
	require.Nil(t, got.LeaseExpiresAt, "进终态后不应残留租约")

	// MarkFailed 只对当前令牌持有者生效：未抢占与已终态都不能被改写
	require.NoError(t, repo.Create(ctx, newJob("job2", model.ExportStatusPending, time.Now().Add(time.Hour))))
	require.NoError(t, repo.MarkFailed(ctx, "job2", "tok-none", "boom", time.Now()))
	got2, err := repo.GetByID(ctx, "job2")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusPending, got2.Status, "无令牌归属的回写应空转")

	_, err = repo.MarkRunning(ctx, "job2", claimOf(time.Now(), time.Now().Add(time.Minute), "tok-2"))
	require.NoError(t, err)
	require.NoError(t, repo.MarkFailed(ctx, "job2", "tok-2", "boom", time.Now()))
	got2, err = repo.GetByID(ctx, "job2")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusFailed, got2.Status)
	require.Equal(t, "boom", got2.ErrorMsg)
}

func TestExportRepo_ListPending(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newJob("p1", model.ExportStatusPending, time.Now().Add(time.Hour))))
	require.NoError(t, repo.Create(ctx, newJob("p2", model.ExportStatusPending, time.Now().Add(time.Hour))))
	require.NoError(t, repo.Create(ctx, newJob("c1", model.ExportStatusCompleted, time.Now().Add(time.Hour))))

	list, err := repo.ListPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, list, 2)
}

func TestExportRepo_ListExpired(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	now := time.Now()
	require.NoError(t, repo.Create(ctx, newJob("exp", model.ExportStatusCompleted, now.Add(-time.Hour))))
	require.NoError(t, repo.Create(ctx, newJob("fresh", model.ExportStatusCompleted, now.Add(time.Hour))))

	list, err := repo.ListExpired(ctx, now, 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "exp", list[0].ID)
}

func TestExportRepo_ListByUser_Ownership(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newJob("j1", model.ExportStatusPending, time.Now().Add(time.Hour))))
	other := newJob("j2", model.ExportStatusPending, time.Now().Add(time.Hour))
	other.UserID = "u2"
	require.NoError(t, repo.Create(ctx, other))

	items, total, err := repo.ListByUser(ctx, "t1", "u1", 1, 20, "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "j1", items[0].ID)
}

func TestExportRepo_CountActiveByTenant(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newJob("p1", model.ExportStatusPending, time.Now().Add(time.Hour))))
	require.NoError(t, repo.Create(ctx, newJob("r1", model.ExportStatusRunning, time.Now().Add(time.Hour))))
	require.NoError(t, repo.Create(ctx, newJob("c1", model.ExportStatusCompleted, time.Now().Add(time.Hour))))
	require.NoError(t, repo.Create(ctx, newJob("f1", model.ExportStatusFailed, time.Now().Add(time.Hour))))

	// 另一租户的任务不计入
	other := newJob("t2p", model.ExportStatusPending, time.Now().Add(time.Hour))
	other.TenantID = "t2"
	require.NoError(t, repo.Create(ctx, other))

	count, err := repo.CountActiveByTenant(ctx, "t1")
	require.NoError(t, err)
	require.Equal(t, int64(2), count, "仅 pending/running 计入未完结")

	count, err = repo.CountActiveByTenant(ctx, "t2")
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	count, err = repo.CountActiveByTenant(ctx, "t-none")
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

func TestExportRepo_Delete(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newJob("job1", model.ExportStatusPending, time.Now().Add(time.Hour))))
	require.NoError(t, repo.Delete(ctx, "job1"))

	_, err := repo.GetByID(ctx, "job1")
	require.True(t, errors.Is(err, ErrExportJobNotFound))
}
