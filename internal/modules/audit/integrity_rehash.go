package audit

import (
	"context"
	"sync"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/service"
)

// integrityAnchorMu 串行化「锚点封口任务」与「密钥轮换 rehash」。
// rehash 会清空并全量重建锚点链，若与周期封口任务并发追加锚点会造成链错乱，故二者共用同一把锁。
var integrityAnchorMu sync.Mutex

// IntegrityCoordinator 完整性用例编排：只读校验直接委托 service；
// rehash 在持锁下完成行重算与锚点重建。
//
// 无需在此显式补记审计：rehash 是挂在受保护分组下的 HTTP POST，已被 AuditMiddleware 自动记录
// （含真实操作者、状态码与响应体里的重算/重建条数），重复埋点反而制造两条审计。
type IntegrityCoordinator struct {
	svc *service.IntegrityService
}

// NewIntegrityCoordinator 创建完整性用例编排器。
func NewIntegrityCoordinator(svc *service.IntegrityService) *IntegrityCoordinator {
	return &IntegrityCoordinator{svc: svc}
}

// Verify 只读校验，直接委托 service（无需持锁）。
func (c *IntegrityCoordinator) Verify(ctx context.Context) (*dto.IntegrityVerifyResp, error) {
	return c.svc.Verify(ctx)
}

// Rehash 密钥轮换迁移：持锁执行 service.Rehash，避免与周期锚点封口并发。
func (c *IntegrityCoordinator) Rehash(ctx context.Context) (*dto.IntegrityRehashResp, error) {
	integrityAnchorMu.Lock()
	defer integrityAnchorMu.Unlock()
	return c.svc.Rehash(ctx)
}
