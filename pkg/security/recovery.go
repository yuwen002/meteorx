package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// 双因素认证备份恢复码：一次性口令，用于验证器 App 丢失时兜底登录。
const (
	// RecoveryCodeCount 每次开启 / 重新生成 2FA 时下发的备份恢复码数量。
	RecoveryCodeCount = 10
	// recoveryCodeGroups 单个恢复码的分组数（XXXXX-XXXXX → 2 组）。
	recoveryCodeGroups = 2
	// recoveryCodeGroupLen 每组的字符数。
	recoveryCodeGroupLen = 5
)

// recoveryAlphabet 去歧义字符集：排除易混淆的 0/O、1/I/L，便于用户手抄。
const recoveryAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// GenerateRecoveryCodes 生成 n 个去歧义格式的备份恢复码明文（格式 XXXXX-XXXXX）。
// 明文仅在生成时返回一次供用户抄录，调用方须自行哈希后持久化，切勿存储明文。
func GenerateRecoveryCodes(n int) ([]string, error) {
	if n <= 0 {
		return nil, errors.New("security: recovery code count must be positive")
	}
	codes := make([]string, 0, n)
	for i := 0; i < n; i++ {
		c, err := generateOneRecoveryCode()
		if err != nil {
			return nil, err
		}
		codes = append(codes, c)
	}
	return codes, nil
}

// generateOneRecoveryCode 生成单个 XXXXX-XXXXX 形式的恢复码。
func generateOneRecoveryCode() (string, error) {
	total := recoveryCodeGroups * recoveryCodeGroupLen
	buf := make([]byte, total)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.New("security: read random for recovery code")
	}
	groups := make([]string, 0, recoveryCodeGroups)
	idx := 0
	for g := 0; g < recoveryCodeGroups; g++ {
		var sb strings.Builder
		for c := 0; c < recoveryCodeGroupLen; c++ {
			sb.WriteByte(recoveryAlphabet[buf[idx]%byte(len(recoveryAlphabet))])
			idx++
		}
		groups = append(groups, sb.String())
	}
	return strings.Join(groups, "-"), nil
}

// NormalizeRecoveryCode 规范化恢复码：转大写并去除空格与横线，消除录入格式差异，便于比对与哈希。
func NormalizeRecoveryCode(code string) string {
	s := strings.ToUpper(strings.TrimSpace(code))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}

// HashRecoveryCode 计算恢复码哈希（SHA-256，十六进制），用于安全存储。
func HashRecoveryCode(code string) string {
	sum := sha256.Sum256([]byte(NormalizeRecoveryCode(code)))
	return hex.EncodeToString(sum[:])
}

// VerifyRecoveryCode 恒定时间比较恢复码明文与存储哈希，避免时序侧信道。
func VerifyRecoveryCode(code, hash string) bool {
	return hmac.Equal([]byte(HashRecoveryCode(code)), []byte(hash))
}
