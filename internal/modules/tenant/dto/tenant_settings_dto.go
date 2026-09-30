package dto

// TenantSettingsResp 租户设置响应
type TenantSettingsResp struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	Logo         string `json:"logo"`
	Favicon      string `json:"favicon"`
	PrimaryColor string `json:"primary_color"`
	Theme        string `json:"theme"`
	Language     string `json:"language"`
	Timezone     string `json:"timezone"`
	Description  string `json:"description"`
	WelcomeText  string `json:"welcome_text"`
	ContactName  string `json:"contact_name"`
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`
	Address      string `json:"address"`
	Extra        string `json:"extra"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// UpdateTenantSettingsReq 更新租户设置请求（均为可选字段）
type UpdateTenantSettingsReq struct {
	Logo         string `json:"logo"`
	Favicon      string `json:"favicon"`
	PrimaryColor string `json:"primary_color"`
	Theme        string `json:"theme"`
	Language     string `json:"language"`
	Timezone     string `json:"timezone"`
	Description  string `json:"description"`
	WelcomeText  string `json:"welcome_text"`
	ContactName  string `json:"contact_name"`
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`
	Address      string `json:"address"`
	Extra        string `json:"extra"`
}

// ToSettings 将更新请求转换为租户设置响应，并填充目标租户ID
func (s *UpdateTenantSettingsReq) ToSettings(tenantID string) *TenantSettingsResp {
	return &TenantSettingsResp{
		TenantID:     tenantID,
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
}
