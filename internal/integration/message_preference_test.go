package integration

import (
	"context"
	"net/http"
	"testing"

	"meteorx/internal/modules/message/dto"
	"meteorx/internal/modules/message/model"
	messagerepo "meteorx/internal/modules/message/repository"
	"meteorx/internal/notify"
	"meteorx/internal/testkit"

	"github.com/stretchr/testify/require"
)

// prefItem 在偏好列表里按分类取一项。
func prefItem(t *testing.T, resp *dto.PreferencesResp, category string) dto.PreferenceItem {
	t.Helper()
	for _, it := range resp.Items {
		if it.Category == category {
			return it
		}
	}
	t.Fatalf("偏好列表缺少分类 %s", category)
	return dto.PreferenceItem{}
}

// TestInteg_MessagePreference_MutedCategoryNotDelivered 端到端锁定消息偏好：
// 关闭某个分类后该类消息不再落库，其余分类不受影响；重新开启即恢复。
// 同时锁定「安全告警不可关闭」与非法分类的 400 语义。
func TestInteg_MessagePreference_MutedCategoryNotDelivered(t *testing.T) {
	// 预置全局通知管理器：bootstrap.InitRouter 会把站内消息服务装配进它
	prev := notify.GetGlobalManager()
	mgr := notify.NewManager(nil, "", "", nil)
	notify.SetGlobalManager(mgr)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })

	app := testkit.NewTestApp(t)
	auth := app.AuthHeader("u-pref", "t-1", []string{"member"})
	ctx := context.Background()

	// 1) 默认偏好：全部分类都在列表里，安全告警标记为强制接收
	w := app.Do(t, http.MethodGet, "/api/v1/messages/preferences", "", auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var prefs dto.PreferencesResp
	testkit.DecodeData(t, w, &prefs)
	require.Len(t, prefs.Items, len(model.AllCategories()))
	require.False(t, prefItem(t, &prefs, model.CategoryTask).Muted, "默认应全部接收")
	require.True(t, prefItem(t, &prefs, model.CategoryAudit).Mandatory)

	// 2) 关闭任务分类，响应直接回最新视图
	w = app.Do(t, http.MethodPut, "/api/v1/messages/preferences", `{"category":"task","muted":true}`, auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	prefs = dto.PreferencesResp{}
	testkit.DecodeData(t, w, &prefs)
	require.True(t, prefItem(t, &prefs, model.CategoryTask).Muted)

	// 3) 关闭后任务提醒不再落库
	mgr.NotifyTaskReminder(ctx, "u-pref", "任务到期", "内容", "task_due",
		map[string]interface{}{"task_id": "tk-pref-1"})

	w = app.Do(t, http.MethodGet, "/api/v1/messages/unread-count", "", auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var unread dto.UnreadCountResp
	testkit.DecodeData(t, w, &unread)
	require.Zero(t, unread.Total, "关闭接收的分类不应产生未读消息")

	// 4) 分类隔离：套餐提醒照常送达
	mgr.NotifySubscriptionExpiry(ctx, "t-1", "租户A", 3, "", []string{"u-pref"})
	unread = dto.UnreadCountResp{}
	w = app.Do(t, http.MethodGet, "/api/v1/messages/unread-count", "", auth)
	require.Equal(t, http.StatusOK, w.Code)
	testkit.DecodeData(t, w, &unread)
	require.Equal(t, int64(1), unread.Total)
	require.Equal(t, int64(1), unread.ByCategory[model.CategoryPlan])

	// 5) 重新开启后恢复投递（换 task_id 以避开去重键）
	w = app.Do(t, http.MethodPut, "/api/v1/messages/preferences", `{"category":"task","muted":false}`, auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	mgr.NotifyTaskReminder(ctx, "u-pref", "任务到期", "内容", "task_due",
		map[string]interface{}{"task_id": "tk-pref-2"})

	unread = dto.UnreadCountResp{}
	w = app.Do(t, http.MethodGet, "/api/v1/messages/unread-count", "", auth)
	require.Equal(t, http.StatusOK, w.Code)
	testkit.DecodeData(t, w, &unread)
	require.Equal(t, int64(2), unread.Total, "重新开启后应恢复投递")

	// 6) 安全告警不可关闭：400，且不留下任何关闭记录
	w = app.Do(t, http.MethodPut, "/api/v1/messages/preferences", `{"category":"audit","muted":true}`, auth)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// 7) 即使库里存在历史关闭记录（如策略上线前写入 / 人工改库），告警也必须送达
	require.NoError(t, app.DB.Create(&messagerepo.UserMessagePreferencePO{
		ID: "pref-audit-muted", UserID: "u-pref", Category: model.CategoryAudit, Muted: true,
	}).Error)
	mgr.NotifyAlert(ctx, notify.AlertNotice{
		ID: "al-pref-1", RuleName: "高风险入库告警", Content: "测试告警",
		Priority: notify.PriorityUrgent, UserIDs: []string{"u-pref"},
	})
	w = app.Do(t, http.MethodGet, "/api/v1/messages?category=audit&page=1&page_size=10", "", auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var page messagePage
	testkit.DecodeData(t, w, &page)
	require.Equal(t, 1, page.Pagination.Total, "安全告警必须强制送达")

	// 8) 非法分类 / 未登录
	w = app.Do(t, http.MethodPut, "/api/v1/messages/preferences", `{"category":"nope","muted":true}`, auth)
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = app.Do(t, http.MethodGet, "/api/v1/messages/preferences", "", "")
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestInteg_MessagePreference_IsolatedByUser 偏好按收件人隔离：
// 同一事件的多个收件人中，只有关闭了该分类的人被过滤。
func TestInteg_MessagePreference_IsolatedByUser(t *testing.T) {
	prev := notify.GetGlobalManager()
	mgr := notify.NewManager(nil, "", "", nil)
	notify.SetGlobalManager(mgr)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })

	app := testkit.NewTestApp(t)
	mutedAuth := app.AuthHeader("u-mute", "t-1", []string{"member"})
	normalAuth := app.AuthHeader("u-normal", "t-1", []string{"member"})

	w := app.Do(t, http.MethodPut, "/api/v1/messages/preferences", `{"category":"task","muted":true}`, mutedAuth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 一键提醒直接发给两个收件人，过滤必须按人判断
	for _, uid := range []string{"u-mute", "u-normal"} {
		mgr.NotifyTaskReminder(context.Background(), uid, "任务到期", "内容", "task_due",
			map[string]interface{}{"task_id": "tk-iso"})
	}

	var page messagePage
	w = app.Do(t, http.MethodGet, "/api/v1/messages?page=1&page_size=10", "", mutedAuth)
	require.Equal(t, http.StatusOK, w.Code)
	testkit.DecodeData(t, w, &page)
	require.Zero(t, page.Pagination.Total, "关闭接收的收件人不应收到")

	page = messagePage{}
	w = app.Do(t, http.MethodGet, "/api/v1/messages?page=1&page_size=10", "", normalAuth)
	require.Equal(t, http.StatusOK, w.Code)
	testkit.DecodeData(t, w, &page)
	require.Equal(t, 1, page.Pagination.Total, "未关闭的收件人应正常收到")
}
