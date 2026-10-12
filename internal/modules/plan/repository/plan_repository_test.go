package repository

import (
	"context"
	"testing"

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
