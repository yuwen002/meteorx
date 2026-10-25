package integration

import (
	"context"
	"net/http"
	"testing"

	"meteorx/internal/modules/message/dto"
	"meteorx/internal/notify"
	"meteorx/internal/testkit"

	"github.com/stretchr/testify/require"
)

// messagePage 收件箱分页响应（pagination 由后端统一包装）。
type messagePage struct {
	Data       []dto.MessageResp `json:"data"`
	Pagination struct {
		Total int `json:"total"`
	} `json:"pagination"`
}

// TestInteg_Message_NotificationToInbox 端到端锁定站内信链路：
// 业务通知（任务提醒）→ notify 落库 → 收件箱未读数 / 列表 / 已读 / 删除。
// 同时验证收件箱按收件人隔离：他人既看不到，也不能代为标记已读。
func TestInteg_Message_NotificationToInbox(t *testing.T) {
	// 预置全局通知管理器：bootstrap.InitRouter 会把站内消息服务装配进它
	prev := notify.GetGlobalManager()
	mgr := notify.NewManager(nil, "", "", nil)
	notify.SetGlobalManager(mgr)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })

	app := testkit.NewTestApp(t)
	auth := app.AuthHeader("u-msg", "t-1", []string{"member"})
	other := app.AuthHeader("u-other", "t-1", []string{"member"})

	// 1) 真实业务入口触发通知落库（无 WS 渠道时也应落库）
	mgr.NotifyTaskReminder(context.Background(), "u-msg", "任务即将到期", "请尽快处理", "task_due",
		map[string]interface{}{"task_id": "tk-1"})

	// 2) 未读数
	w := app.Do(t, http.MethodGet, "/api/v1/messages/unread-count", "", auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var unread dto.UnreadCountResp
	testkit.DecodeData(t, w, &unread)
	require.Equal(t, int64(1), unread.Total)
	require.Equal(t, int64(1), unread.ByCategory["task"])

	// 3) 列表：能拿到消息与跳转链接
	w = app.Do(t, http.MethodGet, "/api/v1/messages?page=1&page_size=10", "", auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var page messagePage
	testkit.DecodeData(t, w, &page)
	require.Equal(t, 1, page.Pagination.Total)
	require.Len(t, page.Data, 1)
	msg := page.Data[0]
	require.Equal(t, "任务即将到期", msg.Title)
	require.Equal(t, "/task", msg.Link)
	require.Nil(t, msg.ReadAt)
	msgID := msg.ID

	// 4) 收件人隔离：他人未读数为 0、列表为空
	w = app.Do(t, http.MethodGet, "/api/v1/messages/unread-count", "", other)
	require.Equal(t, http.StatusOK, w.Code)
	testkit.DecodeData(t, w, &unread)
	require.Zero(t, unread.Total)

	w = app.Do(t, http.MethodGet, "/api/v1/messages", "", other)
	require.Equal(t, http.StatusOK, w.Code)
	testkit.DecodeData(t, w, &page)
	require.Zero(t, page.Pagination.Total)

	// 5) 他人不能代为标记已读
	w = app.Do(t, http.MethodPut, "/api/v1/messages/"+msgID+"/read", "", other)
	require.Equal(t, http.StatusNotFound, w.Code, "非本人消息应返回 404，不暴露存在性")

	// 6) 本人标记已读，重复标记幂等
	w = app.Do(t, http.MethodPut, "/api/v1/messages/"+msgID+"/read", "", auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = app.Do(t, http.MethodPut, "/api/v1/messages/"+msgID+"/read", "", auth)
	require.Equal(t, http.StatusOK, w.Code, "重复标记应幂等成功")

	w = app.Do(t, http.MethodGet, "/api/v1/messages/unread-count", "", auth)
	require.Equal(t, http.StatusOK, w.Code)
	testkit.DecodeData(t, w, &unread)
	require.Zero(t, unread.Total)

	// 7) 删除后列表为空
	w = app.Do(t, http.MethodDelete, "/api/v1/messages/"+msgID, "", auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = app.Do(t, http.MethodGet, "/api/v1/messages", "", auth)
	require.Equal(t, http.StatusOK, w.Code)
	testkit.DecodeData(t, w, &page)
	require.Zero(t, page.Pagination.Total)
}

// TestInteg_Message_MarkAllReadByCategory 分类全部已读只清该分类，其他分类未读保留。
func TestInteg_Message_MarkAllReadByCategory(t *testing.T) {
	prev := notify.GetGlobalManager()
	mgr := notify.NewManager(nil, "", "", nil)
	notify.SetGlobalManager(mgr)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })

	app := testkit.NewTestApp(t)
	auth := app.AuthHeader("u-msg", "t-1", []string{"member"})

	mgr.NotifyTaskReminder(context.Background(), "u-msg", "任务提醒", "内容", "task_due", nil)
	mgr.NotifySubscriptionExpiry(context.Background(), "租户A", 3, "", []string{"u-msg"})

	w := app.Do(t, http.MethodGet, "/api/v1/messages/unread-count", "", auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var unread dto.UnreadCountResp
	testkit.DecodeData(t, w, &unread)
	require.Equal(t, int64(2), unread.Total)

	w = app.Do(t, http.MethodPut, "/api/v1/messages/read-all", `{"category":"task"}`, auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var updated struct {
		Updated int64 `json:"updated"`
	}
	testkit.DecodeData(t, w, &updated)
	require.Equal(t, int64(1), updated.Updated, "只应清除任务分类的 1 条未读")

	w = app.Do(t, http.MethodGet, "/api/v1/messages/unread-count", "", auth)
	require.Equal(t, http.StatusOK, w.Code)
	// json.Unmarshal 会复用 map 并保留旧键，断言前必须重置
	unread = dto.UnreadCountResp{}
	testkit.DecodeData(t, w, &unread)
	require.Equal(t, int64(1), unread.Total, "套餐分类未读应保留")
	require.Zero(t, unread.ByCategory["task"])

	// 非法分类 → 400
	w = app.Do(t, http.MethodPut, "/api/v1/messages/read-all", `{"category":"nope"}`, auth)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// 未登录 → 401
	w = app.Do(t, http.MethodGet, "/api/v1/messages", "", "")
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestInteg_Message_BatchReadAndDelete 端到端锁定批量操作口径：
// 只影响本人消息、个别脏 ID 不报错、空列表回 400、重复删除幂等。
func TestInteg_Message_BatchReadAndDelete(t *testing.T) {
	prev := notify.GetGlobalManager()
	mgr := notify.NewManager(nil, "", "", nil)
	notify.SetGlobalManager(mgr)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })

	app := testkit.NewTestApp(t)
	auth := app.AuthHeader("u-batch", "t-1", []string{"member"})
	other := app.AuthHeader("u-other", "t-1", []string{"member"})

	for _, taskID := range []string{"tk-1", "tk-2", "tk-3"} {
		mgr.NotifyTaskReminder(context.Background(), "u-batch", "任务提醒", "内容", "task_due",
			map[string]interface{}{"task_id": taskID})
	}

	w := app.Do(t, http.MethodGet, "/api/v1/messages?page=1&page_size=10", "", auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var page messagePage
	testkit.DecodeData(t, w, &page)
	require.Equal(t, 3, page.Pagination.Total)
	ids := make([]string, len(page.Data))
	for i, m := range page.Data {
		ids[i] = m.ID
	}

	// 1) 批量已读：混入不存在的 ID 不影响其他条目
	w = app.Do(t, http.MethodPut, "/api/v1/messages/batch/read",
		`{"ids":["`+ids[0]+`","`+ids[1]+`","missing"]}`, auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var updated struct {
		Updated int64 `json:"updated"`
	}
	testkit.DecodeData(t, w, &updated)
	require.Equal(t, int64(2), updated.Updated)

	var unread dto.UnreadCountResp
	w = app.Do(t, http.MethodGet, "/api/v1/messages/unread-count", "", auth)
	require.Equal(t, http.StatusOK, w.Code)
	testkit.DecodeData(t, w, &unread)
	require.Equal(t, int64(1), unread.Total)

	// 2) 他人批量删除不影响本人数据
	w = app.Do(t, http.MethodDelete, "/api/v1/messages/batch/delete",
		`{"ids":["`+ids[0]+`"]}`, other)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var deleted struct {
		Deleted int64 `json:"deleted"`
	}
	testkit.DecodeData(t, w, &deleted)
	require.Zero(t, deleted.Deleted)

	w = app.Do(t, http.MethodGet, "/api/v1/messages", "", auth)
	require.Equal(t, http.StatusOK, w.Code)
	testkit.DecodeData(t, w, &page)
	require.Equal(t, 3, page.Pagination.Total, "他人批量删除不得删到我的消息")

	// 3) 空列表在参数校验阶段打回
	w = app.Do(t, http.MethodPut, "/api/v1/messages/batch/read", `{"ids":[]}`, auth)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// 4) 批量删除本人消息，重复提交幂等
	w = app.Do(t, http.MethodDelete, "/api/v1/messages/batch/delete",
		`{"ids":["`+ids[0]+`","`+ids[1]+`"]}`, auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	testkit.DecodeData(t, w, &deleted)
	require.Equal(t, int64(2), deleted.Deleted)

	w = app.Do(t, http.MethodDelete, "/api/v1/messages/batch/delete",
		`{"ids":["`+ids[0]+`"]}`, auth)
	require.Equal(t, http.StatusOK, w.Code)
	testkit.DecodeData(t, w, &deleted)
	require.Zero(t, deleted.Deleted, "重复删除幂等")

	w = app.Do(t, http.MethodGet, "/api/v1/messages", "", auth)
	require.Equal(t, http.StatusOK, w.Code)
	page = messagePage{}
	testkit.DecodeData(t, w, &page)
	require.Equal(t, 1, page.Pagination.Total)
	require.Equal(t, ids[2], page.Data[0].ID)
}

// TestInteg_Message_DedupeByEvent 同一业务事件的重复投递只落一条：
// 任务提醒在渠道重试 / 多副本 / 重启补偿下可能被调用多次，收件箱不应重复刷屏；
// 不同提醒类型（指派 vs 到期）仍是两条独立消息。
func TestInteg_Message_DedupeByEvent(t *testing.T) {
	prev := notify.GetGlobalManager()
	mgr := notify.NewManager(nil, "", "", nil)
	notify.SetGlobalManager(mgr)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })

	app := testkit.NewTestApp(t)
	auth := app.AuthHeader("u-dedupe", "t-1", []string{"member"})

	extra := map[string]interface{}{"task_id": "tk-dedupe"}
	mgr.NotifyTaskReminder(context.Background(), "u-dedupe", "任务到期", "内容", "task_due", extra)
	mgr.NotifyTaskReminder(context.Background(), "u-dedupe", "任务到期", "内容", "task_due", extra)
	mgr.NotifyTaskReminder(context.Background(), "u-dedupe", "任务指派", "内容", "task_assigned", extra)

	w := app.Do(t, http.MethodGet, "/api/v1/messages/unread-count", "", auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var unread dto.UnreadCountResp
	testkit.DecodeData(t, w, &unread)
	require.Equal(t, int64(2), unread.Total, "同日同类型的重复提醒只落一条，不同类型仍各自保留")

	w = app.Do(t, http.MethodGet, "/api/v1/messages", "", auth)
	require.Equal(t, http.StatusOK, w.Code)
	var page messagePage
	testkit.DecodeData(t, w, &page)
	require.Equal(t, 2, page.Pagination.Total)
}
