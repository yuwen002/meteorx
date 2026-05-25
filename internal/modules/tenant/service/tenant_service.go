package service

import (
	"context"
	"errors"
	"meteorx/internal/modules/tenant/repository"
	"time"

	"meteorx/internal/modules/tenant/model"
	"meteorx/pkg/uuid"
)

// 定义业务层特有的错误，方便 Handler 层判断
var (
	ErrTenantNotFound = errors.New("tenant not found")
	ErrDomainConflict = errors.New("domain already exists")
)

type TenantService struct {
	repo repository.TenantRepository
}

func NewTenantService(repo repository.TenantRepository) *TenantService {
	return &TenantService{repo: repo}
}

// RegisterTenant 改造点：增加时间处理和业务校验
func (s *TenantService) RegisterTenant(ctx context.Context, name, domain string) (*model.Tenant, error) {
	// 1. 【新增】前置校验：域名唯一性检查
	// 虽然数据库有唯一索引，但在 Service 层做显式检查可以返回更友好的业务错误
	exists, err := s.repo.GetByDomain(ctx, domain)
	if err == nil && exists != nil {
		return nil, ErrDomainConflict
	}

	// 2. 构造模型：由 Service 统一控制初始状态和时间点
	now := time.Now()
	tenant := &model.Tenant{
		ID:        uuid.Generate(),
		Name:      name,
		Domain:    domain,
		Status:    1, // 默认启用状态
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 3. 执行持久化
	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}
