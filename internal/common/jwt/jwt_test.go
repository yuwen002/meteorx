package jwt

import (
	"testing"
	"time"

	"meteorx/internal/config"

	"github.com/stretchr/testify/assert"
)

func newTestHelper(expiration string) *TokenHelper {
	return NewTokenHelper(config.JWTConfig{
		Secret:     "test-secret-key",
		Expiration: expiration,
		Issuer:     "meteorx-test",
	})
}

func TestGenerateAndParse_RoundTrip(t *testing.T) {
	helper := newTestHelper("2h")

	token, err := helper.GenerateToken("user-1", "tenant-1", []string{"admin"})
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := helper.ParseToken(token)
	assert.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "tenant-1", claims.TenantID)
	assert.Equal(t, []string{"admin"}, claims.Roles)
	assert.Equal(t, "meteorx-test", claims.Issuer)
	assert.True(t, claims.ExpiresAt.After(time.Now()))
}

func TestParseToken_Expired(t *testing.T) {
	helper := newTestHelper("10ms")
	token, err := helper.GenerateToken("user-1", "tenant-1", nil)
	assert.NoError(t, err)

	time.Sleep(30 * time.Millisecond)

	_, err = helper.ParseToken(token)
	assert.Error(t, err)
}

func TestParseToken_WrongSecret(t *testing.T) {
	helper := newTestHelper("1h")
	other := NewTokenHelper(config.JWTConfig{Secret: "other-secret", Expiration: "1h", Issuer: "meteorx-test"})

	token, err := helper.GenerateToken("user-1", "tenant-1", nil)
	assert.NoError(t, err)

	_, err = other.ParseToken(token)
	assert.Error(t, err)
}

func TestParseToken_Tampered(t *testing.T) {
	helper := newTestHelper("1h")
	token, err := helper.GenerateToken("user-1", "tenant-1", nil)
	assert.NoError(t, err)

	// 篡改签名段使其与原始 token 不一致
	tampered := token[:len(token)-2] + "xx"
	_, err = helper.ParseToken(tampered)
	assert.Error(t, err)
}

func TestGetExpiration_InvalidFallsBackTo24h(t *testing.T) {
	helper := NewTokenHelper(config.JWTConfig{Secret: "s", Expiration: "not-a-duration"})
	assert.Equal(t, 24*time.Hour, helper.expiration)
}
