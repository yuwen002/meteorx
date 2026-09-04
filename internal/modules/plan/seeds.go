package plan

import (
	"context"

	"meteorx/internal/modules/plan/dto"
	"meteorx/internal/modules/plan/service"
	"meteorx/pkg/logger"
)

// SeedPlans 初始化默认套餐（幂等：已存在则跳过）
// 基于 plan code 做唯一性判断
func SeedPlans(ctx context.Context, svc *service.PlanService) {
	defaults := []dto.CreatePlanReq{
		{
			Name:        "基础版",
			Code:        "basic",
			Description: "适合初创团队，提供基础的用户管理能力",
			UserLimit:   10,
			Price:       0,
			Status:      1,
		},
		{
			Name:        "标准版",
			Code:        "standard",
			Description: "适合成长型企业，支持更多用户与完整权限体系",
			UserLimit:   100,
			Price:       299,
			Status:      1,
		},
		{
			Name:        "专业版",
			Code:        "professional",
			Description: "适合大型企业，提供全面配额与优先支持",
			UserLimit:   1000,
			Price:       999,
			Status:      1,
		},
	}

	// 复用 service.CreatePlan 保证校验逻辑一致；若 code 已存在则跳过
	for i, p := range defaults {
		if _, err := svc.CreatePlan(ctx, p); err != nil {
			// 忽略 code 冲突（已存在）；记录其他错误
			if err != service.ErrPlanCodeConflict {
				logger.Errorf("[Plan] 初始化套餐 %d 失败: %v", i+1, err)
			}
		}
	}
	logger.Infof("[Plan] 默认套餐初始化完成 (%d)", len(defaults))
}
