package service

import (
	"context"
	"fmt"

	"meteorx/internal/modules/plan/repository"
	planService "meteorx/internal/modules/plan/service"
)

// quotaVerifierFunc 适配函数到 QuotaVerifier 接口
type quotaVerifierFunc func(ctx context.Context, tenantID string) (bool, int64, int, error)

func (f quotaVerifierFunc) CheckUserLimit(ctx context.Context, tenantID string) (bool, int64, int, error) {
	return f(ctx, tenantID)
}

var _ repository.QuotaVerifier = quotaVerifierFunc(nil)

// 包装真实业务错误，使 errors.Is 能匹配
var errPlanExpiredMock = fmt.Errorf("mock: %w", planService.ErrPlanExpired)
var errUserLimitExceededMock = fmt.Errorf("mock: %w", planService.ErrUserLimitExceeded)
var errGenericMock = fmt.Errorf("mock generic error")
