// Package totp 实现 RFC 6238 (TOTP) 双因素认证口令的生成与校验，
// 仅依赖标准库（HMAC-SHA1 + Base32），兼容 Google Authenticator / 主流验证器 App。
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
)

const (
	// Digits 口令位数（标准 6 位）。
	Digits = 6
	// Period 时间步长（秒），标准 30s。
	Period = 30
	// SecretBytes 随机密钥字节数，Base32 后约 26 字符（128bit 熵）。
	SecretBytes = 16
	// skew 校验时允许的时间步漂移（前后各 skew 步），容忍客户端与服务端时钟差。
	skew = 1
)

var (
	// ErrInvalidSecret 密钥格式非法（非 Base32）。
	ErrInvalidSecret = errors.New("totp: invalid secret")
	// ErrInvalidCode 口令格式非法（非纯数字或位数不符）。
	ErrInvalidCode = errors.New("totp: invalid code")
)

// GenerateSecret 生成随机 Base32 编码密钥（不含填充 '='），用于绑定验证器 App。
func GenerateSecret() (string, error) {
	b := make([]byte, SecretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("totp: read random: %w", err)
	}
	secret := strings.TrimRight(base32.StdEncoding.EncodeToString(b), "=")
	return secret, nil
}

// OTPAuthURL 构造 otpauth:// URI，供前端生成二维码或引导用户手动录入。
// issuer 为服务/组织名，account 为用户标识（如 email 或 username）。
func OTPAuthURL(secret, issuer, account string) string {
	// otpauth://totp/{issuer}:{account}?secret=..&issuer=..&algorithm=SHA1&digits=6&period=30
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(Digits))
	q.Set("period", fmt.Sprint(Period))
	return fmt.Sprintf("otpauth://totp/%s?%s", label, q.Encode())
}

// Now 生成当前时刻的口令（便于展示/测试）。
func Now(secret string) (string, error) {
	return At(secret, time.Now())
}

// At 生成指定时刻 secret 对应的 6 位口令。
func At(secret string, t time.Time) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}
	counter := uint64(math.Floor(float64(t.Unix()) / float64(Period)))
	return generate(key, counter), nil
}

// Validate 校验口令是否匹配当前时间（允许前后 skew 步漂移）。
// secret 为空视为未启用 2FA，返回 (false, nil) 由调用方决定放行策略。
func Validate(secret, code string) (bool, error) {
	return ValidateAt(secret, code, time.Now())
}

// ValidateAt 在指定时刻校验口令，命中返回 true。
func ValidateAt(secret, code string, t time.Time) (bool, error) {
	if secret == "" {
		return false, nil
	}
	normalized := normalizeCode(code)
	if len(normalized) != Digits || !allDigits(normalized) {
		return false, ErrInvalidCode
	}
	key, err := decodeSecret(secret)
	if err != nil {
		return false, err
	}
	base := uint64(math.Floor(float64(t.Unix()) / float64(Period)))
	for i := -skew; i <= skew; i++ {
		// base 为 uint64，规避下溢：负漂移仅在 base 足够大时参与比较
		var c uint64
		if i < 0 {
			if base < uint64(-i) {
				continue
			}
			c = base - uint64(-i)
		} else {
			c = base + uint64(i)
		}
		if hmac.Equal([]byte(generate(key, c)), []byte(normalized)) {
			return true, nil
		}
	}
	return false, nil
}

// decodeSecret 将 Base32 密钥（忽略大小写、空格与填充）解码为字节。
func decodeSecret(secret string) ([]byte, error) {
	s := strings.ToUpper(strings.TrimSpace(secret))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	// 补齐 Base32 填充至 8 的倍数
	if pad := len(s) % 8; pad != 0 {
		s += strings.Repeat("=", 8-pad)
	}
	key, err := base32.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrInvalidSecret
	}
	return key, nil
}

// generate 按 RFC 4226 动态截断算法，对 counter 计算 Digits 位数字口令。
func generate(key []byte, counter uint64) string {
	buf := []byte{
		byte(counter >> 56), byte(counter >> 48), byte(counter >> 40), byte(counter >> 32),
		byte(counter >> 24), byte(counter >> 16), byte(counter >> 8), byte(counter),
	}
	h := hmac.New(sha1.New, key)
	_, _ = h.Write(buf)
	sum := h.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	binary := (uint32(sum[offset]&0x7f) << 24) |
		(uint32(sum[offset+1]) << 16) |
		(uint32(sum[offset+2]) << 8) |
		uint32(sum[offset+3])

	otp := binary % uint32(math.Pow10(Digits))
	return fmt.Sprintf("%0*d", Digits, otp)
}

// normalizeCode 去除口令中的空格。
func normalizeCode(code string) string {
	return strings.ReplaceAll(strings.TrimSpace(code), " ", "")
}

// allDigits 判断字符串是否全为 ASCII 数字。
func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}
