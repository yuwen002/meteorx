package storage

import (
	"path/filepath"
	"strings"

	"meteorx/pkg/idgen"
)

// defaultPresignTTL 私有桶下 GetURL 尽力预签名的默认有效期（秒）。
const defaultPresignTTL int64 = 3600

// normalizeTTL 归一化预签名有效期：<=0 时回退默认值。
func normalizeTTL(ttl int64) int64 {
	if ttl <= 0 {
		return defaultPresignTTL
	}
	return ttl
}

// buildObjectKey 依据前缀与原始文件名生成对象键，沿用本地存储「唯一 ID + 原扩展名」的命名约定，
// 使 FilePath 在数据库与访问链路中的语义保持一致（扁平、无目录穿越风险）。
func buildObjectKey(prefix, originalName string) string {
	ext := strings.ToLower(filepath.Ext(originalName))
	name := idgen.New() + ext
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return name
	}
	return prefix + "/" + name
}

// joinBaseURL 拼接公开访问基址与对象键，避免重复斜杠。
func joinBaseURL(base, key string) string {
	return strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(key, "/")
}
