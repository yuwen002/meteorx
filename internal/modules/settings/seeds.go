package settings

import (
	"context"

	"meteorx/internal/modules/settings/service"
	"meteorx/pkg/logger"
)

// SeedSettings 初始化默认配置（幂等：仅插入注册表中缺失的键，不覆盖已修改值）
// 与 SeedPlans 同款范式，基于配置键做唯一性判断
func SeedSettings(ctx context.Context, svc *service.SettingService) {
	created, err := svc.SeedDefaults(ctx)
	if err != nil {
		logger.Errorf("[Settings] 初始化默认配置失败: %v", err)
		return
	}
	logger.Infof("[Settings] 默认配置初始化完成 (新建 %d 项)", created)
}
