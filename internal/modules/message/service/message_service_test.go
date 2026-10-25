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

// ---- 用例 ----

func TestDeliver_PersistsPerRecipientAndPushesBadge(t *testing.T) {
	repo := &stubRepo{unreadByUser: map[string]int64{"u1": 2, "u2": 5}}
	pusher := &stubPusher{}
	svc := NewMessageService(repo, &stubDirectory{}, pusher)

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
	svc := NewMessageService(repo, &stubDirectory{ids: []string{"a", "b"}}, nil)

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
	svc := NewMessageService(repo, &stubDirectory{}, nil)

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
	svc := NewMessageService(repo, &stubDirectory{ids: ids}, nil)

	require.NoError(t, svc.Deliver(context.Background(), &notify.InboxMessage{TenantID: "t1", Type: "announcement"}))
	require.Len(t, repo.created, maxTenantFanout)
}

func TestDeliver_ReturnsCreateError(t *testing.T) {
	repo := &stubRepo{createErr: errors.New("db down")}
	pusher := &stubPusher{}
	svc := NewMessageService(repo, &stubDirectory{}, pusher)

	err := svc.Deliver(context.Background(), &notify.InboxMessage{UserIDs: []string{"u1"}, Type: "system"})
	require.Error(t, err)
	require.Empty(t, pusher.sent, "落库失败时不应推送角标")
}

// TestDeliver_CustomCategoryWins 调用方显式指定分类时不再按类型推导。
func TestDeliver_CustomCategoryWins(t *testing.T) {
	repo := &stubRepo{}
	svc := NewMessageService(repo, &stubDirectory{}, nil)
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
	svc := NewMessageService(repo, &stubDirectory{}, nil)

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
	svc := NewMessageService(repo, &stubDirectory{}, nil)

	require.True(t, errors.Is(svc.MarkRead(context.Background(), "u1", "m1"), ErrNotFound))
	require.True(t, errors.Is(svc.Delete(context.Background(), "u1", "m1"), ErrNotFound))
}

// TestNewMessageService_TypedNilHubIgnored WS 关闭时 bootstrap 会传入 nil *ws.Hub，
// 装箱后接口不为 nil；必须识别并丢弃，否则推送角标时会空指针崩溃。
func TestNewMessageService_TypedNilHubIgnored(t *testing.T) {
	var nilHub *ws.Hub
	repo := &stubRepo{unreadByUser: map[string]int64{}}
	svc := NewMessageService(repo, &stubDirectory{}, nilHub)
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
			svc := NewMessageService(repo, &stubDirectory{}, nil)
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
	svc := NewMessageService(repo, &stubDirectory{ids: []string{"a", "b"}}, nil)

	require.NoError(t, svc.Deliver(context.Background(), &notify.InboxMessage{
		TenantID: "t1", Type: "announcement", DedupeKey: "announcement:an-1",
	}))
	require.Len(t, repo.created, 2)
	for _, m := range repo.created {
		require.Equal(t, "announcement:an-1", m.DedupeKey)
	}
}
