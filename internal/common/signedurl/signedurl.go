// Package signedurl 提供短时效签名 URL 的生成与校验，
// 用于对 /uploads 等静态资源做"可控公开访问"（知道链接不等于能长期访问）。
package signedurl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// URL 查询参数名（与生成方、校验方共用）
const (
	QueryExpires = "e" // 过期时间戳（Unix 秒）
	QuerySig     = "s" // HMAC-SHA256 签名
)

// DefaultTTL 默认签名有效时长
const DefaultTTL = 30 * time.Minute

// Expires 计算过期时间戳（当前时间 + ttl，Unix 秒）
func Expires(ttl time.Duration) int64 {
	return time.Now().Add(ttl).Unix()
}

// Sign 计算 path+expires 的 HMAC-SHA256 签名（hex 编码）
func Sign(path, secret string, expires int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(path))
	mac.Write([]byte{0}) // 分隔符，避免拼接边界歧义
	mac.Write([]byte(strconv.FormatInt(expires, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

// Build 构造带签名参数(e=..&s=..)的完整访问 URL
// baseURL 形如 http://host:8081/uploads；path 为相对文件路径（单段文件名）
func Build(baseURL, path, secret string, expires int64) string {
	base := strings.TrimSuffix(baseURL, "/")
	return base + "/" + path + "?" + QueryExpires + "=" + strconv.FormatInt(expires, 10) +
		"&" + QuerySig + "=" + Sign(path, secret, expires)
}

// Verify 校验签名是否正确（常量时间比较，防时序侧信道）
func Verify(path, secret string, expires int64, signature string) bool {
	if secret == "" || signature == "" || expires <= 0 {
		return false
	}
	expected := Sign(path, secret, expires)
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(signature)))
}
