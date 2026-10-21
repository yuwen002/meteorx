package integration

import (
	"context"
	"net/http"
	"testing"

	"meteorx/internal/notify"
	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureChannel 录制指定渠道收到的消息；CompositeNotifier 并发投递后
// 等到全部返回，因此 HTTP 响应返回时切片已稳定可读。
type captureChannel struct {
	channel notify.ChannelType
	msgs    []*notify.Message
}

func (c *captureChannel) Type() notify.ChannelType { return c.channel }
func (c *captureChannel) Name() string             { return "capture-" + string(c.channel) }
func (c *captureChannel) Send(_ context.Context, msg *notify.Message) error {
	c.msgs = append(c.msgs, msg)
	return nil
}

// installCaptureManager 用录制型渠道接管全局管理器（InitRouter 不装配通知管理器，
// 默认回调在全局为 nil 时直接返回），并在用例结束后恢复原值。
func installCaptureManager(t *testing.T) (wsCap, mailCap, hookCap *captureChannel) {
	t.Helper()
	wsCap = &captureChannel{channel: notify.ChannelWebSocket}
	mailCap = &captureChannel{channel: notify.ChannelEmail}
	hookCap = &captureChannel{channel: notify.ChannelWebhook}

	mgr := notify.NewManager(nil, "", "", nil)
	mgr.AddChannel(wsCap)
	mgr.AddChannel(mailCap)
	mgr.AddChannel(hookCap)

	prev := notify.GetGlobalManager()
	notify.SetGlobalManager(mgr)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })
	return wsCap, mailCap, hookCap
}

// TestInteg_Announcement_PublishViaStatusNotifiesTargetTenant 后台行内「发布」
// 按钮走 PUT /{id}/status：必须触发一次通知，且定向公告只把目标租户交给 WS 渠道
// （旧实现既不触发、也把 scope 丢掉变成全量广播）；重复发布与下架不得再推。
func TestInteg_Announcement_PublishViaStatusNotifiesTargetTenant(t *testing.T) {
	app := testkit.NewTestApp(t)
	wsCap, mailCap, hookCap := installCaptureManager(t)
	sa := app.SuperadminHeader("sa", "t-1")

	body := `{"title":"定向升级通知","content":"仅 A 公司可见","scope":"tenant",` +
		`"target_tenant_id":"t-a","status":0}`
	w := app.Do(t, http.MethodPost, "/api/v1/admin/announcements", body, sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	id := decodeID(t, w)
	require.Empty(t, wsCap.msgs, "草稿不推送")

	w = app.Do(t, http.MethodPut, "/api/v1/admin/announcements/"+id+"/status", `{"status":1}`, sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, wsCap.msgs, 1, "草稿转发布必须推送一次")
	assert.Equal(t, "t-a", wsCap.msgs[0].TenantID, "定向公告必须携带目标租户，不得广播")
	assert.Empty(t, wsCap.msgs[0].Recipients, "租户定向不靠收件人列表")
	require.Len(t, hookCap.msgs, 1, "平台 Webhook 照常推送")
	assert.Empty(t, mailCap.msgs, "公告不得空跑邮件渠道")

	// 定向可见性：目标租户看得到，其它租户看不到（必须在再次下架之前查，否则不可见）
	w = app.Do(t, http.MethodGet, "/api/v1/announcements", "", app.AuthHeader("m-a", "t-a", []string{"member"}))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), id, "目标租户应看到定向公告")
	w = app.Do(t, http.MethodGet, "/api/v1/announcements", "", app.AuthHeader("m-b", "t-b", []string{"member"}))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), id, "其它租户不得看到定向公告")

	// 重复发布与下架都不再推送
	w = app.Do(t, http.MethodPut, "/api/v1/admin/announcements/"+id+"/status", `{"status":1}`, sa)
	require.Equal(t, http.StatusOK, w.Code)
	w = app.Do(t, http.MethodPut, "/api/v1/admin/announcements/"+id+"/status", `{"status":2}`, sa)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, wsCap.msgs, 1, "重复发布与下架不得重复推送")
}

// TestInteg_Announcement_CreatePublishedPushesOnce 创建即发布（status=1）走 Create 路径：
// 同样只推一次并携带目标租户。
func TestInteg_Announcement_CreatePublishedPushesOnce(t *testing.T) {
	app := testkit.NewTestApp(t)
	wsCap, _, _ := installCaptureManager(t)
	sa := app.SuperadminHeader("sa", "t-1")

	body := `{"title":"定向维护","content":"A 公司专属","scope":"tenant",` +
		`"target_tenant_id":"t-a","status":1}`
	w := app.Do(t, http.MethodPost, "/api/v1/admin/announcements", body, sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Len(t, wsCap.msgs, 1)
	assert.Equal(t, "t-a", wsCap.msgs[0].TenantID)
	assert.Equal(t, "定向维护", wsCap.msgs[0].Title)
}

// TestInteg_Announcement_TenantScopeMissingTargetReturns400 scope=tenant 缺目标租户
// 必须回 400（而不是 500 或落一条无人可见的公告），创建与更新两条路径同口径。
func TestInteg_Announcement_TenantScopeMissingTargetReturns400(t *testing.T) {
	app := testkit.NewTestApp(t)
	sa := app.SuperadminHeader("sa", "t-1")

	w := app.Do(t, http.MethodPost, "/api/v1/admin/announcements",
		`{"title":"x","content":"y","scope":"tenant"}`, sa)
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "target_tenant_id")

	w = app.Do(t, http.MethodPost, "/api/v1/admin/announcements",
		`{"title":"草稿","content":"y","scope":"all","status":0}`, sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	id := decodeID(t, w)

	w = app.Do(t, http.MethodPut, "/api/v1/admin/announcements/"+id,
		`{"title":"草稿","content":"y","scope":"tenant","status":0}`, sa)
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "target_tenant_id")
}

// TestInteg_Notification_AnnouncementCRUD 整体验证公告模块：
// 超管创建/列表/详情/改状态/删除闭环；非超管创建被 403；租户侧公告列表可访问。
func TestInteg_Notification_AnnouncementCRUD(t *testing.T) {
	app := testkit.NewTestApp(t)
	sa := app.SuperadminHeader("sa", "t-1")
	member := app.AuthHeader("m", "t-1", []string{"member"})

	body := `{"title":"系统升级","content":"将于今晚升级","scope":"all","status":1}`
	w := app.Do(t, http.MethodPost, "/api/v1/admin/announcements", body, sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	id := decodeID(t, w)

	w = app.Do(t, http.MethodGet, "/api/v1/admin/announcements", "", sa)
	assert.Contains(t, w.Body.String(), id)
	w = app.Do(t, http.MethodGet, "/api/v1/admin/announcements/"+id, "", sa)
	assert.Equal(t, http.StatusOK, w.Code)
	w = app.Do(t, http.MethodPut, "/api/v1/admin/announcements/"+id+"/status", `{"status":2}`, sa)
	assert.Equal(t, http.StatusOK, w.Code)
	w = app.Do(t, http.MethodDelete, "/api/v1/admin/announcements/"+id, "", sa)
	assert.Equal(t, http.StatusOK, w.Code)

	// 非超管创建公告 → 403
	w = app.Do(t, http.MethodPost, "/api/v1/admin/announcements", body, member)
	assert.Equal(t, http.StatusForbidden, w.Code, "非超管创建公告应 403")

	// 租户侧公告列表（公开侧，需登录）→ 200
	w = app.Do(t, http.MethodGet, "/api/v1/announcements", "", member)
	assert.Equal(t, http.StatusOK, w.Code)
}
