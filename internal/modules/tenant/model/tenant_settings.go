package model

import "time"

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