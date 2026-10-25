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
		m.NotifySubscriptionExpiry(context.Background(), "租户A", 3, "", []string{"u1"})
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
	require.Equal(t, "/task", got.Link, "收件箱条目需要可跳转")
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

	m.NotifySubscriptionExpiry(context.Background(), "租户A", 5, "", []string{"u1", "u2"})
	require.Len(t, inbox.got, 1)
	require.Equal(t, []string{"u1", "u2"}, inbox.got[0].UserIDs)
	require.Equal(t, "subscription_expiry", inbox.got[0].Type)
	require.Equal(t, "/system/plan", inbox.got[0].Link)
	require.Equal(t, "5", inbox.got[0].SourceID)

	// 无成员时不落库
	inbox.got = nil
	m.NotifySubscriptionExpiry(context.Background(), "租户A", 5, "admin@example.com", nil)
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
}

// TestInbox_DedupeKeyTenantScoped 订阅临期与注销审批的去重键含租户与状态，
// 以便同一租户不同剩余天数 / 不同审批结果各自可见。
func TestInbox_DedupeKeyTenantScoped(t *testing.T) {
	today := time.Now().Format("20060102")
	inbox := &recordingInbox{}
	m := NewManager(nil, "", "", nil)
	m.SetInbox(inbox)

	m.NotifySubscriptionExpiry(context.Background(), "租户A", 3, "", []string{"u1"})
	m.NotifyCancelRequestStatus(context.Background(), "租户A", "approved", "", "", []string{"u1"})

	require.Len(t, inbox.got, 2)
	require.Equal(t, "subscription:租户A:3:"+today, inbox.got[0].DedupeKey)
	require.Equal(t, "cancel_request:租户A:approved:"+today, inbox.got[1].DedupeKey)
}
