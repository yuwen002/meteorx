package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FieldErrors 把 ValidateStruct 返回的校验错误展开为 字段名 → 中文消息。
// 批量场景（导入 Excel/CSV）需要按字段回传每一行的具体问题，
// 而 ValidateAndResponse 只会把错误写进 HTTP 响应、拼成一条整体提示。
//
// 键优先取 json 标签名（如 username），未声明标签时退回结构体字段名；
// 前端据此可以直接把错误定位到表格的具体列。
func FieldErrors(s interface{}, err error) map[string]string {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(validationErrors))
	for _, e := range validationErrors {
		key := jsonFieldName(s, e.StructField())
		out[key] = getFieldErrorMessage(s, e)
	}
	return out
}

// jsonFieldName 取结构体字段的 json 标签名，没有标签时返回原字段名。
func jsonFieldName(s interface{}, structField string) string {
	t := reflect.TypeOf(s)
	if t == nil {
		return structField
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return structField
	}
	field, found := t.FieldByName(structField)
	if !found {
		return structField
	}
	name := strings.Split(field.Tag.Get("json"), ",")[0]
	if name == "" || name == "-" {
		return structField
	}
	return name
}
