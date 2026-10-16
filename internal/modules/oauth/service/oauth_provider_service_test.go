package service

import (
	"context"
	"strings"
	"testing"

	"meteorx/internal/config"
	"meteorx/internal/modules/oauth/dto"
	"meteorx/internal/modules/oauth/model"
	oauthRepo "meteorx/internal/modules/oauth/repository"
	"meteorx/pkg/crypto"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const testSecretKeyB64 = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=" // base64(32 bytes)

func newProviderSvcDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, oauthRepo.AutoMigrateProvider(db))
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	return db
}

func newProviderTestService(t *testing.T, cfg config.OAuthConfig) (*OAuthService, oauthRepo.OAuthProviderRepository) {
	t.Helper()
	db := newProviderSvcDB(t)
	repo := oauthRepo.NewOAuthProviderRepository(db)
	cipher, err := crypto.NewSecretCipher(cfg.SecretKey)
	require.NoError(t, err)
	svc := NewOAuthService(cfg, nil, nil, nil, nil, nil, nil, nil, nil, repo, cipher)
	return svc, repo
}

func wechatCfg(secretKey string) config.OAuthConfig {
	return config.OAuthConfig{
		SecretKey: secretKey,
		WeChat: config.OAuthProviderConfig{
			Enabled:      true,
			ClientID:     "wx-app-id",
			ClientSecret: "wx-secret",
			RedirectURL:  "http://localhost:5173/auth/wechat/callback",
		},
	}
}

func TestReloadProviders_SeedsFromConfig(t *testing.T) {
	svc, repo := newProviderTestService(t, wechatCfg(testSecretKeyB64))
	ctx := context.Background()

	require.NoError(t, svc.ReloadProviders(ctx))

	// 数据库应被种子化
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Greater(t, count, int64(0))

	// 注册表应含启用的 wechat
	p, ok := svc.registry.Get("wechat")
	require.True(t, ok)
	assert.True(t, p.IsEnabled())
	assert.Contains(t, p.RedirectURL("st"), "wx-app-id")

	// secret 落库应为密文
	row, err := repo.GetByCode(ctx, "wechat")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(row.ClientSecret, crypto.CipherPrefix), "secret 必须加密存储")
	assert.NotContains(t, row.ClientSecret, "wx-secret")
}

func TestReloadProviders_DBOverridesConfig(t *testing.T) {
	cfg := wechatCfg(testSecretKeyB64)
	svc, repo := newProviderTestService(t, cfg)
	ctx := context.Background()

	// 预置一条「禁用」的 wechat，应覆盖 config.yaml 中的启用态
	cipher, _ := crypto.NewSecretCipher(testSecretKeyB64)
	enc, err := cipher.Encrypt("db-secret")
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, &model.OAuthProvider{
		ID: "id-1", Kind: model.ProviderKindWeChat, Code: "wechat",
		Enabled: false, ClientID: "db-app-id", ClientSecret: enc,
		RedirectURL: "http://localhost:5173/auth/wechat/callback",
	}))

	require.NoError(t, svc.ReloadProviders(ctx))

	p, ok := svc.registry.Get("wechat")
	require.True(t, ok)
	assert.False(t, p.IsEnabled(), "DB 配置应覆盖 config.yaml")
	assert.Contains(t, p.RedirectURL("st"), "db-app-id")

	// 未再次种子化（表非空）
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestReloadProviders_NoStoreIsNoop(t *testing.T) {
	svc := newTestOAuthService(wechatCfg(testSecretKeyB64))
	before := svc.registry

	require.NoError(t, svc.ReloadProviders(context.Background()))
	assert.Same(t, before, svc.registry, "未装配 DB 存储时应保持原注册表")
}

func TestCreateProvider_Validation(t *testing.T) {
	svc, _ := newProviderTestService(t, config.OAuthConfig{SecretKey: testSecretKeyB64})
	ctx := context.Background()

	cases := []struct {
		name string
		req  dto.SaveOAuthProviderReq
		want error
	}{
		{
			name: "不支持的类型",
			req:  dto.SaveOAuthProviderReq{Kind: "unknown", Code: "unknown", ClientID: "a", RedirectURL: "https://e/c"},
			want: ErrOAuthProviderInvalid,
		},
		{
			name: "固定类型标识不匹配",
			req:  dto.SaveOAuthProviderReq{Kind: "wechat", Code: "wechat2", ClientID: "a", RedirectURL: "https://e/c"},
			want: ErrOAuthProviderInvalid,
		},
		{
			name: "缺少 client_id",
			req:  dto.SaveOAuthProviderReq{Kind: "wechat", Code: "wechat", RedirectURL: "https://e/c"},
			want: ErrOAuthProviderInvalid,
		},
		{
			name: "缺少 redirect_url",
			req:  dto.SaveOAuthProviderReq{Kind: "wechat", Code: "wechat", ClientID: "a"},
			want: ErrOAuthProviderInvalid,
		},
		{
			name: "OIDC 缺少端点",
			req:  dto.SaveOAuthProviderReq{Kind: "oidc", Code: "oidc_acme", ClientID: "a", RedirectURL: "https://e/c"},
			want: ErrOAuthProviderInvalid,
		},
		{
			name: "OIDC 标识非法",
			req: dto.SaveOAuthProviderReq{Kind: "oidc", Code: "oidc_Ab", ClientID: "a", RedirectURL: "https://e/c",
				AuthURL: "https://i/a", TokenURL: "https://i/t", UserInfoURL: "https://i/u"},
			want: ErrOAuthProviderInvalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.CreateOAuthProviderConfig(ctx, tc.req)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestCreateProvider_RejectsPlaintextWithoutKey(t *testing.T) {
	svc, _ := newProviderTestService(t, config.OAuthConfig{}) // 未配置 secret_key
	err := svc.CreateOAuthProviderConfig(context.Background(), dto.SaveOAuthProviderReq{
		Kind: "wechat", Code: "wechat", ClientID: "a", ClientSecret: "s",
		RedirectURL: "https://e/c",
	})
	assert.ErrorIs(t, err, ErrOAuthSecretKeyNotConfigured)
}

func TestCreateProvider_Success(t *testing.T) {
	svc, repo := newProviderTestService(t, config.OAuthConfig{SecretKey: testSecretKeyB64})
	ctx := context.Background()

	require.NoError(t, svc.CreateOAuthProviderConfig(ctx, dto.SaveOAuthProviderReq{
		Kind: "wechat", Code: "wechat", DisplayName: "微信登录", Enabled: true,
		ClientID: "wx-id", ClientSecret: "wx-secret",
		RedirectURL: "http://localhost:5173/auth/wechat/callback",
	}))

	// 写入后应热生效
	p, ok := svc.registry.Get("wechat")
	require.True(t, ok)
	assert.True(t, p.IsEnabled())

	// secret 密文存储
	row, err := repo.GetByCode(ctx, "wechat")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(row.ClientSecret, crypto.CipherPrefix))

	// 重复创建同标识应冲突
	err = svc.CreateOAuthProviderConfig(ctx, dto.SaveOAuthProviderReq{
		Kind: "wechat", Code: "wechat", ClientID: "x", RedirectURL: "https://e/c",
	})
	assert.ErrorIs(t, err, ErrOAuthProviderCodeTaken)
}

func TestCreateProvider_OIDC(t *testing.T) {
	svc, _ := newProviderTestService(t, config.OAuthConfig{SecretKey: testSecretKeyB64})
	ctx := context.Background()

	require.NoError(t, svc.CreateOAuthProviderConfig(ctx, dto.SaveOAuthProviderReq{
		Kind: "oidc", Code: "oidc_acme", DisplayName: "Acme SSO", Enabled: true,
		ClientID: "acme-id", ClientSecret: "acme-secret",
		RedirectURL: "http://localhost:5173/auth/oidc_acme/callback",
		AuthURL:     "https://idp.example.com/authorize",
		TokenURL:    "https://idp.example.com/token",
		UserInfoURL: "https://idp.example.com/userinfo",
		Scopes:      []string{"openid", "email"},
	}))

	p, ok := svc.registry.Get("oidc_acme")
	require.True(t, ok)
	assert.True(t, p.IsEnabled())
	assert.Contains(t, p.RedirectURL("st"), "idp.example.com/authorize")

	metas := svc.ListProviders(ctx)
	require.Len(t, metas, 1)
	assert.Equal(t, "oidc_acme", metas[0].ID)
	assert.Equal(t, "Acme SSO", metas[0].Label, "应优先使用配置的展示名")
}

func TestUpdateProvider_KeepsSecretWhenOmitted(t *testing.T) {
	svc, repo := newProviderTestService(t, config.OAuthConfig{SecretKey: testSecretKeyB64})
	ctx := context.Background()

	require.NoError(t, svc.CreateOAuthProviderConfig(ctx, dto.SaveOAuthProviderReq{
		Kind: "wechat", Code: "wechat", ClientID: "wx-id", ClientSecret: "original-secret",
		RedirectURL: "https://e/c",
	}))

	// 更新但不传 secret
	require.NoError(t, svc.UpdateOAuthProviderConfig(ctx, "wechat", dto.SaveOAuthProviderReq{
		Kind: "wechat", Code: "wechat", ClientID: "wx-id-new",
		RedirectURL: "https://e/c2",
	}))

	row, err := repo.GetByCode(ctx, "wechat")
	require.NoError(t, err)
	cipher, _ := crypto.NewSecretCipher(testSecretKeyB64)
	plain, err := cipher.Decrypt(row.ClientSecret)
	require.NoError(t, err)
	assert.Equal(t, "original-secret", plain, "未提供 secret 时应沿用旧值")

	// 传了新 secret 则替换
	require.NoError(t, svc.UpdateOAuthProviderConfig(ctx, "wechat", dto.SaveOAuthProviderReq{
		Kind: "wechat", Code: "wechat", ClientID: "wx-id-new", ClientSecret: "rotated-secret",
		RedirectURL: "https://e/c2",
	}))
	row, err = repo.GetByCode(ctx, "wechat")
	require.NoError(t, err)
	plain, err = cipher.Decrypt(row.ClientSecret)
	require.NoError(t, err)
	assert.Equal(t, "rotated-secret", plain)
}

func TestDeleteProvider(t *testing.T) {
	svc, _ := newProviderTestService(t, config.OAuthConfig{SecretKey: testSecretKeyB64})
	ctx := context.Background()

	require.NoError(t, svc.CreateOAuthProviderConfig(ctx, dto.SaveOAuthProviderReq{
		Kind: "wechat", Code: "wechat", ClientID: "a", RedirectURL: "https://e/c",
	}))
	require.NoError(t, svc.DeleteOAuthProviderConfig(ctx, "wechat"))

	_, ok := svc.registry.Get("wechat")
	assert.False(t, ok, "删除后应从注册表移除")

	err := svc.DeleteOAuthProviderConfig(ctx, "wechat")
	assert.ErrorIs(t, err, ErrOAuthProviderNotFound)
}

func TestSetProviderStatus(t *testing.T) {
	svc, _ := newProviderTestService(t, config.OAuthConfig{SecretKey: testSecretKeyB64})
	ctx := context.Background()

	require.NoError(t, svc.CreateOAuthProviderConfig(ctx, dto.SaveOAuthProviderReq{
		Kind: "wechat", Code: "wechat", Enabled: true, ClientID: "a", RedirectURL: "https://e/c",
	}))
	p, _ := svc.registry.Get("wechat")
	require.True(t, p.IsEnabled())

	require.NoError(t, svc.SetOAuthProviderConfigStatus(ctx, "wechat", false))
	p, _ = svc.registry.Get("wechat")
	assert.False(t, p.IsEnabled())

	assert.Empty(t, svc.ListProviders(ctx), "禁用后不应出现在已启用列表")

	err := svc.SetOAuthProviderConfigStatus(ctx, "nope", true)
	assert.ErrorIs(t, err, ErrOAuthProviderNotFound)
}

func TestListProviderConfigs_MasksSecret(t *testing.T) {
	svc, _ := newProviderTestService(t, config.OAuthConfig{SecretKey: testSecretKeyB64})
	ctx := context.Background()

	require.NoError(t, svc.CreateOAuthProviderConfig(ctx, dto.SaveOAuthProviderReq{
		Kind: "wechat", Code: "wechat", DisplayName: "微信", Enabled: true,
		ClientID: "wx-id", ClientSecret: "super-secret-value",
		RedirectURL: "https://e/c",
	}))

	list, err := svc.ListOAuthProviderConfigs(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)

	resp := list[0]
	assert.Equal(t, "wechat", resp.Code)
	assert.Equal(t, "微信", resp.DisplayName)
	assert.True(t, resp.HasSecret)
	assert.NotContains(t, resp.ClientSecretMask, "super-secret-value", "禁止返回明文 secret")
	assert.True(t, strings.HasSuffix(resp.ClientSecretMask, "alue"))
	assert.False(t, resp.EditableEndpoint, "固定 provider 不允许自配端点")
}
