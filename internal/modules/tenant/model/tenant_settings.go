package model

import "time"

// TenantSettings 租户个性化设置领域模型（品牌/主题/语言时区/联系方式等）。
type TenantSettings struct {
	ID           string
	TenantID     string
	Logo         string
	Favicon      string
	PrimaryColor string
	Theme        string
	Language     string
	Timezone     string
	Description  string
	WelcomeText  string
	ContactName  string
	ContactEmail string
	ContactPhone string
	Address      string
	Extra        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
