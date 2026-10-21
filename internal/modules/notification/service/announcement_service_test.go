package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"meteorx/internal/modules/notification/dto"
	"meteorx/internal/modules/notification/model"
	"meteorx/internal/modules/notification/repository"
	"meteorx/internal/modules/notification/service"
	"meteorx/internal/notify"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAnnouncementRepo 公告仓库内存实现
type mockAnnouncementRepo struct {
	items map[string]*model.Announcement
	// 调用记录
	created   int
	updated   int
	deleted   []string
	lastQuery *repository.AnnouncementQuery
	lastPage  int
	lastSize  int
}

func newMockAnnouncementRepo() *mockAnnouncementRepo {
	return &mockAnnouncementRepo{
		items:   map[string]*model.Announcement{},
		deleted: []string{},
	}
}

func (m *mockAnnouncementRepo) Create(_ context.Context, a *model.Announcement) error {
	m.created++
	m.items[a.ID] = a
	return nil
}

func (m *mockAnnouncementRepo) Update(_ context.Context, a *model.Announcement) error {
	m.updated++
	if _, ok := m.items[a.ID]; ok {
		m.items[a.ID] = a
	}
	return nil
}

func (m *mockAnnouncementRepo) GetByID(_ context.Context, id string) (*model.Announcement, error) {
	if a, ok := m.items[id]; ok {
		return a, nil
	}
	return nil, errors.New("announcement not found")
}

func (m *mockAnnouncementRepo) Delete(_ context.Context, id string) error {
	m.deleted = append(m.deleted, id)
	return nil
}

func (m *mockAnnouncementRepo) List(_ context.Context, query *repository.AnnouncementQuery) ([]*model.Announcement, int64, error) {
	m.lastQuery = query
	var out []*model.Announcement
	for _, a := range m.items {
		out = append(out, a)
	}
	return out, int64(len(out)), nil
}

func (m *mockAnnouncementRepo) ListForTenant(_ context.Context, _ string, page, pageSize int) ([]*model.Announcement, int64, error) {
	m.lastPage = page
	m.lastSize = pageSize
	var out []*model.Announcement
	for _, a := range m.items {
		if a.Status == model.AnnouncementStatusPublished {
			out = append(out, a)
		}
	}
	return out, int64(len(out)), nil
}

func seedAnnouncement(m *mockAnnouncementRepo, id string) *model.Announcement {
	a := &model.Announcement{
		ID:          id,
		Title:       "公告标题",
		Content:     "公告内容",
		Scope:       model.AnnouncementScopeAll,
		Status:      model.AnnouncementStatusDraft,
		PublisherID: "admin-1",
	}
	m.items[id] = a
	return a
}

func TestAnnouncementCreate_DraftWithoutPublishTime(t *testing.T) {
	repo := newMockAnnouncementRepo()
	svc := service.NewAnnouncementService(repo)

	resp, err := svc.Create(context.Background(), "admin-1", dto.CreateAnnouncementReq{
		Title:   "系统升级公告",
		Content: "本周六维护",
		Scope:   model.AnnouncementScopeAll,
		Status:  model.AnnouncementStatusDraft,
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, repo.created)
	assert.Len(t, resp.ID, 26) // ULID
	assert.Equal(t, "系统升级公告", resp.Title)
	assert.Equal(t, model.AnnouncementStatusDraft, resp.Status)
	assert.Empty(t, resp.PublishAt) // 草稿不记发布时间
	assert.Equal(t, "admin-1", resp.PublisherID)
}

func TestAnnouncementCreate_PublishedAutoFillsPublishTime(t *testing.T) {
	repo := newMockAnnouncementRepo()
	svc := service.NewAnnouncementService(repo)

	resp, err := svc.Create(context.Background(), "admin-1", dto.CreateAnnouncementReq{
		Title:          "x",
		Content:        "y",
		Scope:          model.AnnouncementScopeTenant,
		TargetTenantID: "tenant-1",
		Status:         model.AnnouncementStatusPublished,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.PublishAt)
	assert.Equal(t, model.AnnouncementStatusPublished, resp.Status)
}

func TestAnnouncementUpdate_ChangesFieldsAndBackfillsPublishTime(t *testing.T) {
	repo := newMockAnnouncementRepo()
	seedAnnouncement(repo, "ann-1")
	svc := service.NewAnnouncementService(repo)

	resp, err := svc.Update(context.Background(), "ann-1", dto.UpdateAnnouncementReq{
		Title:          "新标题",
		Content:        "新内容",
		Scope:          model.AnnouncementScopeTenant,
		TargetTenantID: "tenant-1",
		Status:         model.AnnouncementStatusPublished,
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, repo.updated)
	assert.Equal(t, "新标题", resp.Title)
	assert.NotEmpty(t, resp.PublishAt)
	assert.Equal(t, "admin-1", repo.items["ann-1"].PublisherID) // 发布人不被覆盖
}

func TestAnnouncementUpdate_NotFound(t *testing.T) {
	repo := newMockAnnouncementRepo()
	svc := service.NewAnnouncementService(repo)

	_, err := svc.Update(context.Background(), "missing", dto.UpdateAnnouncementReq{})

	assert.Error(t, err)
}

func TestAnnouncementUpdateStatus_OfflineKeepsOriginalPublishTime(t *testing.T) {
	repo := newMockAnnouncementRepo()
	pub := time.Now().Add(-24 * time.Hour)
	seedAnnouncement(repo, "ann-1")
	repo.items["ann-1"].PublishAt = &pub
	svc := service.NewAnnouncementService(repo)

	resp, err := svc.UpdateStatus(context.Background(), "ann-1", model.AnnouncementStatusOffline)

	assert.NoError(t, err)
	assert.Equal(t, model.AnnouncementStatusOffline, resp.Status)
	assert.Equal(t, pub.Format("2006-01-02 15:04:05"), resp.PublishAt)
}

func TestAnnouncementGetByID_AndDelete(t *testing.T) {
	repo := newMockAnnouncementRepo()
	seedAnnouncement(repo, "ann-1")
	svc := service.NewAnnouncementService(repo)

	got, err := svc.GetByID(context.Background(), "ann-1")
	assert.NoError(t, err)
	assert.Equal(t, "ann-1", got.ID)

	err = svc.Delete(context.Background(), "ann-1")
	assert.NoError(t, err)
	assert.Equal(t, []string{"ann-1"}, repo.deleted)

	_, err = svc.GetByID(context.Background(), "nope")
	assert.Error(t, err)
}

func TestAnnouncementList_ForwardsQueryAndMapsResp(t *testing.T) {
	repo := newMockAnnouncementRepo()
	seedAnnouncement(repo, "ann-1")
	seedAnnouncement(repo, "ann-2")
	svc := service.NewAnnouncementService(repo)

	resp, err := svc.List(context.Background(), &dto.ListAnnouncementsQuery{
		Page: 2, PageSize: 10, Keyword: "升级", Status: model.AnnouncementStatusDraft, Scope: model.AnnouncementScopeAll,
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Items, 2)
	assert.NotNil(t, repo.lastQuery)
	assert.Equal(t, 2, repo.lastQuery.Page)
	assert.Equal(t, "升级", repo.lastQuery.Keyword)
}

func TestAnnouncementListForTenant_OnlyPublished(t *testing.T) {
	repo := newMockAnnouncementRepo()
	seedAnnouncement(repo, "ann-draft")
	seedAnnouncement(repo, "ann-pub")
	repo.items["ann-pub"].Status = model.AnnouncementStatusPublished
	svc := service.NewAnnouncementService(repo)

	resp, err := svc.ListForTenant(context.Background(), "tenant-1", 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Equal(t, "ann-pub", resp.Items[0].ID)
	assert.Equal(t, 1, repo.lastPage)
	assert.Equal(t, 20, repo.lastSize)
}

// ---------------------------------------------------------------------------
// 租户定向公告：目标校验 / 发布回调（含后台「发布」按钮路径）
// ---------------------------------------------------------------------------

// TestAnnouncementCreate_TenantScopeRequiresTarget 文档口径「scope=tenant 时
// target_tenant_id 必填」必须在服务端落实，否则会产出「定向给空租户」的公告
// （推送无人可收、租户列表也永远看不到）。
func TestAnnouncementCreate_TenantScopeRequiresTarget(t *testing.T) {
	repo := newMockAnnouncementRepo()
	svc := service.NewAnnouncementService(repo)

	_, err := svc.Create(context.Background(), "admin-1", dto.CreateAnnouncementReq{
		Title: "x", Content: "y", Scope: model.AnnouncementScopeTenant,
	})
	assert.ErrorIs(t, err, service.ErrTenantTargetRequired)
	assert.Zero(t, repo.created, "非法请求不得落库")
}

// TestAnnouncementUpdate_TenantScopeRequiresTarget 编辑时同样不得把定向公告改成无目标。
func TestAnnouncementUpdate_TenantScopeRequiresTarget(t *testing.T) {
	repo := newMockAnnouncementRepo()
	seedAnnouncement(repo, "ann-1")
	svc := service.NewAnnouncementService(repo)

	_, err := svc.Update(context.Background(), "ann-1", dto.UpdateAnnouncementReq{
		Title: "x", Content: "y", Scope: model.AnnouncementScopeTenant,
	})
	assert.ErrorIs(t, err, service.ErrTenantTargetRequired)
	assert.Zero(t, repo.updated)
}

// TestAnnouncementUpdateStatus_PublishingTriggersCallback 后台行内「发布」按钮
// 走的就是 UpdateStatus：旧实现不触发 onPublish，草稿转发布一条通知都不发；
// 下架不得重复推送。
func TestAnnouncementUpdateStatus_PublishingTriggersCallback(t *testing.T) {
	repo := newMockAnnouncementRepo()
	seedAnnouncement(repo, "ann-1") // 草稿，scope=all
	svc := service.NewAnnouncementService(repo)

	type call struct{ scope, tenant string }
	var calls []call
	svc.SetPublishCallback(func(_ context.Context, _, _, scope, tenant string) {
		calls = append(calls, call{scope, tenant})
	})

	_, err := svc.UpdateStatus(context.Background(), "ann-1", model.AnnouncementStatusPublished)
	require.NoError(t, err)
	require.Len(t, calls, 1, "草稿转发布必须触发通知回调")
	assert.Equal(t, model.AnnouncementScopeAll, calls[0].scope)

	// 重复发布（已发布→已发布）不得再推一次：后台重复点「发布」不能刷屏
	_, err = svc.UpdateStatus(context.Background(), "ann-1", model.AnnouncementStatusPublished)
	require.NoError(t, err)
	assert.Len(t, calls, 1, "重复发布不得重复推送")

	_, err = svc.UpdateStatus(context.Background(), "ann-1", model.AnnouncementStatusOffline)
	require.NoError(t, err)
	assert.Len(t, calls, 1, "下架不得重复推送")
}

// TestAnnouncementUpdateStatus_PublishingForwardsTenant 定向公告经状态接口发布时
// 必须把目标租户透传给回调（否则下游只能广播）。
func TestAnnouncementUpdateStatus_PublishingForwardsTenant(t *testing.T) {
	repo := newMockAnnouncementRepo()
	a := seedAnnouncement(repo, "ann-1")
	a.Scope = model.AnnouncementScopeTenant
	a.TargetTenantID = "t-a"
	svc := service.NewAnnouncementService(repo)

	var gotTenant string
	svc.SetPublishCallback(func(_ context.Context, _, _, _, tenant string) { gotTenant = tenant })

	_, err := svc.UpdateStatus(context.Background(), "ann-1", model.AnnouncementStatusPublished)
	require.NoError(t, err)
	assert.Equal(t, "t-a", gotTenant)
}

// captureNotifier 记录最近一条消息的通道桩（用于验证默认发布回调的投递参数）。
type captureNotifier struct {
	channel notify.ChannelType
	last    *notify.Message
}

func (c *captureNotifier) Type() notify.ChannelType { return c.channel }
func (c *captureNotifier) Name() string             { return "capture-" + string(c.channel) }
func (c *captureNotifier) Send(_ context.Context, msg *notify.Message) error {
	c.last = msg
	return nil
}

// TestDefaultPublishCallback_ForwardsTenantToWS 默认发布回调不得把 scope 信息
// 丢掉：旧实现写死 nil 收件人 → SendToAll → 定向公告被广播给所有租户在线用户。
func TestDefaultPublishCallback_ForwardsTenantToWS(t *testing.T) {
	wsCap := &captureNotifier{channel: notify.ChannelWebSocket}
	mgr := notify.NewManager(nil, "", "", nil)
	mgr.AddChannel(wsCap)

	prev := notify.GetGlobalManager()
	notify.SetGlobalManager(mgr)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })

	repo := newMockAnnouncementRepo()
	svc := service.NewAnnouncementService(repo) // 使用默认发布回调

	_, err := svc.Create(context.Background(), "admin-1", dto.CreateAnnouncementReq{
		Title: "定向升级通知", Content: "仅 A 公司", Scope: model.AnnouncementScopeTenant,
		TargetTenantID: "t-a", Status: model.AnnouncementStatusPublished,
	})
	require.NoError(t, err)
	require.NotNil(t, wsCap.last, "默认回调必须真正投递到 WS 渠道")
	assert.Equal(t, "t-a", wsCap.last.TenantID, "定向公告必须携带目标租户，不得广播")
}
