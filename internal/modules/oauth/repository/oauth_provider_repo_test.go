package repository

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/oauth/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newProviderTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&oauthProviderPO{}))
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	return db
}

func sampleProvider(code string) *model.OAuthProvider {
	now := time.Now().Truncate(time.Second)
	return &model.OAuthProvider{
		ID:           "id-" + code,
		Kind:         model.ProviderKindWeChat,
		Code:         code,
		DisplayName:  "微信",
		Enabled:      true,
		ClientID:     "wx-app-id",
		ClientSecret: "enc:v1:c2VjcmV0",
		RedirectURL:  "http://localhost:5173/auth/wechat/callback",
		Scopes:       []string{"openid", "email"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestProviderRepo_CreateAndGet(t *testing.T) {
	db := newProviderTestDB(t)
	repo := NewOAuthProviderRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, sampleProvider("wechat")))

	got, err := repo.GetByCode(ctx, "wechat")
	require.NoError(t, err)
	assert.Equal(t, "wechat", got.Code)
	assert.Equal(t, model.ProviderKindWeChat, got.Kind)
	assert.Equal(t, "wx-app-id", got.ClientID)
	assert.Equal(t, "enc:v1:c2VjcmV0", got.ClientSecret, "secret 应以密文形式带回")
	assert.Equal(t, []string{"openid", "email"}, got.Scopes)
	assert.True(t, got.Enabled)
}

func TestProviderRepo_ListOrderedByCreatedAt(t *testing.T) {
	db := newProviderTestDB(t)
	repo := NewOAuthProviderRepository(db)
	ctx := context.Background()

	a := sampleProvider("google")
	a.Kind = model.ProviderKindGoogle
	a.CreatedAt = time.Now().Add(-time.Hour).Truncate(time.Second)
	b := sampleProvider("wechat")
	b.CreatedAt = time.Now().Truncate(time.Second)

	require.NoError(t, repo.Create(ctx, b))
	require.NoError(t, repo.Create(ctx, a))

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "google", list[0].Code, "应按创建时间升序")
	assert.Equal(t, "wechat", list[1].Code)
}

func TestProviderRepo_Update(t *testing.T) {
	db := newProviderTestDB(t)
	repo := NewOAuthProviderRepository(db)
	ctx := context.Background()

	p := sampleProvider("wechat")
	require.NoError(t, repo.Create(ctx, p))

	p.DisplayName = "微信登录"
	p.Enabled = false
	p.ClientID = "new-app-id"
	require.NoError(t, repo.Update(ctx, p))

	got, err := repo.GetByCode(ctx, "wechat")
	require.NoError(t, err)
	assert.Equal(t, "微信登录", got.DisplayName)
	assert.False(t, got.Enabled)
	assert.Equal(t, "new-app-id", got.ClientID)
}

func TestProviderRepo_UpdateStatus(t *testing.T) {
	db := newProviderTestDB(t)
	repo := NewOAuthProviderRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, sampleProvider("wechat")))
	require.NoError(t, repo.UpdateStatus(ctx, "wechat", false))

	got, err := repo.GetByCode(ctx, "wechat")
	require.NoError(t, err)
	assert.False(t, got.Enabled)
}

func TestProviderRepo_Delete(t *testing.T) {
	db := newProviderTestDB(t)
	repo := NewOAuthProviderRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, sampleProvider("wechat")))
	require.NoError(t, repo.DeleteByCode(ctx, "wechat"))

	_, err := repo.GetByCode(ctx, "wechat")
	assert.Error(t, err)

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count)
}

func TestProviderRepo_Count(t *testing.T) {
	db := newProviderTestDB(t)
	repo := NewOAuthProviderRepository(db)
	ctx := context.Background()

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, count)

	require.NoError(t, repo.Create(ctx, sampleProvider("wechat")))
	require.NoError(t, repo.Create(ctx, sampleProvider("google")))

	count, err = repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}
