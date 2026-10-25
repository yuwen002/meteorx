package notify

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// recordingInbox 记录落库请求，可选择返回错误以验证失败隔离。
type recordingInbox struct {
	got []*InboxMessage
	err error
}

func (r *recordingInbox) Deliver(_ context.Context, msg *InboxMessage) error {
	if r.err != nil {
		return r.err
	}
	r.got = append(r.got, msg)
	return nil
}

// TestInbox_NotSetIsSafe 未装配 inbox 时所有通知路径都必须保持原有行为。
func TestInbox_NotSetIsSafe(t *testing.T) {
	m := NewManager(nil, "", "", nil)
	require.NotPanics(t, func() {
		m.NotifyTaskReminder(context.Background(), "u1", "标题", "内容", "task_due", map[string]interface{}{"task_id": "tk1"})
		m.NotifyAnnouncement(context.Background(), "公告", "内容", "t1")
		m.NotifySubscriptionExpiry(context.Background(), "t1", "租户A", 3, "", []string{"u1"})
	})
}

// TestInbox_TaskReminderPersistedEvenWithoutWS 任务提醒的落库不依赖 WS 渠道：
// 用户离线时没有 WS 结果，但收件箱必须仍有这条提醒。
func TestInbox_TaskReminderPersistedEvenWithoutWS(t *testing.T) {
	inbox := &recordingInbox{}
	m := NewManager(nil, "", "", nil)
	m.SetInbox(inbox)

	results := m.NotifyTaskReminder(context.Background(), "u1", "任务即将到期", "请尽快处理", "task_due",
		map[string]interface{}{"task_id": "tk1", "due_at": "2026-10-24"})

	require.Empty(t, results, "无 WS 渠道时返回空结果")
	require.Len(t, inbox.got, 1)
	got := inbox.got[0]
	require.Equal(t, []string{"u1"}, got.UserIDs)
	require.Equal(t, "task_due", got.Type)
	require.Equal(t, "/task?task_id=tk1", got.Link, "链接需带上任务 ID，点开可定位到具体任务")
	require.Equal(t, "task", got.SourceType)
	require.Equal(t, "tk1", got.SourceID)
	require.Equal(t, "任务即将到期", got.Title)
}

func TestInbox_TaskReminderSkippedForEmptyRecipient(t *testing.T) {
	inbox := &recordingInbox{}
	m := NewManager(nil, "", "", nil)
	m.SetInbox(inbox)

	require.Empty(t, m.NotifyTaskReminder(context.Background(), "", "t", "c", "task_due", nil))
	require.Empty(t, inbox.got, "无收件人时不应写无人可见的行")
}

// TestInbox_AnnouncementOnlyTenantScoped 平台级公告不展开落库（写放大），租户定向公告才落。
func TestInbox_AnnouncementOnlyTenantScoped(t *testing.T) {
	inbox := &recordingInbox{}
	m := NewManager(nil, "", "", nil)
	m.SetInbox(inbox)

	m.NotifyAnnouncement(context.Background(), "全平台公告", "内容", "")
	require.Empty(t, inbox.got, "平台级公告由公告列表承载，不逐人写行")

	m.NotifyAnnouncement(context.Background(), "租户公告", "内容", "t1")
	require.Len(t, inbox.got, 1)
	require.Equal(t, "t1", inbox.got[0].TenantID)
	require.Equal(t, "announcement", inbox.got[0].SourceType)
}

func TestInbox_SubscriptionExpiryPersisted(t *testing.T) {
	inbox := &recordingInbox{}
	m := NewManager(nil, "", "", nil)
	m.SetInbox(inbox)

	m.NotifySubscriptionExpiry(context.Background(), "t1", "租户A", 5, "", []string{"u1", "u2"})
	require.Len(t, inbox.got, 1)
	require.Equal(t, []string{"u1", "u2"}, inbox.got[0].UserIDs)
	require.Equal(t, "subscription_expiry", inbox.got[0].Type)
	require.Equal(t, "/system/tenant?tenant_id=t1", inbox.got[0].Link, "链接需带租户 ID，点开可定位到该租户的套餐配置")
	require.Equal(t, "5", inbox.got[0].SourceID)

	// 取不到租户 ID 时不拼出空 query
	inbox.got = nil
	m.NotifySubscriptionExpiry(context.Background(), "", "租户A", 5, "", []string{"u1"})
	require.Len(t, inbox.got, 1)
	require.Equal(t, "/system/tenant", inbox.got[0].Link, "无 tenant_id 时不得拼出空 query")

	// 无成员时不落库
	inbox.got = nil
	m.NotifySubscriptionExpiry(context.Background(), "t1", "租户A", 5, "admin@example.com", nil)
	require.Empty(t, inbox.got)
}

func TestInbox_CancelRequestPersisted(t *testing.T) {
	inbox := &recordingInbox{}
	m := NewManager(nil, "", "", nil)
	m.SetInbox(inbox)

	m.NotifyCancelRequestStatus(context.Background(), "租户A", "approved", "ok", "", []string{"u1"})
	require.Len(t, inbox.got, 1)
	require.Equal(t, "cancel_request", inbox.got[0].Type)
	require.Equal(t, "tenant_cancel_request", inbox.got[0].SourceType)
	require.Equal(t, "approved", inbox.got[0].SourceID)
}

// TestInbox_ErrorDoesNotAffectDelivery 落库失败只告警，不得改变渠道送达结果判定。
func TestInbox_ErrorDoesNotAffectDelivery(t *testing.T) {
	inbox := &recordingInbox{err: errors.New("db down")}
	m := NewManager(nil, "", "", nil)
	m.SetInbox(inbox)

	require.NotPanics(t, func() {
		results := m.NotifyTaskReminder(context.Background(), "u1", "t", "c", "task_due", nil)
		require.Empty(t, results, "落库失败不产生额外的送达结果")
	})
}

// TestInbox_DedupeKeyPerDayAndReminderType 去重键按「业务事件 + 自然日」构造：
// 同一天同一任务的同类提醒得到同一键（重复投递只落一条），
// 不同提醒类型不得互相合并；无法定位任务的事件不参与去重。
func TestInbox_DedupeKeyPerDayAndReminderType(t *testing.T) {
	today := time.Now().Format("20060102")
	inbox := &recordingInbox{}
	m := NewManager(nil, "", "", nil)
	m.SetInbox(inbox)

	extra := map[string]interface{}{"task_id": "tk1"}
	m.NotifyTaskReminder(context.Background(), "u1", "t", "c", "task_due", extra)
	m.NotifyTaskReminder(context.Background(), "u1", "t", "c", "task_due", extra)
	m.NotifyTaskReminder(context.Background(), "u1", "t", "c", "task_assigned", extra)
	m.NotifyTaskReminder(context.Background(), "u1", "t", "c", "task_due", nil)

	require.Len(t, inbox.got, 4)
	require.Equal(t, "task:tk1:task_due:"+today, inbox.got[0].DedupeKey)
	require.Equal(t, inbox.got[0].DedupeKey, inbox.got[1].DedupeKey, "同一天同一提醒类型的重试应命中去重")
	require.NotEqual(t, inbox.got[0].DedupeKey, inbox.got[2].DedupeKey, "不同提醒类型不得互相合并")
	require.Empty(t, inbox.got[3].DedupeKey, "无法定位任务的事件不参与去重")
	require.Equal(t, "/task", inbox.got[3].Link, "无 task_id 时不得拼出空 query")
}

// TestInbox_AlertPersistedForRecipients 告警的收件箱收件人只能是用户 ID：
// 规则配置的邮箱由调用方反查成用户 ID 后传入；告警正文含用户行为明细，
// 不得在这里展开租户或广播。分类固定 audit，去重键按告警 ID（同一告警只落一条）。
func TestInbox_AlertPersistedForRecipients(t *testing.T) {
	inbox := &recordingInbox{}
	m := NewManager(nil, "", "", nil)
	m.SetInbox(inbox)

	for i := 0; i < 2; i++ { // 渠道重试 / 多副本会把同一条告警投递多次
		m.NotifyAlert(context.Background(), AlertNotice{
			ID: "al-1", RuleName: "高风险操作", Content: "用户 alice 删除了资源",
			Priority: PriorityUrgent, Emails: []string{"a@x.com"}, UserIDs: []string{"u-admin"},
		})
	}

	require.Len(t, inbox.got, 2)
	got := inbox.got[0]
	require.Equal(t, []string{"u-admin"}, got.UserIDs, "收件箱只认用户 ID，不得混入邮箱")
	require.Equal(t, "alert", got.Type)
	require.Equal(t, "audit", got.Category)
	require.Equal(t, "audit_alert", got.SourceType)
	require.Equal(t, "al-1", got.SourceID)
	require.Equal(t, "/system/audit/alert?alert_id=al-1", got.Link, "链接需带上告警 ID，点开可定位到具体记录")
	require.Empty(t, got.TenantID, "平台级告警不归属任何租户")
	require.Equal(t, "alert:al-1", got.DedupeKey)
	require.Equal(t, got.DedupeKey, inbox.got[1].DedupeKey, "同一告警的重复投递共享去重键")

	// 无账号收件人（wiki 运维告警）不落库：不得因空收件人触发广播
	inbox.got = nil
	m.NotifyAlert(context.Background(), AlertNotice{RuleName: "清理失败", Content: "c", Priority: PriorityHigh})
	require.Empty(t, inbox.got)
}

// TestInbox_AlertLinkWithoutID 无告警记录 ID 的通知（如定时任务自检）不拼 query：
// 带一个空 alert_id 会让前端把详情弹窗打开成空壳。
func TestInbox_AlertLinkWithoutID(t *testing.T) {
	inbox := &recordingInbox{}
	m := NewManager(nil, "", "", nil)
	m.SetInbox(inbox)

	m.NotifyAlert(context.Background(), AlertNotice{
		RuleName: "清理失败", Content: "c", Priority: PriorityHigh, UserIDs: []string{"u-ops"},
	})

	require.Len(t, inbox.got, 1)
	require.Equal(t, "/system/audit/alert", inbox.got[0].Link, "无 ID 时不得拼出空 alert_id")
}

// TestInbox_DedupeKeyTenantScoped 订阅临期与注销审批的去重键含租户与状态，
// 以便同一租户不同剩余天数 / 不同审批结果各自可见。
func TestInbox_DedupeKeyTenantScoped(t *testing.T) {
	today := time.Now().Format("20060102")
	inbox := &recordingInbox{}
	m := NewManager(nil, "", "", nil)
	m.SetInbox(inbox)

	m.NotifySubscriptionExpiry(context.Background(), "t1", "租户A", 3, "", []string{"u1"})
	m.NotifyCancelRequestStatus(context.Background(), "租户A", "approved", "", "", []string{"u1"})

	require.Len(t, inbox.got, 2)
	require.Equal(t, "subscription:租户A:3:"+today, inbox.got[0].DedupeKey)
	require.Equal(t, "cancel_request:租户A:approved:"+today, inbox.got[1].DedupeKey)
}
