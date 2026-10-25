package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"meteorx/internal/modules/message/dto"
	"meteorx/internal/modules/message/model"
	messagerepo "meteorx/internal/modules/message/repository"
	"meteorx/internal/notify"
	"meteorx/internal/ws"

	"github.com/stretchr/testify/require"
)

// ---- 测试桩 ----

type stubRepo struct {
	created      []*model.Message
	createErr    error
	listItems    []*model.Message
	listTotal    int64
	listFilter   messagerepo.ListFilter
	unreadByCat  map[string]int64
	unreadByUser map[string]int64
	markReadErr  error
	markAllN     int64
	deleteErr    error
	deletedCut   time.Time
	// 批量接口：记录服务层实际下发的 ID 列表与实际返回条数
	markBatchIDs   []string
	markBatchN     int64
	markBatchErr   error
	deleteBatchIDs []string
	deleteBatchN   int64
	deleteBatchErr error
}

func (s *stubRepo) BatchCreate(_ context.Context, msgs []*model.Message) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.created = append(s.created, msgs...)
	return nil
}

func (s *stubRepo) ListByUser(_ context.Context, f messagerepo.ListFilter) ([]*model.Message, int64, error) {
	s.listFilter = f
	return s.listItems, s.listTotal, nil
}

func (s *stubRepo) CountUnread(_ context.Context, userID, category string) (int64, error) {
	return s.unreadByCat[category], nil
}

func (s *stubRepo) CountUnreadByUsers(_ context.Context, userIDs []string) (map[string]int64, error) {
	out := make(map[string]int64, len(userIDs))
	for _, id := range userIDs {
		out[id] = s.unreadByUser[id]
	}
	return out, nil
}

func (s *stubRepo) CountUnreadByCategory(context.Context, string) (map[string]int64, error) {
	return s.unreadByCat, nil
}

func (s *stubRepo) MarkRead(context.Context, string, string) error { return s.markReadErr }

func (s *stubRepo) MarkAllRead(context.Context, string, string) (int64, error) {
	return s.markAllN, nil
}

func (s *stubRepo) Delete(context.Context, string, string) error { return s.deleteErr }

func (s *stubRepo) MarkReadBatch(_ context.Context, userID string, ids []string) (int64, error) {
	s.markBatchIDs = ids
	return s.markBatchN, s.markBatchErr
}

func (s *stubRepo) DeleteBatch(_ context.Context, userID string, ids []string) (int64, error) {
	s.deleteBatchIDs = ids
	return s.deleteBatchN, s.deleteBatchErr
}

func (s *stubRepo) DeleteOlderThan(_ context.Context, cut time.Time) (int64, error) {
	s.deletedCut = cut
	return 3, nil
}

type stubDirectory struct {
	ids []string
	err error
}

func (s *stubDirectory) ListActiveUserIDs(context.Context, string, int) ([]string, error) {
	return s.ids, s.err
}

type stubPusher struct {
	sent []*ws.Message
	to   []string
}

func (s *stubPusher) SendToUser(userID string, msg *ws.Message) {
	s.to = append(s.to, userID)
	s.sent = append(s.sent, msg)
}

// stubPrefs 偏好仓储桩：mutedByCat 按分类预置「已关闭接收」的收件人。
type stubPrefs struct {
	mutedCats  map[string]bool
	mutedByCat map[string]map[string]struct{}
	filterErr  error
	listErr    error
	upsertErr  error
	upserted   []model.CategoryPreference
	gotUserID  string
	gotCat     string
}

func (s *stubPrefs) MutedCategories(context.Context, string) (map[string]bool, error) {
	return s.mutedCats, s.listErr
}

func (s *stubPrefs) Upsert(_ context.Context, userID string, prefs []model.CategoryPreference) error {
	s.gotUserID = userID
	s.upserted = append(s.upserted, prefs...)
	return s.upsertErr
}

func (s *stubPrefs) MutedUserIDs(_ context.Context, category string, _ []string) (map[string]struct{}, error) {
	s.gotCat = category
	return s.mutedByCat[category], s.filterErr
}

// ---- 用例 ----

func TestDeliver_PersistsPerRecipientAndPushesBadge(t *testing.T) {
	repo := &stubRepo{unreadByUser: map[string]int64{"u1": 2, "u2": 5}}
	pusher := &stubPusher{}
	svc := NewMessageService(repo, &stubDirectory{}, nil, pusher)

	err := svc.Deliver(context.Background(), &notify.InboxMessage{
		UserIDs: []string{"u1", "u2", "u1", " "}, // 含重复与空白，均应被去重
		Type:    "task_due",
		Title:   "任务即将到期",
		Content: "请尽快处理",
		Link:    "/task",
	})
	require.NoError(t, err)

	require.Len(t, repo.created, 2, "同一收件人只落一行")
	require.Equal(t, "u1", repo.created[0].UserID)
	require.Equal(t, "u2", repo.created[1].UserID)
	for _, m := range repo.created {
		require.NotEmpty(t, m.ID, "每条消息都必须有独立 ID")
		require.Equal(t, model.CategoryTask, m.Category, "task 前缀类型应映射到任务分类")
		require.Equal(t, "/task", m.Link)
	}
	require.NotEqual(t, repo.created[0].ID, repo.created[1].ID)

	require.Len(t, pusher.sent, 2)
	require.Equal(t, ws.MsgTypeUnreadCount, pusher.sent[0].Type)
	payload, ok := pusher.sent[0].Payload.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, int64(2), payload["count"], "推送载荷应与该收件人的未读数一致")
	require.Equal(t, "u2", pusher.to[1])
}

func TestDeliver_ExpandsTenantWhenNoUserIDs(t *testing.T) {
	repo := &stubRepo{}
	svc := NewMessageService(repo, &stubDirectory{ids: []string{"a", "b"}}, nil, nil)

	require.NoError(t, svc.Deliver(context.Background(), &notify.InboxMessage{
		TenantID: "t1",
		Type:     "announcement",
		Title:    "维护通知",
	}))
	require.Len(t, repo.created, 2)
	require.Equal(t, model.CategoryAnnouncement, repo.created[0].Category)
	require.Equal(t, "t1", repo.created[0].TenantID)
}

func TestDeliver_SkipsWhenNoRecipients(t *testing.T) {
	repo := &stubRepo{}
	svc := NewMessageService(repo, &stubDirectory{}, nil, nil)

	require.NoError(t, svc.Deliver(context.Background(), nil))
	require.NoError(t, svc.Deliver(context.Background(), &notify.InboxMessage{Type: "system"}))
	require.Empty(t, repo.created, "无人可投递时不应产生不可见的消息行")
}

// TestDeliver_CapsTenantFanout 一条租户定向消息最多展开 maxTenantFanout 人，
// 防止单条公告按用户量放大写入。
func TestDeliver_CapsTenantFanout(t *testing.T) {
	ids := make([]string, maxTenantFanout+50)
	for i := range ids {
		ids[i] = fmt.Sprintf("u%d", i)
	}
	repo := &stubRepo{}
	svc := NewMessageService(repo, &stubDirectory{ids: ids}, nil, nil)

	require.NoError(t, svc.Deliver(context.Background(), &notify.InboxMessage{TenantID: "t1", Type: "announcement"}))
	require.Len(t, repo.created, maxTenantFanout)
}

func TestDeliver_ReturnsCreateError(t *testing.T) {
	repo := &stubRepo{createErr: errors.New("db down")}
	pusher := &stubPusher{}
	svc := NewMessageService(repo, &stubDirectory{}, nil, pusher)

	err := svc.Deliver(context.Background(), &notify.InboxMessage{UserIDs: []string{"u1"}, Type: "system"})
	require.Error(t, err)
	require.Empty(t, pusher.sent, "落库失败时不应推送角标")
}

// TestDeliver_CustomCategoryWins 调用方显式指定分类时不再按类型推导。
func TestDeliver_CustomCategoryWins(t *testing.T) {
	repo := &stubRepo{}
	svc := NewMessageService(repo, &stubDirectory{}, nil, nil)
	require.NoError(t, svc.Deliver(context.Background(), &notify.InboxMessage{
		UserIDs:  []string{"u1"},
		Type:     "task_due",
		Category: model.CategoryPlan,
	}))
	require.Len(t, repo.created, 1)
	require.Equal(t, model.CategoryPlan, repo.created[0].Category)
}

func TestCategoryForType(t *testing.T) {
	cases := map[string]string{
		"announcement":        model.CategoryAnnouncement,
		"subscription_expiry": model.CategoryPlan,
		"task_assigned":       model.CategoryTask,
		"task_due":            model.CategoryTask,
		"alert":               model.CategoryAudit,
		"cancel_request":      model.CategorySystem,
		"":                    model.CategorySystem,
	}
	for in, want := range cases {
		require.Equal(t, want, CategoryForType(in), "type=%s", in)
	}
}

func TestListAndUnreadCount(t *testing.T) {
	repo := &stubRepo{
		listItems:   []*model.Message{{ID: "m1", Title: "hello"}},
		listTotal:   1,
		unreadByCat: map[string]int64{model.CategoryTask: 2, model.CategorySystem: 3},
	}
	svc := NewMessageService(repo, &stubDirectory{}, nil, nil)

	items, total, err := svc.List(context.Background(), messagerepo.ListFilter{UserID: "u1", Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "m1", items[0].ID)
	require.Equal(t, "u1", repo.listFilter.UserID)

	count, err := svc.UnreadCount(context.Background(), "u1")
	require.NoError(t, err)
	require.Equal(t, int64(5), count.Total, "未读总数应等于各分类之和")
	require.Equal(t, int64(2), count.ByCategory[model.CategoryTask])
	require.IsType(t, &dto.UnreadCountResp{}, count)
}

func TestPassthroughErrors(t *testing.T) {
	repo := &stubRepo{markReadErr: ErrNotFound, deleteErr: ErrNotFound}
	svc := NewMessageService(repo, &stubDirectory{}, nil, nil)

	require.True(t, errors.Is(svc.MarkRead(context.Background(), "u1", "m1"), ErrNotFound))
	require.True(t, errors.Is(svc.Delete(context.Background(), "u1", "m1"), ErrNotFound))
}

// TestNewMessageService_TypedNilHubIgnored WS 关闭时 bootstrap 会传入 nil *ws.Hub，
// 装箱后接口不为 nil；必须识别并丢弃，否则推送角标时会空指针崩溃。
func TestNewMessageService_TypedNilHubIgnored(t *testing.T) {
	var nilHub *ws.Hub
	repo := &stubRepo{unreadByUser: map[string]int64{}}
	svc := NewMessageService(repo, &stubDirectory{}, nil, nilHub)
	require.Nil(t, svc.hub, "typed nil hub 应被丢弃")

	require.NotPanics(t, func() {
		require.NoError(t, svc.Deliver(context.Background(), &notify.InboxMessage{UserIDs: []string{"u1"}}))
	})
	require.Len(t, repo.created, 1)
}

// TestDeliver_DedupeKey 去重键：显式值优先，缺省回退「来源类型:来源ID」，
// 两者都定位不到业务事件（如平台公告）时不去重。
func TestDeliver_DedupeKey(t *testing.T) {
	cases := []struct {
		name string
		in   notify.InboxMessage
		want string
	}{
		{
			name: "显式键优先",
			in:   notify.InboxMessage{UserIDs: []string{"u1"}, SourceType: "task", SourceID: "tk1", DedupeKey: "custom:key"},
			want: "custom:key",
		},
		{
			name: "回退来源类型与来源ID",
			in:   notify.InboxMessage{UserIDs: []string{"u1"}, SourceType: "task", SourceID: "tk1"},
			want: "task:tk1",
		},
		{
			name: "来源不完整时不设键",
			in:   notify.InboxMessage{UserIDs: []string{"u1"}, SourceType: "announcement"},
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubRepo{}
			svc := NewMessageService(repo, &stubDirectory{}, nil, nil)
			in := tc.in
			require.NoError(t, svc.Deliver(context.Background(), &in))
			require.Len(t, repo.created, 1)
			require.Equal(t, tc.want, repo.created[0].DedupeKey)
		})
	}
}

// TestDeliver_DedupeKeySharedByRecipients 同一事件的所有收件人共享一个去重键，
// 唯一性由 (user_id, dedupe_key) 保证，不能因多人接收而生成不同键。
func TestDeliver_DedupeKeySharedByRecipients(t *testing.T) {
	repo := &stubRepo{}
	svc := NewMessageService(repo, &stubDirectory{ids: []string{"a", "b"}}, nil, nil)

	require.NoError(t, svc.Deliver(context.Background(), &notify.InboxMessage{
		TenantID: "t1", Type: "announcement", DedupeKey: "announcement:an-1",
	}))
	require.Len(t, repo.created, 2)
	for _, m := range repo.created {
		require.Equal(t, "announcement:an-1", m.DedupeKey)
	}
}

// TestDeliver_SkipsUsersWhoMutedCategory 关闭接收的收件人既不落库也不推角标，
// 其余收件人照常投递。
func TestDeliver_SkipsUsersWhoMutedCategory(t *testing.T) {
	repo := &stubRepo{unreadByUser: map[string]int64{"u1": 1}}
	pusher := &stubPusher{}
	prefs := &stubPrefs{mutedByCat: map[string]map[string]struct{}{
		model.CategoryTask: {"u2": {}},
	}}
	svc := NewMessageService(repo, &stubDirectory{}, prefs, pusher)

	require.NoError(t, svc.Deliver(context.Background(), &notify.InboxMessage{
		UserIDs: []string{"u1", "u2"}, Type: "task_due", Title: "任务即将到期",
	}))
	require.Equal(t, model.CategoryTask, prefs.gotCat, "应按推导出的分类查询偏好")
	require.Len(t, repo.created, 1)
	require.Equal(t, "u1", repo.created[0].UserID)
	require.Len(t, pusher.to, 1)
	require.Equal(t, "u1", pusher.to[0], "被过滤的收件人不应收到角标")
}

// TestDeliver_AllRecipientsMutedWritesNothing 全员关闭时不应产生任何行或推送。
func TestDeliver_AllRecipientsMutedWritesNothing(t *testing.T) {
	repo := &stubRepo{}
	pusher := &stubPusher{}
	prefs := &stubPrefs{mutedByCat: map[string]map[string]struct{}{
		model.CategoryAnnouncement: {"u1": {}, "u2": {}},
	}}
	svc := NewMessageService(repo, &stubDirectory{}, prefs, pusher)

	require.NoError(t, svc.Deliver(context.Background(), &notify.InboxMessage{
		UserIDs: []string{"u1", "u2"}, Type: "announcement",
	}))
	require.Empty(t, repo.created)
	require.Empty(t, pusher.sent)
}

// TestDeliver_MandatoryCategoryIgnoresMute 安全告警强制接收：
// 即使库里存在关闭记录（历史数据/人工修改），也必须投递且不去查偏好。
func TestDeliver_MandatoryCategoryIgnoresMute(t *testing.T) {
	repo := &stubRepo{}
	prefs := &stubPrefs{mutedByCat: map[string]map[string]struct{}{
		model.CategoryAudit: {"u1": {}},
	}}
	svc := NewMessageService(repo, &stubDirectory{}, prefs, nil)

	require.NoError(t, svc.Deliver(context.Background(), &notify.InboxMessage{
		UserIDs: []string{"u1"}, Type: "alert", Category: model.CategoryAudit, Title: "[告警] x",
	}))
	require.Empty(t, prefs.gotCat, "强制接收的分类不应进入偏好过滤")
	require.Len(t, repo.created, 1)
}

// TestDeliver_PreferenceQueryErrorFailsOpen 偏好查询失败时按默认接收投递：
// 宁可多打扰，不可漏消息。
func TestDeliver_PreferenceQueryErrorFailsOpen(t *testing.T) {
	repo := &stubRepo{}
	prefs := &stubPrefs{filterErr: errors.New("db down")}
	svc := NewMessageService(repo, &stubDirectory{}, prefs, nil)

	require.NoError(t, svc.Deliver(context.Background(), &notify.InboxMessage{
		UserIDs: []string{"u1", "u2"}, Type: "task_due",
	}))
	require.Len(t, repo.created, 2)
}

// TestPreferences_DefaultsAndMandatory 未设置过的分类按接收返回，
// 且全部分类都在列表里、顺序固定，安全告警标记为强制接收。
func TestPreferences_DefaultsAndMandatory(t *testing.T) {
	prefs := &stubPrefs{mutedCats: map[string]bool{model.CategoryTask: true}}
	svc := NewMessageService(&stubRepo{}, &stubDirectory{}, prefs, nil)

	resp, err := svc.Preferences(context.Background(), "u1")
	require.NoError(t, err)
	require.Len(t, resp.Items, len(model.AllCategories()))
	for i, c := range model.AllCategories() {
		require.Equal(t, c, resp.Items[i].Category, "分类顺序应与 AllCategories 一致")
	}
	require.True(t, resp.Items[2].Muted, "task 应读取到已关闭")
	require.False(t, resp.Items[0].Muted, "未设置的分类默认为接收")

	var audit *dto.PreferenceItem
	for i := range resp.Items {
		if resp.Items[i].Category == model.CategoryAudit {
			audit = &resp.Items[i]
		}
	}
	require.NotNil(t, audit)
	require.True(t, audit.Mandatory, "安全告警必须标记为强制接收")
}

// TestUpdatePreference_RejectsInvalidAndMandatory 非法分类与关闭安全告警都必须拒绝，
// 且拒绝时不能落库（否则前端报错但实际已生效）。
func TestUpdatePreference_RejectsInvalidAndMandatory(t *testing.T) {
	prefs := &stubPrefs{}
	svc := NewMessageService(&stubRepo{}, &stubDirectory{}, prefs, nil)

	_, err := svc.UpdatePreference(context.Background(), "u1", dto.UpdatePreferenceReq{Category: "nope"})
	require.ErrorIs(t, err, ErrInvalidCategory)

	_, err = svc.UpdatePreference(context.Background(), "u1", dto.UpdatePreferenceReq{Category: model.CategoryAudit, Muted: true})
	require.ErrorIs(t, err, ErrMandatoryCategory)

	require.Empty(t, prefs.upserted, "被拒绝的请求不应写入偏好")
}

// TestUpdatePreference_UpsertsAndReturnsView 开关写回后返回最新视图，
// 前端可直接用返回值刷新开关状态。
func TestUpdatePreference_UpsertsAndReturnsView(t *testing.T) {
	prefs := &stubPrefs{}
	svc := NewMessageService(&stubRepo{}, &stubDirectory{}, prefs, nil)

	resp, err := svc.UpdatePreference(context.Background(), "u7", dto.UpdatePreferenceReq{
		Category: model.CategoryTask, Muted: true,
	})
	require.NoError(t, err)
	require.Equal(t, "u7", prefs.gotUserID, "偏好必须落在当前登录用户名下")
	require.Equal(t, []model.CategoryPreference{{Category: model.CategoryTask, Muted: true}}, prefs.upserted)

	// stub 不回写映射，因此这里只校验返回结构完整；开关值由仓储层用例覆盖
	require.Len(t, resp.Items, len(model.AllCategories()))
}

// TestUpdatePreference_WithoutPreferenceRepository 未装配偏好仓储时写入必须报错，
// 不能假装成功（否则前端提示已保存、实际什么都没改）。
func TestUpdatePreference_WithoutPreferenceRepository(t *testing.T) {
	svc := NewMessageService(&stubRepo{}, &stubDirectory{}, nil, nil)

	resp, err := svc.Preferences(context.Background(), "u1")
	require.NoError(t, err, "读偏好不应因未装配而失败：全部按接收返回")
	require.Len(t, resp.Items, len(model.AllCategories()))

	_, err = svc.UpdatePreference(context.Background(), "u1", dto.UpdatePreferenceReq{Category: model.CategoryTask, Muted: true})
	require.Error(t, err)
}

// TestMarkReadBatch_NormalizesIDs 批量入参先去空白再去重：
// 同一 ID 在 IN 列表里重复出现既多占位，也让「实际更新条数」难以对账。
func TestMarkReadBatch_NormalizesIDs(t *testing.T) {
	repo := &stubRepo{markBatchN: 2}
	svc := NewMessageService(repo, &stubDirectory{}, nil, nil)

	n, err := svc.MarkReadBatch(context.Background(), "u1", []string{" m1 ", "", "m2", "m1", "   "})
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
	require.Equal(t, []string{"m1", "m2"}, repo.markBatchIDs)
}

// TestDeleteBatch_CapsIDs 超过上限时只取前 maxBatchIDs 个，
// 服务层不能依赖 handler 的校验（内部调用同样可能传入超长列表）。
func TestDeleteBatch_CapsIDs(t *testing.T) {
	ids := make([]string, maxBatchIDs+20)
	for i := range ids {
		ids[i] = fmt.Sprintf("m%d", i)
	}
	repo := &stubRepo{deleteBatchN: int64(maxBatchIDs)}
	svc := NewMessageService(repo, &stubDirectory{}, nil, nil)

	n, err := svc.DeleteBatch(context.Background(), "u1", ids)
	require.NoError(t, err)
	require.Equal(t, int64(maxBatchIDs), n)
	require.Len(t, repo.deleteBatchIDs, maxBatchIDs)
	require.Equal(t, "m0", repo.deleteBatchIDs[0])
}

// TestBatch_PropagatesRepositoryError 仓储失败必须原样上抛，
// handler 才能回 500 而不是把「0 条」当成功告诉前端。
func TestBatch_PropagatesRepositoryError(t *testing.T) {
	errBoom := errors.New("db down")
	repo := &stubRepo{markBatchErr: errBoom, deleteBatchErr: errBoom}
	svc := NewMessageService(repo, &stubDirectory{}, nil, nil)

	_, err := svc.MarkReadBatch(context.Background(), "u1", []string{"m1"})
	require.ErrorIs(t, err, errBoom)

	_, err = svc.DeleteBatch(context.Background(), "u1", []string{"m1"})
	require.ErrorIs(t, err, errBoom)
}

// TestBatch_EmptyIDsAfterNormalize 全部是空白字符时视为无操作，
// 不应把空 IN 列表下发给仓储层。
func TestBatch_EmptyIDsAfterNormalize(t *testing.T) {
	repo := &stubRepo{}
	svc := NewMessageService(repo, &stubDirectory{}, nil, nil)

	n, err := svc.MarkReadBatch(context.Background(), "u1", []string{"", "  "})
	require.NoError(t, err)
	require.Zero(t, n)
	require.Empty(t, repo.markBatchIDs)
}
