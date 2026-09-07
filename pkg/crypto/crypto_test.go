package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("S3cure!pass")
	assert.NoError(t, err)
	assert.NotEqual(t, "S3cure!pass", hash)
	assert.Len(t, hash, 60) // bcrypt cost=10 输出固定 60 字符
}

func TestCheckPassword_Success(t *testing.T) {
	hash, err := HashPassword("S3cure!pass")
	assert.NoError(t, err)
	assert.True(t, CheckPassword("S3cure!pass", hash))
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("S3cure!pass")
	assert.NoError(t, err)
	assert.False(t, CheckPassword("wrong-pass", hash))
}

func TestCheckPassword_SamePlaintextIsNotHash(t *testing.T) {
	// 直接把明文当 hash 使用必须失败（防止误用）
	assert.False(t, CheckPassword("abc12345", "abc12345"))
}

func TestHashPassword_IsRandomizedPerCall(t *testing.T) {
	h1, err := HashPassword("same-password")
	assert.NoError(t, err)
	h2, err := HashPassword("same-password")
	assert.NoError(t, err)
	assert.NotEqual(t, h1, h2) // bcrypt salt 随机
	assert.True(t, CheckPassword("same-password", h1))
	assert.True(t, CheckPassword("same-password", h2))
}
