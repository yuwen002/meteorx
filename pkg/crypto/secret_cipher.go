package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// CipherPrefix 密文前缀，用于区分密文与历史明文（无前缀视为明文，读取时原样返回）。
const CipherPrefix = "enc:v1:"

// ErrCipherNotConfigured 未配置加密主密钥。
var ErrCipherNotConfigured = errors.New("crypto: secret key not configured")

// ErrCipherInvalidKey 主密钥格式非法（非 base64 或长度不为 32 字节）。
var ErrCipherInvalidKey = errors.New("crypto: invalid secret key (expect base64-encoded 32 bytes)")

// ErrCipherDecryptFailed 解密失败（密文损坏或主密钥不匹配）。
var ErrCipherDecryptFailed = errors.New("crypto: decryption failed")

// SecretCipher 基于 AES-256-GCM 的凭据加解密器。
// 主密钥为 base64 编码的 32 字节随机值（openssl rand -base64 32）。
type SecretCipher struct {
	key []byte
}

// NewSecretCipher 从 base64 主密钥构造加解密器。
// 入参为空时返回 (nil, nil)，表示未启用加密（调用方应据此降级并告警）。
func NewSecretCipher(base64Key string) (*SecretCipher, error) {
	trimmed := strings.TrimSpace(base64Key)
	if trimmed == "" {
		return nil, nil
	}
	raw, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCipherInvalidKey, err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("%w: got %d bytes", ErrCipherInvalidKey, len(raw))
	}
	return &SecretCipher{key: raw}, nil
}

// Enabled 是否已配置主密钥（未配置时 Encrypt 返回 ErrCipherNotConfigured）。
func (c *SecretCipher) Enabled() bool {
	return c != nil && len(c.key) == 32
}

// Encrypt 加密明文，返回带 CipherPrefix 前缀的 base64 字符串（nonce 附于密文前）。
func (c *SecretCipher) Encrypt(plaintext string) (string, error) {
	if !c.Enabled() {
		return "", ErrCipherNotConfigured
	}
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: read nonce: %w", err)
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return CipherPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt 解密由 Encrypt 产出的密文。
// 无前缀的历史明文原样返回（便于平滑升级），解密失败返回 ErrCipherDecryptFailed。
func (c *SecretCipher) Decrypt(stored string) (string, error) {
	if !strings.HasPrefix(stored, CipherPrefix) {
		// 历史明文：未启用加密前写入的数据
		return stored, nil
	}
	if !c.Enabled() {
		return "", ErrCipherNotConfigured
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, CipherPrefix))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCipherDecryptFailed, err)
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCipherDecryptFailed, err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCipherDecryptFailed, err)
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("%w: ciphertext too short", ErrCipherDecryptFailed)
	}

	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCipherDecryptFailed, err)
	}
	return string(plaintext), nil
}

// MaskSecret 脱敏展示：仅保留末尾 4 位，其余以 * 代替；过短或为空时返回固定占位。
func MaskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(secret)-4) + secret[len(secret)-4:]
}
