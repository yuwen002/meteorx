package plan

import (
	"context"
	"errors"
	"testing"
	"time"

	planModel "meteorx/internal/modules/plan/model"
	"meteorx/internal/modules/plan/repository"
	"meteorx/internal/modules/plan/service"
	tenantModel "meteorx/internal/modules/tenant/model"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	userRepo "meteorx/internal/modules/user/repository"
	"meteorx/internal/notify"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// recordNotifier 记录送达消息的测试渠道，可按渠道类型编排失败
type recordNotifier struct {
	typ   notify.ChannelType
	err   error
	msgs  []*notify.Message
	calls int
}

func (r *recordNotifier) Type() notify.ChannelType { return r.typ }
func (r *recordNotifier) Name() string             { return "record-" + string(r.typ) }
func (r *recordNotifier) Send(_ context.Context, msg *notify.Message) error {
	r.calls++
	r.msgs = append(r.msgs, msg)
	return r.err
}

// useRecorders 把指定渠道注册为全局通知管理器，测试结束后恢复原值
func useRecorders(t *testing.T, channels ...*recordNotifier) {
	t.Helper()
	prev := notify.GetGlobalManager()
	m := notify.NewManager(nil, "", "", nil)
	for _, c := range channels {
		m.AddChannel(c)
	}
	notify.SetGlobalManager(m)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })
}

// fakeTenants 满足 TenantDirectory 的最小租户查询实现
type fakeTenants struct{ ids []string }

func (f *fakeTenants) GetByID(_ context.Context, id string) (*tenantModel.Tenant, error) {
	f.ids = append(f.ids, id)
	return &tenantModel.Tenant{ID: id, Name: "示例租户", ContactEmail: "admin@example.com"}, nil
}

// fakeMembers 满足 MemberDirectory：按租户返回成员用户 ID
type fakeMembers struct {
	idsByTenant map[string][]string
	err         error
	called      int
}

func (f *fakeMembers) ListUserIDs(_ context.Context, tenantID string) ([]string, error) {
	f.called++
	if f.err != nil {
		return nil, f.err
	}
	return f.idsByTenant[tenantID], nil
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

func newJobWith(subRepo *repository.MockSubscriptionRepository, tenants TenantDirectory, members MemberDirectory) *ExpiryJob {
	planSvc := service.NewPlanService(repository.NewMockPlanRepository(), subRepo, nil)
	return newExpiryJobWith(planSvc, subRepo, tenants, members)
}

// TestExpiryJob_RemindsEvenWhenNothingExpired 本轮没有订阅到期时，临期提醒仍必须发出。
// 原实现在 expiredTenantIDs 为空时提前 return，使「7 天内到期」提醒只有在恰好有别的订阅
// 过期的那一轮才发得出——多数轮次一条也不发，续费提醒形同虚设。
func TestExpiryJob_RemindsEvenWhenNothingExpired(t *testing.T) {
	subRepo := repository.NewMockSubscriptionRepository()
	in3 := time.Now().Add(3 * 24 * time.Hour)
	seedSub(t, subRepo, "sub-soon", "tenant-1", &in3)

	email := &recordNotifier{typ: notify.ChannelEmail}
	useRecorders(t, email)
	job := newJobWith(subRepo, &fakeTenants{}, &fakeMembers{})

	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}
	require.Equal(t, 1, email.calls, "本轮无到期订阅时也必须发出临期提醒")
}

// TestExpiryJob_MarksRemindedSoNextRoundSkips 提醒成功后必须把提醒时间落库，
// 使下一轮（5 分钟后）扫描因冷却而跳过该订阅——去重从进程内 map 改为落库，
// 才能同时解决「重启重发」与「多副本各发一份」。
func TestExpiryJob_MarksRemindedSoNextRoundSkips(t *testing.T) {
	subRepo := repository.NewMockSubscriptionRepository()
	in3 := time.Now().Add(3 * 24 * time.Hour)
	seedSub(t, subRepo, "sub-soon", "tenant-1", &in3)

	email := &recordNotifier{typ: notify.ChannelEmail}
	useRecorders(t, email)
	job := newJobWith(subRepo, &fakeTenants{}, &fakeMembers{})

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		require.NoError(t, job.RunOnce(ctx))
	}
	require.Equal(t, 1, email.calls, "冷却期内不得重复提醒")
	require.NotNil(t, subRepo.RemindedAtOf("sub-soon"), "成功送达后提醒时间必须落库")
}

// TestExpiryJob_KeepsTryingWhenDeliveryFails 渠道发送失败时不能销账：
// 原实现丢弃 []SendResult 并无条件记录「已提醒」，一次 SMTP 抖动就把该到期日的提醒永久吞掉。
func TestExpiryJob_KeepsTryingWhenDeliveryFails(t *testing.T) {
	subRepo := repository.NewMockSubscriptionRepository()
	in3 := time.Now().Add(3 * 24 * time.Hour)
	seedSub(t, subRepo, "sub-soon", "tenant-1", &in3)

	email := &recordNotifier{typ: notify.ChannelEmail, err: errors.New("smtp down")}
	useRecorders(t, email)
	job := newJobWith(subRepo, &fakeTenants{}, &fakeMembers{})

	ctx := context.Background()
	require.NoError(t, job.RunOnce(ctx))
	require.NoError(t, job.RunOnce(ctx))

	require.Equal(t, 2, email.calls, "发送失败应留待下一轮重试")
	require.Nil(t, subRepo.RemindedAtOf("sub-soon"), "未送达不得写入提醒时间")
}

// TestExpiryJob_EmailFailureNotCoveredByWSSuccess 邮件是唯一能在收件人不在线时仍可达的渠道，
// 只要注册了邮件渠道就必须成功才算送达；否则「WS 成功 + 邮件失败」会被误销账，
// 管理员离线时这条提醒就此消失。
func TestExpiryJob_EmailFailureNotCoveredByWSSuccess(t *testing.T) {
	subRepo := repository.NewMockSubscriptionRepository()
	in3 := time.Now().Add(3 * 24 * time.Hour)
	seedSub(t, subRepo, "sub-soon", "tenant-1", &in3)

	email := &recordNotifier{typ: notify.ChannelEmail, err: errors.New("smtp down")}
	wsC := &recordNotifier{typ: notify.ChannelWebSocket}
	useRecorders(t, email, wsC)
	job := newJobWith(subRepo, &fakeTenants{}, &fakeMembers{idsByTenant: map[string][]string{"tenant-1": {"u-1"}}})

	require.NoError(t, job.RunOnce(context.Background()))
	require.Nil(t, subRepo.RemindedAtOf("sub-soon"), "邮件失败时不得仅凭 WS 成功销账")
}

// TestExpiryJob_WSTargetsMemberIDs WS 分支的收件人必须是租户成员的用户 ID。
// 旧写法传 tenant.ContactEmail，而 hub.clients 以 claims.UserID 为键，邮箱必然 miss 且静默无错，
// 在默认部署（email.enabled=false）下到期提醒 100% 空转。
func TestExpiryJob_WSTargetsMemberIDs(t *testing.T) {
	subRepo := repository.NewMockSubscriptionRepository()
	in3 := time.Now().Add(3 * 24 * time.Hour)
	seedSub(t, subRepo, "sub-soon", "tenant-1", &in3)

	email := &recordNotifier{typ: notify.ChannelEmail}
	wsC := &recordNotifier{typ: notify.ChannelWebSocket}
	useRecorders(t, email, wsC)
	job := newJobWith(subRepo, &fakeTenants{},
		&fakeMembers{idsByTenant: map[string][]string{"tenant-1": {"u-1", "u-2"}}})

	require.NoError(t, job.RunOnce(context.Background()))

	require.Equal(t, 1, wsC.calls, "有成员 ID 时应走 WS 推送")
	require.Equal(t, []string{"u-1", "u-2"}, wsC.msgs[0].Recipients)
	require.Equal(t, []string{"admin@example.com"}, email.msgs[0].Recipients)
}

// TestExpiryJob_MemberLookupFailureStillSendsEmail 成员查询失败只降级掉 WS，
// 不应连带吞掉邮件提醒（邮件是主渠道）。
func TestExpiryJob_MemberLookupFailureStillSendsEmail(t *testing.T) {
	subRepo := repository.NewMockSubscriptionRepository()
	in3 := time.Now().Add(3 * 24 * time.Hour)
	seedSub(t, subRepo, "sub-soon", "tenant-1", &in3)

	email := &recordNotifier{typ: notify.ChannelEmail}
	wsC := &recordNotifier{typ: notify.ChannelWebSocket}
	useRecorders(t, email, wsC)
	job := newJobWith(subRepo, &fakeTenants{}, &fakeMembers{err: errors.New("用户库不可用")})

	require.NoError(t, job.RunOnce(context.Background()))
	require.Equal(t, 1, email.calls, "成员查询失败仍应发出邮件提醒")
	require.Equal(t, 0, wsC.calls, "拿不到成员 ID 时不应空转 WS")
	require.NotNil(t, subRepo.RemindedAtOf("sub-soon"), "邮件已成功，应正常销账")
}

// TestExpiryRemindDelivered 送达判定的口径：注册了邮件渠道就必须邮件成功；
// 未配置邮件（本项目默认）时退化为看是否有任一渠道成功，避免每轮反复推送骚扰在线用户。
func TestExpiryRemindDelivered(t *testing.T) {
	cases := []struct {
		name    string
		results []notify.SendResult
		want    bool
	}{
		{"空结果不算送达", nil, false},
		{"邮件成功", []notify.SendResult{{Channel: notify.ChannelEmail, Success: true}}, true},
		{"邮件失败不算送达", []notify.SendResult{{Channel: notify.ChannelEmail, Success: false}}, false},
		{"邮件失败即使WS成功也不算", []notify.SendResult{
			{Channel: notify.ChannelEmail, Success: false},
			{Channel: notify.ChannelWebSocket, Success: true},
		}, false},
		{"无邮件渠道时WS成功算送达", []notify.SendResult{{Channel: notify.ChannelWebSocket, Success: true}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, expiryRemindDelivered(c.results))
		})
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
	useRecorders(t, &recordNotifier{typ: notify.ChannelEmail})
	job := newExpiryJobWith(planSvc, subRepo, &fakeTenants{}, &fakeMembers{})

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

// newExpiryDB 建内存 SQLite 并迁移套餐/订阅、租户与用户表。
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
	require.NoError(t, userRepo.AutoMigrate(db))
	return db
}

// seedLateSubscription 写入一条已到期订阅与其所属租户（enabled 状态）
func seedLateSubscription(t *testing.T, db *gorm.DB, tenantID, subID string) {
	t.Helper()
	name := "到期租户"
	email := "late@example.com"
	require.NoError(t, db.Create(&tenantRepo.TenantPO{
		ID: tenantID, Name: &name, Domain: tenantID + ".example.com",
		Status: tenantModel.StatusEnabled, ContactEmail: &email,
	}).Error)

	late := time.Now().Add(-24 * time.Hour)
	require.NoError(t, db.Create(&repository.SubscriptionPO{
		ID: subID, TenantID: tenantID, PlanID: "p-1",
		Status: planModel.SubscriptionActive, StartedAt: time.Now().Add(-48 * time.Hour), ExpiresAt: &late,
	}).Error)
}

// TestNewExpiryJob_WiresTenantDisabler 锁住生产装配线：NewExpiryJob 必须把禁用租户的回调注入 service。
// 这一行接线一旦被删，订阅仍会被翻转成 expired，但租户停在 enabled 且再无人重扫（静默免费续期），
// 而上面的 service 层用例用的是自建注入，盖不到这个位置。
func TestNewExpiryJob_WiresTenantDisabler(t *testing.T) {
	db := newExpiryDB(t)
	ctx := context.Background()
	seedLateSubscription(t, db, "t-late", "sub-late")

	email := &recordNotifier{typ: notify.ChannelEmail}
	useRecorders(t, email)

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

// TestNewExpiryJob_WiresMemberDirectory 锁住成员查询的生产装配线：
// NewExpiryJob 必须把 user 仓库接进任务，否则 memberIDs 永远为空、WS 分支永不触发，
// 而未配置邮件的默认部署就变成一条提醒也发不出去（且不再报错，只是静默不发）。
func TestNewExpiryJob_WiresMemberDirectory(t *testing.T) {
	db := newExpiryDB(t)
	ctx := context.Background()

	in3 := time.Now().Add(3 * 24 * time.Hour)
	name := "临期租户"
	email := "soon@example.com"
	require.NoError(t, db.Create(&tenantRepo.TenantPO{
		ID: "t-soon", Name: &name, Domain: "soon.example.com",
		Status: tenantModel.StatusEnabled, ContactEmail: &email,
	}).Error)
	require.NoError(t, db.Create(&repository.SubscriptionPO{
		ID: "sub-soon", TenantID: "t-soon", PlanID: "p-1",
		Status: planModel.SubscriptionActive, StartedAt: time.Now().Add(-48 * time.Hour), ExpiresAt: &in3,
	}).Error)
	require.NoError(t, db.Create(&userRepo.UserPO{
		ID: "u-1", TenantID: "t-soon", Username: "alice", Password: "x", Status: 1,
	}).Error)

	wsC := &recordNotifier{typ: notify.ChannelWebSocket}
	emailC := &recordNotifier{typ: notify.ChannelEmail}
	useRecorders(t, emailC, wsC)

	require.NoError(t, NewExpiryJob(db).RunOnce(ctx))

	require.Equal(t, 1, wsC.calls, "接线丢失则 memberIDs 恒为空，WS 分支永不执行")
	require.Equal(t, []string{"u-1"}, wsC.msgs[0].Recipients, "WS 收件人必须是租户成员的用户 ID")
}

// TestExpiryJob_ReminderSurvivesRestart 提醒状态必须跨重启生效：
// 用同一个库构造第二个任务实例（等价于服务重启后新起的 job），
// 它不得对同一订阅重发提醒。旧的进程内 notified map 正是此处失守——每次部署都重推一遍。
func TestExpiryJob_ReminderSurvivesRestart(t *testing.T) {
	db := newExpiryDB(t)
	ctx := context.Background()

	in3 := time.Now().Add(3 * 24 * time.Hour)
	name := "临期租户"
	email := "soon@example.com"
	require.NoError(t, db.Create(&tenantRepo.TenantPO{
		ID: "t-soon", Name: &name, Domain: "soon.example.com",
		Status: tenantModel.StatusEnabled, ContactEmail: &email,
	}).Error)
	require.NoError(t, db.Create(&repository.SubscriptionPO{
		ID: "sub-soon", TenantID: "t-soon", PlanID: "p-1",
		Status: planModel.SubscriptionActive, StartedAt: time.Now().Add(-48 * time.Hour), ExpiresAt: &in3,
	}).Error)

	emailC := &recordNotifier{typ: notify.ChannelEmail}
	useRecorders(t, emailC)

	require.NoError(t, NewExpiryJob(db).RunOnce(ctx))
	require.Equal(t, 1, emailC.calls)

	var sub repository.SubscriptionPO
	require.NoError(t, db.First(&sub, "id = ?", "sub-soon").Error)
	require.NotNil(t, sub.ExpiryRemindedAt, "提醒时间必须落库，否则重启后重发")

	// 模拟重启：全新实例共用同一数据库
	require.NoError(t, NewExpiryJob(db).RunOnce(ctx))
	require.Equal(t, 1, emailC.calls, "重启后同一订阅不得重发提醒")
}
