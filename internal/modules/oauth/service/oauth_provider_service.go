package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"meteorx/internal/modules/oauth/dto"
	"meteorx/internal/modules/oauth/model"
	oauthRepo "meteorx/internal/modules/oauth/repository"
	"meteorx/pkg/crypto"
	"meteorx/pkg/idgen"
	"meteorx/pkg/logger"
)

// OAuth 提供商运行时配置相关错误
var (
	ErrOAuthProviderNotFound    = errors.New("oauth provider config not found")
	ErrOAuthProviderCodeTaken   = errors.New("oauth provider code already exists")
	ErrOAuthProviderInvalid     = errors.New("invalid oauth provider input")
	ErrOAuthSecretKeyNotConfigured = errors.New("未配置 OAuth 凭据加密主密钥（oauth.secret_key），拒绝保存明文 client_secret")
)

// oidcCodePattern 通用 OIDC 的 provider 标识规则（最终 Code 形如 oidc_<key>，此处校验 key 部分）
var oidcCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,63}$`)

// ProviderReloadInterval 注册表定时刷新间隔：多副本部署下任一副本写入后，其余副本在此时间内收敛。
const ProviderReloadInterval = 30 * time.Second

// ReloadProviders 从数据库重建 provider 注册表（热更新入口）。
// 首次运行且表为空时，先用 config.yaml 的静态配置种子化，保证既有部署平滑升级。
// 未装配 DB 存储时（测试/降级）为 no-op，沿用 config 注册表。
func (s *OAuthService) ReloadProviders(ctx context.Context) error {
	if s.providerRepo == nil {
		return nil
	}

	count, err := s.providerRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("count oauth providers: %w", err)
	}
	if count == 0 && !s.providersSeeded(ctx) {
		if err := s.seedProvidersFromConfig(ctx); err != nil {
			return err
		}
		if err := s.providerRepo.SetMeta(ctx, oauthRepo.MetaKeySeededFromConfig, time.Now().Format(time.RFC3339)); err != nil {
			logger.Warnf("[oauth] mark provider seed failed: %v", err)
		}
	}

	rows, err := s.providerRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list oauth providers: %w", err)
	}

	specs := make([]ProviderSpec, 0, len(rows))
	for _, row := range rows {
		secret := row.ClientSecret
		if secret != "" {
			plain, derr := s.cipher.Decrypt(secret)
			if derr != nil {
				logger.Warnf("[oauth] decrypt client_secret for %s failed: %v", row.Code, derr)
				continue
			}
			secret = plain
		}
		specs = append(specs, ProviderSpec{
			Kind:         row.Kind,
			Code:         row.Code,
			DisplayName:  row.DisplayName,
			Enabled:      row.Enabled,
			ClientID:     row.ClientID,
			ClientSecret: secret,
			RedirectURL:  row.RedirectURL,
			Tenant:       row.Tenant,
			AuthURL:      row.AuthURL,
			TokenURL:     row.TokenURL,
			UserInfoURL:  row.UserInfoURL,
			Scopes:       row.Scopes,
		})
	}

	fresh := NewProviderRegistryFromSpecs(specs)
	s.registry.Replace(fresh.providers, fresh.labels)
	return nil
}

// providersSeeded 是否已完成过 config.yaml 种子化（读不到元数据视为未种子化）。
func (s *OAuthService) providersSeeded(ctx context.Context) bool {
	v, err := s.providerRepo.GetMeta(ctx, oauthRepo.MetaKeySeededFromConfig)
	return err == nil && v != ""
}

// seedProvidersFromConfig 把 config.yaml 的 provider 配置写入数据库（仅在从未种子化时调用，幂等）。
func (s *OAuthService) seedProvidersFromConfig(ctx context.Context) error {
	seeded := 0
	for _, spec := range ProviderSpecsFromConfig(s.cfg) {
		if spec.ClientID == "" {
			continue // 未填写 client_id 即未真正配置，不种子化（避免空壳条目）
		}

		secretEnc := spec.ClientSecret
		if secretEnc != "" {
			if !s.cipher.Enabled() {
				logger.Warnf("[oauth] secret key not configured, storing client_secret of %s in plaintext", spec.Code)
			} else {
				enc, err := s.cipher.Encrypt(secretEnc)
				if err != nil {
					return fmt.Errorf("encrypt seed secret for %s: %w", spec.Code, err)
				}
				secretEnc = enc
			}
		}

		now := time.Now()
		row := &model.OAuthProvider{
			ID:           idgen.New(),
			Kind:         spec.Kind,
			Code:         spec.Code,
			DisplayName:  spec.DisplayName,
			Enabled:      spec.Enabled,
			ClientID:     spec.ClientID,
			ClientSecret: secretEnc,
			RedirectURL:  spec.RedirectURL,
			Tenant:       spec.Tenant,
			AuthURL:      spec.AuthURL,
			TokenURL:     spec.TokenURL,
			UserInfoURL:  spec.UserInfoURL,
			Scopes:       spec.Scopes,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := s.providerRepo.Create(ctx, row); err != nil {
			// 并发种子化导致唯一键冲突时忽略即可
			logger.Warnf("[oauth] seed provider %s skipped: %v", spec.Code, err)
			continue
		}
		seeded++
	}
	if seeded > 0 {
		logger.Infof("[oauth] seeded %d provider config(s) from config.yaml", seeded)
	}
	return nil
}

// ListOAuthProviderConfigs 列出全部提供商配置（脱敏）。
func (s *OAuthService) ListOAuthProviderConfigs(ctx context.Context) ([]dto.OAuthProviderResp, error) {
	if s.providerRepo == nil {
		return []dto.OAuthProviderResp{}, nil
	}
	rows, err := s.providerRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.OAuthProviderResp, 0, len(rows))
	for _, row := range rows {
		resp, err := s.toProviderResp(ctx, row)
		if err != nil {
			return nil, err
		}
		out = append(out, *resp)
	}
	return out, nil
}

// GetOAuthProviderConfig 按标识查询单个提供商配置（脱敏）。
func (s *OAuthService) GetOAuthProviderConfig(ctx context.Context, code string) (*dto.OAuthProviderResp, error) {
	row, err := s.requireProviderRow(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.toProviderResp(ctx, row)
}

// CreateOAuthProviderConfig 新建提供商配置并热生效。
func (s *OAuthService) CreateOAuthProviderConfig(ctx context.Context, req dto.SaveOAuthProviderReq) error {
	if s.providerRepo == nil {
		return errors.New("oauth provider store not available")
	}
	spec, err := s.normalizeProviderReq(ctx, "", req)
	if err != nil {
		return err
	}

	now := time.Now()
	row := &model.OAuthProvider{
		ID:           idgen.New(),
		Kind:         spec.Kind,
		Code:         spec.Code,
		DisplayName:  spec.DisplayName,
		Enabled:      spec.Enabled,
		ClientID:     spec.ClientID,
		ClientSecret: spec.ClientSecret, // normalizeProviderReq 已加密
		RedirectURL:  spec.RedirectURL,
		Tenant:       spec.Tenant,
		AuthURL:      spec.AuthURL,
		TokenURL:     spec.TokenURL,
		UserInfoURL:  spec.UserInfoURL,
		Scopes:       spec.Scopes,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.providerRepo.Create(ctx, row); err != nil {
		if isDuplicateKeyErr(err) {
			return ErrOAuthProviderCodeTaken
		}
		return err
	}
	return s.ReloadProviders(ctx)
}

// UpdateOAuthProviderConfig 更新提供商配置并热生效。client_secret 传空表示保持原值。
func (s *OAuthService) UpdateOAuthProviderConfig(ctx context.Context, code string, req dto.SaveOAuthProviderReq) error {
	if s.providerRepo == nil {
		return errors.New("oauth provider store not available")
	}
	existing, err := s.requireProviderRow(ctx, code)
	if err != nil {
		return err
	}

	spec, err := s.normalizeProviderReq(ctx, code, req)
	if err != nil {
		return err
	}
	if spec.ClientSecret == "" {
		spec.ClientSecret = existing.ClientSecret // 未提供则沿用已加密的旧值
	}

	existing.Kind = spec.Kind
	existing.DisplayName = spec.DisplayName
	existing.Enabled = spec.Enabled
	existing.ClientID = spec.ClientID
	existing.ClientSecret = spec.ClientSecret
	existing.RedirectURL = spec.RedirectURL
	existing.Tenant = spec.Tenant
	existing.AuthURL = spec.AuthURL
	existing.TokenURL = spec.TokenURL
	existing.UserInfoURL = spec.UserInfoURL
	existing.Scopes = spec.Scopes
	existing.UpdatedAt = time.Now()

	if err := s.providerRepo.Update(ctx, existing); err != nil {
		return err
	}
	return s.ReloadProviders(ctx)
}

// DeleteOAuthProviderConfig 删除提供商配置并热生效。
func (s *OAuthService) DeleteOAuthProviderConfig(ctx context.Context, code string) error {
	if s.providerRepo == nil {
		return errors.New("oauth provider store not available")
	}
	if _, err := s.requireProviderRow(ctx, code); err != nil {
		return err
	}
	if err := s.providerRepo.DeleteByCode(ctx, code); err != nil {
		return err
	}
	return s.ReloadProviders(ctx)
}

// SetOAuthProviderConfigStatus 切换启用状态并热生效。
func (s *OAuthService) SetOAuthProviderConfigStatus(ctx context.Context, code string, enabled bool) error {
	if s.providerRepo == nil {
		return errors.New("oauth provider store not available")
	}
	if _, err := s.requireProviderRow(ctx, code); err != nil {
		return err
	}
	if err := s.providerRepo.UpdateStatus(ctx, code, enabled); err != nil {
		return err
	}
	return s.ReloadProviders(ctx)
}

// requireProviderRow 按标识取配置，不存在返回 ErrOAuthProviderNotFound
func (s *OAuthService) requireProviderRow(ctx context.Context, code string) (*model.OAuthProvider, error) {
	if s.providerRepo == nil {
		return nil, errors.New("oauth provider store not available")
	}
	row, err := s.providerRepo.GetByCode(ctx, code)
	if err != nil || row == nil {
		return nil, ErrOAuthProviderNotFound
	}
	return row, nil
}

// normalizeProviderReq 校验并归一化请求；返回的 spec.ClientSecret 已是密文（或空表示沿用旧值）。
func (s *OAuthService) normalizeProviderReq(ctx context.Context, currentCode string, req dto.SaveOAuthProviderReq) (ProviderSpec, error) {
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if !model.ValidProviderKind(kind) {
		return ProviderSpec{}, fmt.Errorf("%w: 不支持的 provider 类型 %q", ErrOAuthProviderInvalid, req.Kind)
	}

	code := strings.ToLower(strings.TrimSpace(req.Code))
	if code == "" {
		return ProviderSpec{}, fmt.Errorf("%w: provider 标识不能为空", ErrOAuthProviderInvalid)
	}
	if model.NeedsEndpoints(kind) {
		key := strings.TrimPrefix(code, "oidc_")
		if !oidcCodePattern.MatchString(key) {
			return ProviderSpec{}, fmt.Errorf("%w: OIDC 标识需为 3-64 位小写字母/数字/下划线且以字母开头（如 oidc_acme）", ErrOAuthProviderInvalid)
		}
		if !strings.HasPrefix(code, "oidc_") {
			code = "oidc_" + key
		}
	} else if code != kind {
		return ProviderSpec{}, fmt.Errorf("%w: %s 的标识必须为 %s", ErrOAuthProviderInvalid, kind, kind)
	}

	if code != currentCode {
		if _, err := s.providerRepo.GetByCode(ctx, code); err == nil {
			return ProviderSpec{}, ErrOAuthProviderCodeTaken
		}
	}

	if strings.TrimSpace(req.ClientID) == "" {
		return ProviderSpec{}, fmt.Errorf("%w: client_id 不能为空", ErrOAuthProviderInvalid)
	}
	if strings.TrimSpace(req.RedirectURL) == "" {
		return ProviderSpec{}, fmt.Errorf("%w: redirect_url 不能为空", ErrOAuthProviderInvalid)
	}

	spec := ProviderSpec{
		Kind:         kind,
		Code:         code,
		DisplayName:  strings.TrimSpace(req.DisplayName),
		Enabled:      req.Enabled,
		ClientID:     strings.TrimSpace(req.ClientID),
		RedirectURL:  strings.TrimSpace(req.RedirectURL),
		Tenant:       strings.TrimSpace(req.Tenant),
		AuthURL:      strings.TrimSpace(req.AuthURL),
		TokenURL:     strings.TrimSpace(req.TokenURL),
		UserInfoURL:  strings.TrimSpace(req.UserInfoURL),
		Scopes:       req.Scopes,
	}

	if model.NeedsEndpoints(kind) {
		if spec.AuthURL == "" || spec.TokenURL == "" || spec.UserInfoURL == "" {
			return ProviderSpec{}, fmt.Errorf("%w: 通用 OIDC 必须填写 auth_url / token_url / userinfo_url", ErrOAuthProviderInvalid)
		}
		for _, u := range []string{spec.AuthURL, spec.TokenURL, spec.UserInfoURL} {
			if !isHTTPURL(u) {
				return ProviderSpec{}, fmt.Errorf("%w: 端点必须为 http/https 地址", ErrOAuthProviderInvalid)
			}
		}
	}

	// client_secret：传了就必须加密落库；未配置主密钥则拒绝保存明文
	if strings.TrimSpace(req.ClientSecret) != "" {
		if !s.cipher.Enabled() {
			return ProviderSpec{}, ErrOAuthSecretKeyNotConfigured
		}
		enc, err := s.cipher.Encrypt(strings.TrimSpace(req.ClientSecret))
		if err != nil {
			return ProviderSpec{}, fmt.Errorf("encrypt client_secret: %w", err)
		}
		spec.ClientSecret = enc
	}

	return spec, nil
}

// toProviderResp 转换为响应 DTO，secret 仅返回脱敏值。
func (s *OAuthService) toProviderResp(ctx context.Context, row *model.OAuthProvider) (*dto.OAuthProviderResp, error) {
	masked := ""
	if row.ClientSecret != "" {
		plain, err := s.cipher.Decrypt(row.ClientSecret)
		if err != nil {
			logger.Warnf("[oauth] decrypt client_secret for %s failed: %v", row.Code, err)
			masked = "****（解密失败）"
		} else {
			masked = crypto.MaskSecret(plain)
		}
	}
	return &dto.OAuthProviderResp{
		Code:             row.Code,
		Kind:             row.Kind,
		DisplayName:      displayName(row.Code, row.DisplayName),
		Enabled:          row.Enabled,
		ClientID:         row.ClientID,
		ClientSecretMask: masked,
		HasSecret:        row.ClientSecret != "",
		RedirectURL:      row.RedirectURL,
		Tenant:           row.Tenant,
		AuthURL:          row.AuthURL,
		TokenURL:         row.TokenURL,
		UserInfoURL:      row.UserInfoURL,
		Scopes:           row.Scopes,
		EditableEndpoint: model.NeedsEndpoints(row.Kind),
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}, nil
}

// displayName 优先使用配置的展示名，否则按内置规则推导
func displayName(code, configured string) string {
	if configured != "" {
		return configured
	}
	return defaultDisplayName(code)
}

// isHTTPURL 判断是否为合法的 http/https 地址
func isHTTPURL(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// isDuplicateKeyErr 判断是否为唯一键冲突错误（MySQL 1062 / SQLite UNIQUE constraint）
func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique constraint")
}
