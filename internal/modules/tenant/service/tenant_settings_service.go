package service

import (
	"context"
	"errors"
	"time"

	"meteorx/internal/modules/tenant/dto"
	"meteorx/internal/modules/tenant/model"
	"meteorx/internal/modules/tenant/repository"
	"meteorx/pkg/idgen"
)

// tenantNameProvider 抽象获取租户名称的能力（TenantRepository 实现），用于白标下发品牌显示名。
type tenantNameProvider interface {
	GetByID(ctx context.Context, id string) (*model.Tenant, error)
}

// TenantSettingsService 租户个性化设置业务服务。
type TenantSettingsService struct {
	repo       repository.TenantSettingsRepository
	nameGetter tenantNameProvider // 可选：注入后 GetSettings/UpdateSettings 会 best-effort 填充 tenant_name
}

// SettingsOption 租户设置服务的可变参数选项。
type SettingsOption func(*TenantSettingsService)

// WithTenantNameProvider 注入租户名称提供者，用于白标下发品牌显示名（未注入时跳过）。
func WithTenantNameProvider(p tenantNameProvider) SettingsOption {
	return func(s *TenantSettingsService) { s.nameGetter = p }
}

// NewTenantSettingsService 创建租户设置服务实例。
func NewTenantSettingsService(repo repository.TenantSettingsRepository, opts ...SettingsOption) *TenantSettingsService {
	s := &TenantSettingsService{repo: repo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// fillTenantName best-effort 填充租户显示名：未注入 provider 或查询失败时保持空值，不影响主流程。
func (s *TenantSettingsService) fillTenantName(ctx context.Context, tenantID string, resp *dto.TenantSettingsResp) {
	if s.nameGetter == nil || resp == nil || tenantID == "" {
		return
	}
	if t, err := s.nameGetter.GetByID(ctx, tenantID); err == nil && t != nil {
		resp.TenantName = t.Name
	}
}

// GetSettings 获取租户设置，不存在时返回默认值。
func (s *TenantSettingsService) GetSettings(ctx context.Context, tenantID string) (*dto.TenantSettingsResp, error) {
	settings, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var resp *dto.TenantSettingsResp
	if settings == nil {
		resp = s.getDefaultSettings(tenantID)
	} else {
		resp = toSettingsResp(settings)
	}
	s.fillTenantName(ctx, tenantID, resp)
	return resp, nil
}

// UpdateSettings 更新租户设置，不存在则创建（upsert）后返回最新值。
func (s *TenantSettingsService) UpdateSettings(ctx context.Context, tenantID string, req dto.UpdateTenantSettingsReq) (*dto.TenantSettingsResp, error) {
	existing, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	settings := &model.TenantSettings{
		TenantID:     tenantID,
		Logo:         req.Logo,
		Favicon:      req.Favicon,
		PrimaryColor: req.PrimaryColor,
		Theme:        req.Theme,
		Language:     req.Language,
		Timezone:     req.Timezone,
		Description:  req.Description,
		WelcomeText:  req.WelcomeText,
		ContactName:  req.ContactName,
		ContactEmail: req.ContactEmail,
		ContactPhone: req.ContactPhone,
		Address:      req.Address,
		Extra:        req.Extra,
	}

	if existing == nil {
		settings.ID = idgen.New()
		now := time.Now()
		settings.CreatedAt = now
		settings.UpdatedAt = now

		if err := s.repo.Create(ctx, settings); err != nil {
			return nil, err
		}
	} else {
		settings.ID = existing.ID
		if err := s.repo.Update(ctx, settings); err != nil {
			return nil, err
		}
	}

	updated, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if updated == nil {
		return nil, errors.New("settings not found after update")
	}

	resp := toSettingsResp(updated)
	s.fillTenantName(ctx, tenantID, resp)
	return resp, nil
}

// getDefaultSettings 构造一份默认租户设置。
func (s *TenantSettingsService) getDefaultSettings(tenantID string) *dto.TenantSettingsResp {
	return &dto.TenantSettingsResp{
		TenantID:     tenantID,
		Logo:         "",
		Favicon:      "",
		PrimaryColor: "#667eea",
		Theme:        "light",
		Language:     "zh-CN",
		Timezone:     "Asia/Shanghai",
		Description:  "",
		WelcomeText:  "欢迎使用 MeteorX 多租户平台",
		ContactName:  "",
		ContactEmail: "",
		ContactPhone: "",
		Address:      "",
		Extra:        "",
	}
}

// toSettingsResp 将领域模型转换为响应 DTO。
func toSettingsResp(s *model.TenantSettings) *dto.TenantSettingsResp {
	resp := &dto.TenantSettingsResp{
		ID:           s.ID,
		TenantID:     s.TenantID,
		Logo:         s.Logo,
		Favicon:      s.Favicon,
		PrimaryColor: s.PrimaryColor,
		Theme:        s.Theme,
		Language:     s.Language,
		Timezone:     s.Timezone,
		Description:  s.Description,
		WelcomeText:  s.WelcomeText,
		ContactName:  s.ContactName,
		ContactEmail: s.ContactEmail,
		ContactPhone: s.ContactPhone,
		Address:      s.Address,
		Extra:        s.Extra,
	}
	if !s.CreatedAt.IsZero() {
		resp.CreatedAt = s.CreatedAt.Format("2006-01-02 15:04:05")
	}
	if !s.UpdatedAt.IsZero() {
		resp.UpdatedAt = s.UpdatedAt.Format("2006-01-02 15:04:05")
	}
	return resp
}
