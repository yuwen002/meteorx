package crypto

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testKey(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i)
	}
	return base64.StdEncoding.EncodeToString(raw)
}

func TestNewSecretCipher_Empty(t *testing.T) {
	c, err := NewSecretCipher("")
	require.NoError(t, err)
	assert.Nil(t, c)
	assert.False(t, c.Enabled())
}

func TestNewSecretCipher_Invalid(t *testing.T) {
	_, err := NewSecretCipher("not-base64!!!")
	assert.ErrorIs(t, err, ErrCipherInvalidKey)

	short := base64.StdEncoding.EncodeToString(make([]byte, 16))
	_, err = NewSecretCipher(short)
	assert.ErrorIs(t, err, ErrCipherInvalidKey)
}

func TestSecretCipher_RoundTrip(t *testing.T) {
	c, err := NewSecretCipher(testKey(t))
	require.NoError(t, err)
	require.True(t, c.Enabled())

	secret := "GOCSPX-super-secret-value-123456"
	enc, err := c.Encrypt(secret)
	require.NoError(t, err)
	assert.NotEqual(t, secret, enc)
	assert.Contains(t, enc, CipherPrefix)

	dec, err := c.Decrypt(enc)
	require.NoError(t, err)
	assert.Equal(t, secret, dec)
}

func TestSecretCipher_EmptyPlaintext(t *testing.T) {
	c, _ := NewSecretCipher(testKey(t))
	enc, err := c.Encrypt("")
	require.NoError(t, err)
	assert.Equal(t, "", enc)

	dec, err := c.Decrypt("")
	require.NoError(t, err)
	assert.Equal(t, "", dec)
}

func TestSecretCipher_NonceUniqueness(t *testing.T) {
	c, _ := NewSecretCipher(testKey(t))
	a, _ := c.Encrypt("same-secret")
	b, _ := c.Encrypt("same-secret")
	assert.NotEqual(t, a, b, "same plaintext must yield different ciphertexts (random nonce)")
}

func TestSecretCipher_WrongKey(t *testing.T) {
	c1, _ := NewSecretCipher(testKey(t))
	enc, err := c1.Encrypt("hello")
	require.NoError(t, err)

	other := make([]byte, 32)
	for i := range other {
		other[i] = byte(i + 1)
	}
	c2, _ := NewSecretCipher(base64.StdEncoding.EncodeToString(other))
	_, err = c2.Decrypt(enc)
	assert.ErrorIs(t, err, ErrCipherDecryptFailed)
}

func TestSecretCipher_LegacyPlaintext(t *testing.T) {
	c, _ := NewSecretCipher(testKey(t))
	dec, err := c.Decrypt("legacy-plaintext-secret")
	require.NoError(t, err)
	assert.Equal(t, "legacy-plaintext-secret", dec)
}

func TestSecretCipher_NotConfigured(t *testing.T) {
	var c *SecretCipher
	_, err := c.Encrypt("x")
	assert.ErrorIs(t, err, ErrCipherNotConfigured)

	_, err = c.Decrypt(CipherPrefix + "AAAA")
	assert.ErrorIs(t, err, ErrCipherNotConfigured)
}

func TestMaskSecret(t *testing.T) {
	assert.Equal(t, "", MaskSecret(""))
	assert.Equal(t, "****", MaskSecret("abc"))
	assert.Equal(t, "********3456", MaskSecret("GOCSPX123456"))
	assert.Equal(t, "**cdef", MaskSecret("abcdef"))
}
