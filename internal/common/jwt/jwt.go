package jwt

import (
	"errors"
	"time"

	"meteorx/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

type TokenHelper struct {
	secret     []byte
	expiration time.Duration
	issuer     string
}

// CustomClaims 自定义载荷
type CustomClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
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
func (h *TokenHelper) GenerateToken(userID, tenantID, role string) (string, error) {
	claims := CustomClaims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
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
