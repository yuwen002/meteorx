package idgen

import (
	"crypto/rand"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// New 生成一个新的 ULID（Universally Unique Lexicographically Sortable Identifier）
// ULID 是 26 字符的字符串，按时间有序，适合作为数据库主键。
// 业务实体主键统一使用本方法生成。
func New() string {
	timestamp := ulid.Timestamp(time.Now())
	entropy := rand.Reader
	id := ulid.MustNew(timestamp, entropy)
	return id.String()
}

// NewULID 是 New 的语义化别名，明确返回 ULID。
func NewULID() string {
	return New()
}

// NewUUID 生成一个新的 UUID v4 字符串（36 字符，含连字符）。
// 仅用于语义上必须是 UUID 的场景（如 OAuth token、外部系统约定的 UUID 字段），
// 业务主键请优先使用 New()。
func NewUUID() string {
	return uuid.New().String()
}

// Parse 解析一个 ULID 字符串，返回其 26 字符规范字符串形式。
// 若输入不是合法 ULID 则返回错误。用于查询数据库前校验 ID 格式。
func Parse(s string) (string, error) {
	id, err := ulid.Parse(s)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// MustParse 解析 ULID 字符串，对非法输入直接 panic。
// 仅对已知有效的字符串使用。
func MustParse(s string) string {
	return ulid.MustParse(s).String()
}
