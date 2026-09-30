package config

import (
	"fmt"
	"os"
	"strings"
)

// ValidationError 单个配置字段的校验错误。
type ValidationError struct {
	Field   string
	Message string
}

// Error 实现 error 接口，格式化输出字段与错误消息。
func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation error: %s: %s", e.Field, e.Message)
}

// Validator 配置校验器，收集多项错误后统一报出。
type Validator struct {
	errors []ValidationError
}

// NewValidator 创建一个空校验器。
func NewValidator() *Validator {
	return &Validator{}
}

// Require 校验字段值非空，为空时记录错误，返回自身以链式调用。
func (v *Validator) Require(field, value, msg string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.errors = append(v.errors, ValidationError{Field: field, Message: msg})
	}
	return v
}

// RequireEnv 校验指定环境变量已设置，未设置时记录错误。
func (v *Validator) RequireEnv(field, envKey, msg string) *Validator {
	if os.Getenv(envKey) == "" {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("%s (env: %s)", msg, envKey),
		})
	}
	return v
}

// MustValid 若存在校验错误则汇总返回，否则返回 nil。
func (v *Validator) MustValid() error {
	if len(v.errors) > 0 {
		var msgs []string
		for _, e := range v.errors {
			msgs = append(msgs, e.Error())
		}
		return fmt.Errorf("%d validation error(s): %s", len(v.errors), strings.Join(msgs, "; "))
	}
	return nil
}

// HasErrors 返回是否已收集到校验错误。
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}
