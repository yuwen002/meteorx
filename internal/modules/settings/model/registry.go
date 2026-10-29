package model

// 配置键常量：全局唯一来源，新增旋钮必须先在此登记并同步实装强制点，
// 避免产生「可编辑却无效」的死配置。
const (
	KeyRegistrationEnabled = "registration_enabled" // 是否开放用户自助注册
	KeyMaintenanceMode     = "maintenance_mode"     // 站点维护模式开关
	KeyMaintenanceMessage  = "maintenance_message"  // 维护模式提示文案
)

// 配置值类型常量
const (
	ValueTypeBool     = "bool"     // 布尔值，字符串形态 "true"/"false"
	ValueTypeInt      = "int"      // 整数值
	ValueTypeString   = "string"   // 字符串值
	ValueTypeDuration = "duration" // 时长值，Go duration 字符串形态（如 "10m"）
)

// SettingDef 配置键注册项：键、内置默认值（DB 无行时的三层回退之一）、类型与公开性
type SettingDef struct {
	Key          string
	DefaultValue string
	ValueType    string
	IsPublic     bool
	Description  string
}

// registry 平台配置键注册表。
// 白名单模式：仅注册过的键允许读写，未注册键的更新请求直接拒绝；
// IsPublic 键会随 GET /api/v1/settings/public 暴露给未登录前端（登录/注册页消费）。
var registry = []SettingDef{
	{
		Key:          KeyRegistrationEnabled,
		DefaultValue: "true",
		ValueType:    ValueTypeBool,
		IsPublic:     true,
		Description:  "是否开放用户自助注册（关闭后注册接口返回 403，前端隐藏注册入口）",
	},
	{
		Key:          KeyMaintenanceMode,
		DefaultValue: "false",
		ValueType:    ValueTypeBool,
		IsPublic:     true,
		Description:  "站点维护模式：开启后已登录用户的业务接口返回 503（平台超管不受限）",
	},
	{
		Key:          KeyMaintenanceMessage,
		DefaultValue: "系统维护中，请稍后再试",
		ValueType:    ValueTypeString,
		IsPublic:     true,
		Description:  "维护模式提示文案，供前端全局展示",
	},
}

// Registry 返回全部已登记的配置键（按登记顺序）。
func Registry() []SettingDef {
	return registry
}

// Lookup 按键查找注册项，未登记时返回 false
func Lookup(key string) (SettingDef, bool) {
	for _, d := range registry {
		if d.Key == key {
			return d, true
		}
	}
	return SettingDef{}, false
}
