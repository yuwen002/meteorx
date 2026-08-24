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

type TenantSettingsService struct {
	repo repository.TenantSettingsRepository
}

func NewTenantSettingsService(repo repository.TenantSettingsRepository) *TenantSettingsService {
	return &TenantSettingsService{repo: repo}
}

func (s *TenantSettingsService) GetSettings(ctx context.Context, tenantID string) (*dto.TenantSettingsResp, error) {
	settings, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if settings == nil {
		return s.getDefaultSettings(tenantID), nil
	}

	return toSettingsResp(settings), nil
}

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

	return toSettingsResp(updated), nil
}

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