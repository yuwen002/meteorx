// Package jwt 提供 JWT 令牌的签发、解析和校验功能，支持 Access Token 和 Refresh Token。
package jwt

import (
	"errors"
	"time"

	"meteorx/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

// TokenHelper JWT 签发与解析助手，封装密钥、过期时长与签发者。
type TokenHelper struct {
	secret     []byte
	expiration time.Duration
	issuer     string
}

// CustomClaims 自定义载荷
type CustomClaims struct {
	UserID       string   `json:"user_id"`
	TenantID     string   `json:"tenant_id"`
	Roles        []string `json:"roles"`
	EmailPending bool     `json:"email_pending,omitempty"` // 第三方登录尚未补全邮箱时的临时态
	jwt.RegisteredClaims
}

// RoleSuperadmin 平台超级管理员角色编码，与中间件 RequiresMasterAdmin / 权限豁免逻辑的判定口径一致。
const RoleSuperadmin = "superadmin"

// EnsureSuperadminRole 保证主账号（is_master）用户的角色列表始终包含 superadmin 角色。
// 背景：主账号被显式分配其他角色后，原“无角色才兜底”的逻辑会使 JWT 缺失 superadmin，
// 导致 is_master 可见的超管端点反而返回 403（前后端超管口径不一致）。
func EnsureSuperadminRole(isMaster bool, roleCodes []string) []string {
	if !isMaster {
		return roleCodes
	}
	for _, c := range roleCodes {
		if c == RoleSuperadmin {
			return roleCodes
		}
	}
	return append([]string{RoleSuperadmin}, roleCodes...)
}

// NewTokenHelper 关联配置并初始化助手
func NewTokenHelper(cfg config.JWTConfig) *TokenHelper {
	return &TokenHelper{
		secret:     []byte(cfg.Secret),
		expiration: cfg.GetExpiration(),
		issuer:     cfg.Issuer,
	}
}

// GenerateToken 生成 Token
func (h *TokenHelper) GenerateToken(userID, tenantID string, roles []string) (string, error) {
	return h.GenerateTokenEx(userID, tenantID, roles, false)
}

// GenerateTokenEx 生成 Token，emailPending 为 true 时写入 email_pending 声明，
// 标记该账号仍需补全邮箱（第三方登录未返回邮箱场景），前端据此引导补全后才放行。
func (h *TokenHelper) GenerateTokenEx(userID, tenantID string, roles []string, emailPending bool) (string, error) {
	claims := CustomClaims{
		UserID:       userID,
		TenantID:     tenantID,
		Roles:        roles,
		EmailPending: emailPending,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    h.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.secret)
}

// ParseToken 解析 Token
func (h *TokenHelper) ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		return h.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
