package dto

import "meteorx/internal/modules/tenant/model"

// TenantConverter 租户模型到 DTO 的转换器
type TenantConverter struct{}

// ToTenantResponse 将租户领域模型转换为租户响应 DTO
func (a *TenantConverter) ToTenantResponse(tenant *model.Tenant) *TenantResp {
	return &TenantResp{
		ID:           tenant.ID,
		Name:         tenant.Name,
		Domain:       tenant.Domain,
		Status:       tenant.Status,
		Description:  tenant.Description,
		ContactEmail: tenant.ContactEmail,
		Region:       tenant.Region,
		Logo:         tenant.Logo,
		Extra:        tenant.Extra,
		CreatedAt:    tenant.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    tenant.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
