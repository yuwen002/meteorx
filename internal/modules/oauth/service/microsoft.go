package service

import (
	"fmt"

	"meteorx/internal/config"
	"meteorx/internal/modules/oauth/model"
)

// newMicrosoftProvider 构造 Microsoft Entra ID（Azure AD）provider。
// 授权/令牌端点需包含目录租户 ID，用户信息走 Microsoft Graph OIDC 端点。
func newMicrosoftProvider(c config.MicrosoftOAuthConfig) *standardOAuthProvider {
	tenant := c.Tenant
	if tenant == "" {
		tenant = "common"
	}
	base := "https://login.microsoftonline.com/" + tenant + "/oauth2/v2.0"
	return &standardOAuthProvider{
		name:         model.ProviderMicrosoft,
		enabled:      c.Enabled,
		clientID:     c.ClientID,
		clientSecret: c.ClientSecret,
		redirectURL:  c.RedirectURL,
		authURL:      fmt.Sprintf("%s/authorize", base),
		tokenURL:     fmt.Sprintf("%s/token", base),
		userInfoURL:  "https://graph.microsoft.com/oidc/userinfo",
		scopes:       []string{"openid", "email", "profile"},
	}
}
