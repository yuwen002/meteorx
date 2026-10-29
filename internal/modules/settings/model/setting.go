// Package model 定义平台运行时配置中心（settings 模块）的领域模型与配置键注册表。
package model

import "time"

// Setting 系统配置领域模型（纯净领域模型，不含第三方框架 Tag）。
// Value 以字符串承载，读写两侧统一按 ValueType 做类型化转换。
type Setting struct {
	Key       string
	Value     string
	ValueType string
	IsPublic  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
