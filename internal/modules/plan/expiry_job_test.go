package plan

import (
	"context"
	"testing"
	"time"

	planModel "meteorx/internal/modules/plan/model"
	"meteorx/internal/modules/plan/repository"
	"meteorx/internal/modules/plan/service"
	tenantModel "meteorx/internal/modules/tenant/model"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	"meteorx/internal/notify"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// recordNotifier 记录送达消息的最小通知渠道，用于断言提醒是否发出、发出几次
type recordNotifier struct {
	msgs []*notify.Message
}

func (r *recordNotifier) Type() notify.ChannelType { return notify.ChannelEmail }
func (r *recordNotifier) Name() string             { return "record" }
func (r *recordNotifier) Send(_ context.Context, msg *notify.Message) error {
	r.msgs = append(r.msgs, msg)
	return nil
}

// useRecorder 把记录型渠道注册为全局通知管理器，测试结束后恢复原值
func useRecorder(t *testing.T) *recordNotifier {
	t.Helper()
	rec := &recordNotifier{}
	prev := notify.GetGlobalManager()
	m := notify.NewManager(nil, "", "", nil)
	m.AddChannel(rec)
	notify.SetGlobalManager(m)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })
	return rec
}

// fakeTenants 满足 TenantDirectory 的最小租户查询实现
type fakeTenants struct{ ids []string }

func (f *fakeTenants) GetByID(_ context.Context, id string) (*tenantModel.Tenant, error) {
	f.ids = append(f.ids, id)
	return &tenantModel.Tenant{ID: id, Name: "示例租户", ContactEmail: "admin@example.com"}, nil
}

// recordingDisabler 记录到期禁用调用
type recordingDisabler struct{ tenantIDs []string }

func (d *recordingDisabler) DisableTenant(_ context.Context, tenantID string) error {
	d.tenantIDs = append(d.tenantIDs, tenantID)
	return nil
}

// seedSub 写入一条生效订阅，expiresAt 为 nil 表示长期有效
func seedSub(t *testing.T, subRepo *repository.MockSubscriptionRepository, id, tenantID string, expiresAt *time.Time) {
	t.Helper()
	if err := subRepo.Create(context.Background(), &planModel.TenantSubscription{
		ID: id, TenantID: tenantID, PlanID: "plan-1",
		Status: planModel.SubscriptionActive, StartedAt: time.Now().Add(-24 * time.Hour),
		ExpiresAt: expiresAt,
	}); err != nil {
		t.Fatalf("seed subscription failed: %v", err)
	}
}

func newJobWith(t *testing.T, subRepo *repository.MockSubscriptionRepository, tenants TenantDirectory) *ExpiryJob {
	t.Helper()
	planSvc := service.NewPlanService(repository.NewMockPlanRepository(), subRepo, nil)
	return newExpiryJobWith(planSvc, subRepo, tenants)
}

// TestExpiryJob_RemindsEvenWhenNothingExpired 本轮没有订阅到期时，临期提醒仍必须发出。
// 原实现在 expiredTenantIDs 为空时提前 return，使「7 天内到期」提醒只有在恰好有别的订阅
// 过期的那一轮才发得出——多数轮次一条也不发，续费提醒形同虚设。
func TestExpiryJob_RemindsEvenWhenNothingExpired(t *testing.T) {
	subRepo := repository.NewMockSubscriptionRepository()
	in3 := time.Now().Add(3 * 24 * time.Hour)
	seedSub(t, subRepo, "sub-soon", "tenant-1", &in3)

	rec := useRecorder(t)
	job := newJobWith(t, subRepo, &fakeTenants{})

	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}
	if len(rec.msgs) != 1 {
		t.Errorf("expected 1 expiry reminder even when nothing expired this round, got %d", len(rec.msgs))
	}
}

// TestExpiryJob_RemindsOncePerExpiryDay 同一订阅在同一到期日内只提醒一次，
// 避免每轮扫描（默认小时级）都推一条重复提醒。
func TestExpiryJob_RemindsOncePerExpiryDay(t *testing.T) {
	subRepo := repository.NewMockSubscriptionRepository()
	in3 := time.Now().Add(3 * 24 * time.Hour)
	seedSub(t, subRepo, "sub-soon", "tenant-1", &in3)

	rec := useRecorder(t)
	job := newJobWith(t, subRepo, &fakeTenants{})

	for i := 0; i < 3; i++ {
		if err := job.RunOnce(context.Background()); err != nil {
			t.Fatalf("RunOnce #%d failed: %v", i+1, err)
		}
	}
	if len(rec.msgs) != 1 {
		t.Errorf("expected the same subscription reminded once per expiry day, got %d", len(rec.msgs))
	}
}

// TestExpiryJob_DisablesTenantThroughService 定时任务必须经 service 编排禁用到期租户：
// runOnce 重写后不再自行 UpdateStatus，若这层接线丢失，租户将在订阅翻转后继续可用。
func TestExpiryJob_DisablesTenantThroughService(t *testing.T) {
	subRepo := repository.NewMockSubscriptionRepository()
	late := time.Now().Add(-24 * time.Hour)
	seedSub(t, subRepo, "sub-late", "tenant-late", &late)

	planSvc := service.NewPlanService(repository.NewMockPlanRepository(), subRepo, nil)
	d := &recordingDisabler{}
	planSvc.SetTenantDisabler(d)
	job := newExpiryJobWith(planSvc, subRepo, &fakeTenants{})

	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}
	if len(d.tenantIDs) != 1 || d.tenantIDs[0] != "tenant-late" {
		t.Errorf("expected tenant-late disabled once, got %v", d.tenantIDs)
	}
	if got := subRepo.StatusOf("sub-late"); got != planModel.SubscriptionExpired {
		t.Errorf("expected subscription flipped after successful disable, got status=%d", got)
	}
}

// newExpiryDB 建内存 SQLite 并迁移套餐/订阅与租户表。
// 用具名 shared-cache DSN + 单一连接：裸 ":memory:" 下连接池每新开一条连接都是空库，会随机 no such table。
func newExpiryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, repository.AutoMigrate(db))
	require.NoError(t, tenantRepo.AutoMigrate(db))
	return db
}

// TestNewExpiryJob_WiresTenantDisabler 锁住生产装配线：NewExpiryJob 必须把禁用租户的回调注入 service。
// 这一行接线一旦被删，订阅仍会被翻转成 expired，但租户停在 enabled 且再无人重扫（静默免费续期），
// 而上面的 service 层用例用的是自建注入，盖不到这个位置。
func TestNewExpiryJob_WiresTenantDisabler(t *testing.T) {
	db := newExpiryDB(t)
	ctx := context.Background()

	name := "到期租户"
	email := "admin@example.com"
	require.NoError(t, db.Create(&tenantRepo.TenantPO{
		ID: "t-late", Name: &name, Domain: "late.example.com",
		Status: tenantModel.StatusEnabled, ContactEmail: &email,
	}).Error)

	late := time.Now().Add(-24 * time.Hour)
	require.NoError(t, db.Create(&repository.SubscriptionPO{
		ID: "sub-late", TenantID: "t-late", PlanID: "p-1",
		Status: planModel.SubscriptionActive, StartedAt: time.Now().Add(-48 * time.Hour), ExpiresAt: &late,
	}).Error)

	job := NewExpiryJob(db)
	require.NoError(t, job.RunOnce(ctx))

	var tenant tenantRepo.TenantPO
	require.NoError(t, db.First(&tenant, "id = ?", "t-late").Error)
	require.Equal(t, tenantModel.StatusDisabled, tenant.Status,
		"到期任务的唯一作用就是禁用租户，接线丢失则订阅翻转后租户仍可用")

	var sub repository.SubscriptionPO
	require.NoError(t, db.First(&sub, "id = ?", "sub-late").Error)
	require.Equal(t, planModel.SubscriptionExpired, sub.Status)
}
