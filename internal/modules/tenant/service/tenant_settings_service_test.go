package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"meteorx/internal/modules/tenant/dto"
	"meteorx/internal/modules/tenant/model"
	"meteorx/internal/modules/tenant/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSettingsRepo struct {
	settings map[string]*model.TenantSettings
}

func newStubSettingsRepo() *stubSettingsRepo {
	return &stubSettingsRepo{settings: make(map[string]*model.TenantSettings)}
}

func (r *stubSettingsRepo) GetByTenantID(_ context.Context, tenantID string) (*model.TenantSettings, error) {
	s, ok := r.settings[tenantID]
	if !ok {
		return nil, nil
	}
	return s, nil
}

func (r *stubSettingsRepo) Create(_ context.Context, s *model.TenantSettings) error {
	r.settings[s.TenantID] = s
	return nil
}

func (r *stubSettingsRepo) Update(_ context.Context, s *model.TenantSettings) error {
	r.settings[s.TenantID] = s
	return nil
}

func (r *stubSettingsRepo) Upsert(_ context.Context, s *model.TenantSettings) error {
	r.settings[s.TenantID] = s
	return nil
}

var _ repository.TenantSettingsRepository = (*stubSettingsRepo)(nil)

func TestTenantSettingsService_GetSettings_Default(t *testing.T) {
	repo := newStubSettingsRepo()
	svc := NewTenantSettingsService(repo)

	resp, err := svc.GetSettings(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, "t1", resp.TenantID)
	assert.Equal(t, "#667eea", resp.PrimaryColor)
	assert.Equal(t, "light", resp.Theme)
	assert.Equal(t, "zh-CN", resp.Language)
	assert.Equal(t, "Asia/Shanghai", resp.Timezone)
}

func TestTenantSettingsService_GetSettings_Existing(t *testing.T) {
	repo := newStubSettingsRepo()
	now := time.Now()
	repo.settings["t1"] = &model.TenantSettings{
		ID:           "s1",
		TenantID:     "t1",
		PrimaryColor: "#ff0000",
		Theme:        "dark",
		Language:     "en-US",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	svc := NewTenantSettingsService(repo)

	resp, err := svc.GetSettings(context.Background(), "t1")
	require.NoError(t, err)
	assert.Equal(t, "#ff0000", resp.PrimaryColor)
	assert.Equal(t, "dark", resp.Theme)
	assert.Equal(t, "en-US", resp.Language)
}

func TestTenantSettingsService_GetSettings_RepoError(t *testing.T) {
	svc := NewTenantSettingsService(&errorSettingsRepo{})
	_, err := svc.GetSettings(context.Background(), "t1")
	assert.Error(t, err)
}

func TestTenantSettingsService_UpdateSettings_CreateNew(t *testing.T) {
	repo := newStubSettingsRepo()
	svc := NewTenantSettingsService(repo)

	resp, err := svc.UpdateSettings(context.Background(), "t1", dto.UpdateTenantSettingsReq{
		PrimaryColor: "#00ff00",
		Theme:        "dark",
		Language:     "ja-JP",
	})
	require.NoError(t, err)
	assert.Equal(t, "#00ff00", resp.PrimaryColor)
	assert.Equal(t, "dark", resp.Theme)
	assert.Equal(t, "ja-JP", resp.Language)

	stored, ok := repo.settings["t1"]
	assert.True(t, ok)
	assert.Equal(t, "#00ff00", stored.PrimaryColor)
}

func TestTenantSettingsService_UpdateSettings_UpdateExisting(t *testing.T) {
	repo := newStubSettingsRepo()
	repo.settings["t1"] = &model.TenantSettings{
		ID:           "s1",
		TenantID:     "t1",
		PrimaryColor: "#ff0000",
		Theme:        "light",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	svc := NewTenantSettingsService(repo)

	resp, err := svc.UpdateSettings(context.Background(), "t1", dto.UpdateTenantSettingsReq{
		PrimaryColor: "#00ff00",
		Theme:        "dark",
	})
	require.NoError(t, err)
	assert.Equal(t, "#00ff00", resp.PrimaryColor)
	assert.Equal(t, "dark", resp.Theme)
}

type errorSettingsRepo struct{}

func (r *errorSettingsRepo) GetByTenantID(_ context.Context, _ string) (*model.TenantSettings, error) {
	return nil, errors.New("db error")
}
func (r *errorSettingsRepo) Create(_ context.Context, _ *model.TenantSettings) error {
	return errors.New("db error")
}
func (r *errorSettingsRepo) Update(_ context.Context, _ *model.TenantSettings) error {
	return errors.New("db error")
}
func (r *errorSettingsRepo) Upsert(_ context.Context, _ *model.TenantSettings) error {
	return errors.New("db error")
}