package model

import "time"

type OAuthAccount struct {
	ID          string
	UserID      string
	Provider    string
	ProviderID  string
	Email       string
	AccessToken string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const (
	ProviderGoogle = "google"
	ProviderGitHub = "github"
)