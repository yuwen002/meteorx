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

	"github.com/stretchr/testify/assert"
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
		Title:   "x",
		Content: "y",
		Scope:   model.AnnouncementScopeTenant,
		Status:  model.AnnouncementStatusPublished,
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
		Title:   "新标题",
		Content: "新内容",
		Scope:   model.AnnouncementScopeTenant,
		Status:  model.AnnouncementStatusPublished,
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
