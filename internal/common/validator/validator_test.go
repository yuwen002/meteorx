package validator

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

type sampleReq struct {
	Title  string `json:"title" label:"标题" validate:"required,max=200"`
	Scope  string `json:"scope" validate:"required,oneof=all tenant"`
	Email  string `json:"email" validate:"omitempty,email"`
	Home   string `json:"home" validate:"omitempty,url"`
	User   string `json:"user" validate:"omitempty,username"`
	Status int    `json:"status" validate:"oneof=0 1 2"`
}

func TestValidateStruct_Valid(t *testing.T) {
	req := sampleReq{Title: "ok", Scope: "tenant", Email: "a@b.com", Home: "https://x.com", User: "admin_1", Status: 1}
	assert.NoError(t, ValidateStruct(&req))
}

func TestValidateStruct_InvalidRequiredAndRules(t *testing.T) {
	err := ValidateStruct(&sampleReq{Scope: "all", Status: 9})
	assert.Error(t, err)

	validationErrors, ok := err.(validator.ValidationErrors)
	assert.True(t, ok)
	has := map[string]bool{}
	for _, e := range validationErrors {
		has[e.Tag()] = true
	}
	assert.True(t, has["required"], "标题必填应触发 required")
	assert.True(t, has["oneof"], "非法状态应触发 oneof")
}

func TestValidateStruct_RejectsBadFormat(t *testing.T) {
	err := ValidateStruct(&sampleReq{Title: "t", Scope: "all", Email: "not-an-email", Home: "://bad", User: "含中文"})
	assert.Error(t, err)
}

func TestUsernameCustomRule(t *testing.T) {
	cases := []struct {
		value string
		valid bool
	}{
		{"admin", true},
		{"admin_1", true},
		{"admin-1", true},
		{"含中文", false},
		{"has space", false},
		{"", true}, // omitempty：空串跳过 username 规则
	}
	for _, tc := range cases {
		err := ValidateStruct(&sampleReq{Title: "t", Scope: "all", User: tc.value})
		if tc.valid {
			assert.NoError(t, err, "username=%q 应通过", tc.value)
		} else {
			assert.Error(t, err, "username=%q 应被拒绝", tc.value)
		}
	}
}

func TestGetValidationErrorMessage(t *testing.T) {
	type probe struct {
		Title string `json:"title" validate:"required"`
		Scope string `json:"scope" validate:"oneof=all tenant"`
	}
	validationErrors := ValidateStruct(&probe{Scope: "bad"}).(validator.ValidationErrors)
	messages := map[string]string{}
	for _, e := range validationErrors {
		messages[e.Field()] = getValidationErrorMessage(e)
	}
	assert.Equal(t, "不能为空", messages["title"])
	assert.Equal(t, "值必须是all tenant中的一个", messages["scope"])
}

func TestGetFieldLabel_UsesChineseLabel(t *testing.T) {
	req := sampleReq{Title: "", Scope: "all"} // title 缺失触发 required
	validationErrors := ValidateStruct(&req).(validator.ValidationErrors)
	for _, e := range validationErrors {
		if e.Field() == "title" {
			assert.Equal(t, "标题不能为空", getFieldErrorMessage(&req, e))
		}
	}
}

func TestValidateAndResponse_BadRequestBody(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"title":""}`))

	ok := ValidateJSON(w, r, &sampleReq{})

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "标题不能为空")
}

func TestValidateJSON_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{broken`))

	ok := ValidateJSON(w, r, &sampleReq{})

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "无效的JSON格式")
}
