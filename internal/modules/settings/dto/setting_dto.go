// Package dto 平台运行时配置的传输对象定义。
package dto

// SettingView 配置项视图（管理接口返回，含元数据与来源）
type SettingView struct {
	Key          string `json:"key"`
	Value        string `json:"value"`
	ValueType    string `json:"value_type"`
	DefaultValue string `json:"default_value"`
	IsPublic     bool   `json:"is_public"`
	Source       string `json:"source"` // db=已落库覆盖值 / default=注册表内置默认
	Description  string `json:"description"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// SettingUpdateItem 单个配置更新项，Value 为字符串形态（bool 用 "true"/"false"，int 用十进制）
type SettingUpdateItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// UpdateSettingsReq 批量更新配置请求
type UpdateSettingsReq struct {
	Settings []SettingUpdateItem `json:"settings" validate:"required,min=1"`
}
