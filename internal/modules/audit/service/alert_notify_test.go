package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"
	"meteorx/internal/notify"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// recNotifier 记录投递到某渠道的消息，用于断言收件人语义（与 tenant 层同模式）。
type recNotifier struct {
	typ   notify.ChannelType
	err   error
	msgs  []*notify.Message
	calls int32
}

func (r *recNotifier) Type() notify.ChannelType { return r.typ }
func (r *recNotifier) Name() string             { return "record-" + string(r.typ) }
func (r *recNotifier) Send(_ context.Context, msg *notify.Message) error {
	atomic.AddInt32(&r.calls, 1)
	r.msgs = append(r.msgs, msg)
	return r.err
}

// useNotifyRecorders 把指定渠道注册为全局通知管理器，测试结束后恢复原值
func useNotifyRecorders(t *testing.T, channels ...*recNotifier) {
	t.Helper()
	useNotifyRecordersWithInbox(t, nil, channels...)
}

// recInbox 记录站内信落库请求（告警落收件箱的断言面向）。
type recInbox struct {
	got []*notify.InboxMessage
}

func (r *recInbox) Deliver(_ context.Context, msg *notify.InboxMessage) error {
	r.got = append(r.got, msg)
	return nil
}

// useNotifyRecordersWithInbox 注册渠道并装配站内信落库能力（inbox 可为 nil）。
func useNotifyRecordersWithInbox(t *testing.T, inbox notify.InboxWriter, channels ...*recNotifier) {
	t.Helper()
	prev := notify.GetGlobalManager()
	m := notify.NewManager(nil, "", "", nil)
	for _, c := range channels {
		m.AddChannel(c)
	}
	if inbox != nil {
		m.SetInbox(inbox)
	}
	notify.SetGlobalManager(m)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })
}

// stubRecipients 邮箱 → 用户 ID 的桩：未登记的邮箱返回错误，模拟「不属于任何账号」。
type stubRecipients map[string]string

func (s stubRecipients) UserIDByEmail(_ context.Context, email string) (string, error) {
	if id, ok := s[email]; ok {
		return id, nil
	}
	return "", errors.New("user not found")
}

// useNoNotifyManager 摘除全局通知管理器（真实库用例只验证直发渠道与落库回写）
func useNoNotifyManager(t *testing.T) {
	t.Helper()
	prev := notify.GetGlobalManager()
	notify.SetGlobalManager(nil)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })
}

// seedHighRiskRule 建一条匹配 high 风险等级的邮件告警规则
func seedHighRiskRule(t *testing.T, svc *AlertService, channels, targets []string) {
	t.Helper()
	_, err := svc.CreateRule(context.Background(), dto.CreateAlertRuleReq{
		Name:            "高风险操作告警",
		Enabled:         true,
		TriggerType:     model.TriggerTypeRiskLevel,
		TriggerValue:    model.RiskHigh,
		NotifyChannels:  channels,
		NotifyTargets:   targets,
		CooldownMinutes: 30,
	})
	require.NoError(t, err)
}

func highRiskAuditLog() *model.AuditLog {
	return &model.AuditLog{
		ID: "al-notify-001", UserID: "u-001", Username: "alice", TenantID: "t-001",
		Module: "auth", Action: "delete", Result: "success",
		RiskLevel: model.RiskHigh, CreatedAt: time.Now(),
	}
}

// newAlertDB 具名 shared-cache 内存库（与 testkit/plan 同口径），迁移审计三表
func newAlertDB(t *testing.T) *gorm.DB {
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

// TestSendNotifications_DeliveredMarksNotified 有渠道真实送达时才回写 notified，
// 且回写走 MarkAlertNotified（UPDATE）。旧写法不判定送达、也不 UPDATE——它拿同一
// ID 再次 CreateAlert（INSERT），主键冲突的错误还被 `_ =` 吞掉。
func TestSendNotifications_DeliveredMarksNotified(t *testing.T) {
	repo := repository.NewMockAlertRuleRepository()
	svc := NewAlertService(repo, nil)
	seedHighRiskRule(t, svc, []string{model.NotifyChannelEmail}, []string{"admin@example.com"})

	email := &recNotifier{typ: notify.ChannelEmail}
	useNotifyRecorders(t, email)

	require.NoError(t, svc.EvaluateAndAlert(context.Background(), highRiskAuditLog()))

	require.Len(t, repo.NotifiedIDs, 1, "送达后必须以 UPDATE 回写 notified")
	alerts, _, err := repo.ListAlerts(context.Background(), 1, 10, "", "", "")
	require.NoError(t, err)
	require.Len(t, alerts, 1, "告警记录只应有 1 条（首插），不得二次 INSERT")
	require.True(t, alerts[0].Notified)
	require.False(t, alerts[0].NotifyTime.IsZero())
}

// TestSendNotifications_NoDeliveryDoesNotMark 所有渠道都失败时不得标记已通知：
// 标记了也不会重驱（冷却按 created_at 判定），但会让看板 notified_count/未通知数
// 永久性失真，故障被登记成「已处理」。
func TestSendNotifications_NoDeliveryDoesNotMark(t *testing.T) {
	repo := repository.NewMockAlertRuleRepository()
	svc := NewAlertService(repo, nil) // emailer=nil，直发邮件渠道空转
	seedHighRiskRule(t, svc, []string{model.NotifyChannelEmail}, []string{"admin@example.com"})

	email := &recNotifier{typ: notify.ChannelEmail, err: errors.New("smtp down")}
	useNotifyRecorders(t, email)

	require.NoError(t, svc.EvaluateAndAlert(context.Background(), highRiskAuditLog()))

	require.Empty(t, repo.NotifiedIDs, "一个渠道都没送达时不得回写 notified")
	alerts, _, err := repo.ListAlerts(context.Background(), 1, 10, "", "", "")
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	require.False(t, alerts[0].Notified)
}

// TestSendNotifications_TargetsSplitByType 规则 targets 混装邮箱与 webhook URL 是常态，
// 进全局管理器的收件人必须按类型拆分：URL 不得混入邮件渠道（EmailNotifier 会对
// "http://…" 地址发信失败），告警也不得进 WS（收件人类型不对）。
func TestSendNotifications_TargetsSplitByType(t *testing.T) {
	var hookHits int32
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hookHits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	repo := repository.NewMockAlertRuleRepository()
	svc := NewAlertService(repo, nil)
	seedHighRiskRule(t, svc,
		[]string{model.NotifyChannelEmail, model.NotifyChannelDingTalk},
		[]string{"admin@example.com", hook.URL})

	email := &recNotifier{typ: notify.ChannelEmail}
	useNotifyRecorders(t, email)

	require.NoError(t, svc.EvaluateAndAlert(context.Background(), highRiskAuditLog()))

	require.Equal(t, int32(1), hookHits, "URL 目标由直发渠道消化，须真实 POST")
	require.Equal(t, int32(1), email.calls)
	require.Equal(t, []string{"admin@example.com"}, email.msgs[0].Recipients, "URL 不得混入邮件渠道收件人")
	require.Len(t, repo.NotifiedIDs, 1, "直发成功即算送达，应回写")
}

// TestSendNotifications_AlertReachesInbox 告警的收件箱收件人来自规则里的邮箱：
// 邮箱能反查到账号才落库，未注册账号的外部邮箱不得成为收件人（否则等于把
// 含用户行为明细的告警发给无关账号）。
func TestSendNotifications_AlertReachesInbox(t *testing.T) {
	repo := repository.NewMockAlertRuleRepository()
	svc := NewAlertService(repo, nil)
	svc.SetRecipientDirectory(stubRecipients{"admin@example.com": "u-admin"})
	seedHighRiskRule(t, svc,
		[]string{model.NotifyChannelEmail},
		[]string{"admin@example.com", "ops@external.com"})

	email := &recNotifier{typ: notify.ChannelEmail}
	inbox := &recInbox{}
	useNotifyRecordersWithInbox(t, inbox, email)

	require.NoError(t, svc.EvaluateAndAlert(context.Background(), highRiskAuditLog()))

	require.Len(t, inbox.got, 1, "同一条告警只落一条站内信")
	got := inbox.got[0]
	require.Equal(t, []string{"u-admin"}, got.UserIDs, "只解析出已注册账号的邮箱")
	require.Equal(t, "alert", got.Type)
	require.Equal(t, "audit", got.Category, "告警固定归入审计分类")
	require.Equal(t, "audit_alert", got.SourceType)
	require.NotEmpty(t, got.SourceID, "来源 ID 应为告警记录 ID")
	require.Regexp(t, "^alert:", got.DedupeKey, "去重键按告警 ID，非按自然日")
	require.Equal(t, int32(1), email.calls, "收件箱不得替代邮件渠道")
	require.Len(t, repo.NotifiedIDs, 1, "送达回写不受收件箱影响")
}

// TestSendNotifications_NoRecipientDirectorySkipsInbox 未装配邮箱解析能力时
// 不落收件箱、也不得 panic：其余渠道与销账照常。
func TestSendNotifications_NoRecipientDirectorySkipsInbox(t *testing.T) {
	repo := repository.NewMockAlertRuleRepository()
	svc := NewAlertService(repo, nil)
	seedHighRiskRule(t, svc, []string{model.NotifyChannelEmail}, []string{"admin@example.com"})

	email := &recNotifier{typ: notify.ChannelEmail}
	inbox := &recInbox{}
	useNotifyRecordersWithInbox(t, inbox, email)

	require.NotPanics(t, func() {
		require.NoError(t, svc.EvaluateAndAlert(context.Background(), highRiskAuditLog()))
	})
	require.Empty(t, inbox.got, "未装配解析能力时不得凭空写收件箱")
	require.Len(t, repo.NotifiedIDs, 1)
}

// TestEvaluateAndAlert_RealDBPersistsNotified 真库端到端锁住本用例的主病灶：旧实现
// 的 notified 回写是「同 ID 二次 INSERT」，真实 MySQL/sqlite 上必然主键冲突，错误又被
// `_ =` 丢弃——notified/notify_time 从未落过库，看板的「已通知数」恒为 0、告警列表
// 永远显示未通知。mock 测试因 CreateAlert 只 append 指针而全部盖不住这一环。
func TestEvaluateAndAlert_RealDBPersistsNotified(t *testing.T) {
	var hookHits int32
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hookHits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	db := newAlertDB(t)
	repo := repository.NewAlertRuleRepository(db)
	svc := NewAlertService(repo, nil)
	seedHighRiskRule(t, svc, []string{model.NotifyChannelWebhook}, []string{hook.URL})
	useNoNotifyManager(t)

	ctx := context.Background()
	require.NoError(t, svc.EvaluateAndAlert(ctx, highRiskAuditLog()))

	alerts, total, err := repo.ListAlerts(ctx, 1, 10, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total, "只能有首插的 1 条，二次 INSERT 冲突正是旧回写的死因")
	require.NotEmpty(t, alerts)

	fresh, err := repo.GetAlertByID(ctx, alerts[0].ID)
	require.NoError(t, err)
	require.True(t, fresh.Notified, "送达必须真实落库（UPDATE），而非停在内存对象上")
	require.False(t, fresh.NotifyTime.IsZero(), "notify_time 须随回写落库")
}
