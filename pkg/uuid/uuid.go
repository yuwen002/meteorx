package uuid

import (
	"github.com/google/uuid"
)

// Generate 函数用于生成一个新的UUID（Universally Unique Identifier，通用唯一标识符）
// UUID是一种128位的唯一标识符，通常用于在计算机系统中需要唯一标识的场景
// 该函数使用了uuid包的New方法生成一个新的UUID，并将其转换为字符串形式返回
// 返回值: 返回一个字符串类型的UUID，确保全局唯一性
func Generate() string {
	return uuid.New().String()
}
