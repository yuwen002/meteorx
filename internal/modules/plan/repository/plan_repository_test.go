package repository

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/plan/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// newPlanTestDB 创建内存 SQLite 并迁移套餐相关表
func newPlanTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	require.NoError(t, AutoMigrate(db))
	return db
}

// TestPlanRepo_toDomain_NormalizeLegacyZero 遗留行（配额列为 0）读路径统一归一为 -1（不限），
// 防止消费端以 limit<0 判不限时将 0 误当作「上限为 0」而拦截全部操作。
func TestPlanRepo_toDomain_NormalizeLegacyZero(t *testing.T) {
	db := newPlanTestDB(t)
	ctx := context.Background()

	// 直接落库一条四个配额字段均为 0 的行，模拟配额列引入前创建的遗留套餐
	// （repo.Create 原样写入传入值，传 0 即存 0，绕过 service 层写路径归一化）
	require.NoError(t, db.WithContext(ctx).Create(&PlanPO{
		ID: "plan-legacy", Name: strPtr("遗留版"), Code: "legacy",
		UserLimit: 0, StorageLimit: 0, ApiRateLimit: 0, WikiSpaceLimit: 0,
		Status: 1,
	}).Error)

	repo := NewPlanRepository(db)

	got, err := repo.GetByID(ctx, "plan-legacy")
	require.NoError(t, err)
	require.Equal(t, -1, got.UserLimit)
	require.Equal(t, int64(-1), got.StorageLimit)
	require.Equal(t, -1, got.ApiRateLimit)
	require.Equal(t, -1, got.WikiSpaceLimit)

	byCode, err := repo.GetByCode(ctx, "legacy")
	require.NoError(t, err)
	require.Equal(t, int64(-1), byCode.StorageLimit)

	// 分页列表同样归一
	plans, total, err := repo.FindPage(ctx, 1, 10, "", nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, plans, 1)
	require.Equal(t, -1, plans[0].ApiRateLimit)
}

// TestPlanRepo_toDomain_KeepsExplicitValues 显式设置的 -1 与正数配额原样保留，不被归一化误改。
func TestPlanRepo_toDomain_KeepsExplicitValues(t *testing.T) {
	db := newPlanTestDB(t)
	ctx := context.Background()
	repo := NewPlanRepository(db)

	// 显式正数配额
	require.NoError(t, repo.Create(ctx, &model.Plan{
		ID: "plan-pro", Name: "专业版", Code: "pro",
		UserLimit: 100, StorageLimit: 1073741824, ApiRateLimit: 600, WikiSpaceLimit: 20,
		Status: 1,
	}))
	// 显式 -1（不限）
	require.NoError(t, repo.Create(ctx, &model.Plan{
		ID: "plan-free", Name: "免费版", Code: "free",
		UserLimit: -1, StorageLimit: -1, ApiRateLimit: -1, WikiSpaceLimit: -1,
		Status: 1,
	}))

	pro, err := repo.GetByID(ctx, "plan-pro")
	require.NoError(t, err)
	require.Equal(t, 100, pro.UserLimit)
	require.Equal(t, int64(1073741824), pro.StorageLimit)
	require.Equal(t, 600, pro.ApiRateLimit)
	require.Equal(t, 20, pro.WikiSpaceLimit)

	free, err := repo.GetByID(ctx, "plan-free")
	require.NoError(t, err)
	require.Equal(t, -1, free.UserLimit)
	require.Equal(t, int64(-1), free.StorageLimit)
	require.Equal(t, -1, free.ApiRateLimit)
	require.Equal(t, -1, free.WikiSpaceLimit)
}

// seedSoonSub 写入一条 3 天后到期的生效订阅，remindedAt 为上次提醒时间（nil 表示从未提醒）
func seedSoonSub(t *testing.T, db *gorm.DB, id string, remindedAt *time.Time) {
	t.Helper()
	in3 := time.Now().Add(3 * 24 * time.Hour)
	require.NoError(t, db.Create(&SubscriptionPO{
		ID: id, TenantID: "t-" + id, PlanID: "p-1", Status: model.SubscriptionActive,
		StartedAt: time.Now().Add(-24 * time.Hour), ExpiresAt: &in3, ExpiryRemindedAt: remindedAt,
	}).Error)
}

// TestSubRepo_FindExpiringSoon_AppliesRemindCooldown 去重必须依赖库内时间而非进程内存：
// 原实现只按到期窗口返回，调用方拿不到「是否刚提醒过」，重启后同一订阅会立即重发。
// 现扫描同时承担冷却判定：未提醒过、上次提醒早于冷却期的入选，冷却期内的排除。
func TestSubRepo_FindExpiringSoon_AppliesRemindCooldown(t *testing.T) {
	db := newPlanTestDB(t)
	ctx := context.Background()
	now := time.Now()

	seedSoonSub(t, db, "s-none", nil)
	old := now.Add(-25 * time.Hour)
	seedSoonSub(t, db, "s-old", &old)
	fresh := now.Add(-time.Hour)
	seedSoonSub(t, db, "s-fresh", &fresh)

	repo := NewSubscriptionRepository(db)
	subs, err := repo.FindExpiringSoon(ctx, 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)

	got := map[string]bool{}
	for _, s := range subs {
		got[s.ID] = true
	}
	require.True(t, got["s-none"], "从未提醒过的订阅必须进入提醒集合")
	require.True(t, got["s-old"], "上次提醒已超冷却期的订阅应可再次提醒")
	require.False(t, got["s-fresh"], "冷却期内的订阅不得重复提醒")
	require.Len(t, subs, 2)

	// 提醒时间需随领域模型上浮，否则上层无法区分「新提醒」与「重发」
	for _, s := range subs {
		if s.ID == "s-old" {
			require.NotNil(t, s.ExpiryRemindedAt)
			require.WithinDuration(t, old, *s.ExpiryRemindedAt, time.Minute)
		} else {
			require.Nil(t, s.ExpiryRemindedAt)
		}
	}
}

// TestSubRepo_MarkExpiryReminded 发送后回写提醒时间，且只影响目标行
func TestSubRepo_MarkExpiryReminded(t *testing.T) {
	db := newPlanTestDB(t)
	ctx := context.Background()
	seedSoonSub(t, db, "s-a", nil)
	seedSoonSub(t, db, "s-b", nil)

	at := time.Now()
	repo := NewSubscriptionRepository(db)
	require.NoError(t, repo.MarkExpiryReminded(ctx, "s-a", at))

	var a, b SubscriptionPO
	require.NoError(t, db.First(&a, "id = ?", "s-a").Error)
	require.NoError(t, db.First(&b, "id = ?", "s-b").Error)
	require.NotNil(t, a.ExpiryRemindedAt, "提醒时间必须落库，重启后仍生效")
	require.WithinDuration(t, at, *a.ExpiryRemindedAt, time.Minute)
	require.Nil(t, b.ExpiryRemindedAt, "不得误伤其它订阅")

	// 回写后该行应立即从冷却期内的提醒集合中消失
	subs, err := repo.FindExpiringSoon(ctx, 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)
	require.Len(t, subs, 1)
	require.Equal(t, "s-b", subs[0].ID)
}
