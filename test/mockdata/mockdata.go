// Package mockdata 提供测试用的 Mock API 响应数据
// 用于 API 集成测试和前端开发联调
package mockdata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// load 加载 mock JSON 文件并解析到 target
func load(filename string, target interface{}) error {
	_, thisFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(thisFile)

	data, err := os.ReadFile(filepath.Join(baseDir, "api", filename))
	if err != nil {
		return fmt.Errorf("读取 mock 文件 %s 失败: %w", filename, err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("解析 mock 文件 %s 失败: %w", filename, err)
	}
	return nil
}

// GetTenant 获取单个租户 mock 数据
func GetTenant() (map[string]interface{}, error) {
	var result map[string]interface{}
	err := load("tenant_get.json", &result)
	return result, err
}

// GetTenantList 获取租户列表 mock 数据
func GetTenantList() (map[string]interface{}, error) {
	var result map[string]interface{}
	err := load("tenant_list.json", &result)
	return result, err
}

// GetUserList 获取用户列表 mock 数据
func GetUserList() (map[string]interface{}, error) {
	var result map[string]interface{}
	err := load("user_list.json", &result)
	return result, err
}

// GetAuditDashboard 获取审计仪表盘 mock 数据
func GetAuditDashboard() (map[string]interface{}, error) {
	var result map[string]interface{}
	err := load("audit_dashboard.json", &result)
	return result, err
}

// LoadMock 通用加载方法，filename 是 api 目录下的 JSON 文件名
func LoadMock(filename string, target interface{}) error {
	return load(filename, target)
}