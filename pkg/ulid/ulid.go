package ulid

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// Generate 生成一个新的 ULID（Universally Unique Lexicographically Sortable Identifier）
// ULID 是一种128位的唯一标识符，具有以下特点：
// 1. 26个字符的字符串表示（比UUID短）
// 2. 按字典序排序（基于时间戳）
// 3. 高并发安全
// 4. 适合作为数据库主键
// 返回值: 返回一个26字符的ULID字符串
func Generate() string {
	// 使用当前时间作为时间戳
	timestamp := ulid.Timestamp(time.Now())

	// 使用加密安全的随机数生成器
	entropy := rand.Reader

	// 生成ULID
	id := ulid.MustNew(timestamp, entropy)

	// 返回字符串形式
	return id.String()
}
