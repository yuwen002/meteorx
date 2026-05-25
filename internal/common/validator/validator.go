package validator

import (
	"encoding/json"
	"meteorx/internal/common/response"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// 注册自定义验证规则
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := fld.Tag.Get("json")
		if name == "-" {
			return ""
		}
		return name
	})
}

// ValidateStruct 验证结构体
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// ValidateAndResponse 验证结构体，如果验证失败则返回错误响应
func ValidateAndResponse(w http.ResponseWriter, s interface{}) bool {
	if err := ValidateStruct(s); err != nil {
		// 转换验证错误为更友好的消息
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			errors := make(map[string]string)
			for _, e := range validationErrors {
				errors[e.Field()] = getFieldErrorMessage(s, e)
			}
			response.JSON(w, http.StatusBadRequest, 400, "请求参数验证失败", errors)
		} else {
			response.Fail(w, 400, "请求参数验证失败")
		}
		return false
	}
	return true
}

// getFieldErrorMessage 获取字段级别的错误消息
func getFieldErrorMessage(s interface{}, e validator.FieldError) string {
	fieldName := getFieldLabel(s, e.Field())
	errorMsg := getValidationErrorMessage(e)
	return fieldName + errorMsg
}

// getFieldLabel 获取字段的中文标签
func getFieldLabel(s interface{}, fieldName string) string {
	t := reflect.TypeOf(s)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	if t.Kind() != reflect.Struct {
		return fieldName
	}
	
	// 首先尝试直接查找字段名（处理大小写）
	field, found := t.FieldByName(fieldName)
	if !found {
		// 如果直接查找失败，尝试将首字母大写后查找
		if len(fieldName) > 0 {
			capitalized := strings.ToUpper(fieldName[:1]) + fieldName[1:]
			field, found = t.FieldByName(capitalized)
		}
		if !found {
			// 如果还找不到，遍历所有字段查找匹配的 JSON 标签
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				jsonTag := f.Tag.Get("json")
				if jsonTag == fieldName {
					field = f
					found = true
					break
				}
			}
		}
	}
	
	if !found {
		return fieldName
	}
	
	label := field.Tag.Get("label")
	if label == "" {
		return fieldName
	}
	
	return label
}

// getValidationErrorMessage 获取验证规则的错误消息
func getValidationErrorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "字段不能为空"
	case "min":
		return "长度不能少于" + e.Param() + "个字符"
	case "max":
		return "长度不能超过" + e.Param() + "个字符"
	case "email":
		return "格式不正确"
	case "url":
		return "格式不正确"
	case "alphanum":
		return "只能包含字母和数字"
	case "oneof":
		return "值必须是" + e.Param() + "中的一个"
	default:
		return "格式不正确"
	}
}

// getErrorMessage 将验证错误转换为中文消息（保留原函数以兼容）
func getErrorMessage(e validator.FieldError) string {
	return getValidationErrorMessage(e)
}

// ValidateJSON 验证JSON请求体
func ValidateJSON(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		response.BadRequest(w, "无效的JSON格式")
		return false
	}

	return ValidateAndResponse(w, target)
}
