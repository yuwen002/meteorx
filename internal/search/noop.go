package search

import (
	"context"
	"fmt"

	"meteorx/pkg/logger"
)

// noopEngine 空操作引擎：当未配置搜索引擎时使用，所有操作记录日志但不报错。
// 业务层发现 IsAvailable() == false 时降级为数据库 LIKE 搜索。
type noopEngine struct {
	reason error
}

func newNoopEngine(reason error) *noopEngine {
	e := &noopEngine{reason: reason}
	if reason != nil {
		logger.Warnf("Search engine not available: %v. Falling back to database LIKE search.", reason)
	} else {
		logger.Info("Search engine not configured (provider=none). Using database LIKE search.")
	}
	return e
}

// Index 空实现，不写入任何数据。
func (e *noopEngine) Index(_ context.Context, _ ...Document) error {
	return nil
}

// Delete 空实现，不删除任何数据。
func (e *noopEngine) Delete(_ context.Context, _ ...string) error {
	return nil
}

// Search 引擎不可用，始终返回 ErrSearchEngineUnavailable。
func (e *noopEngine) Search(_ context.Context, q *Query) (*SearchResponse, error) {
	return nil, ErrSearchEngineUnavailable
}

// ClearIndex 空实现。
func (e *noopEngine) ClearIndex(_ context.Context) error {
	return nil
}

// Close 空实现。
func (e *noopEngine) Close() error {
	return nil
}

// IsAvailable 始终返回 false，触发业务层降级为数据库 LIKE 搜索。
func (e *noopEngine) IsAvailable() bool {
	return false
}

// Reason 返回引擎不可用的原因
func (e *noopEngine) Reason() error {
	return e.reason
}

// String 返回引擎的可读描述。
func (e *noopEngine) String() string {
	if e.reason != nil {
		return fmt.Sprintf("NoopEngine(reason=%v)", e.reason)
	}
	return "NoopEngine(not configured)"
}
