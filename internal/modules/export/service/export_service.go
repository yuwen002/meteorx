// Package service 提供数据导出模块的业务逻辑。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"meteorx/internal/modules/export/dto"
	"meteorx/internal/modules/export/model"
	"meteorx/internal/modules/export/repository"
	"meteorx/pkg/ulid"
)

// 业务错误
var (
	ErrExportNotFound    = errors.New("导出任务不存在")
	ErrExportForbidden   = errors.New("无权访问该导出任务")
	ErrInvalidResource   = errors.New("不支持的资源类型")
	ErrExportQueueFull   = errors.New("导出队列已满，请稍后再试")
)

// Config 导出服务配置
type Config struct {
	Dir           string        // 导出文件目录
	RetentionDays int           // 文件保留天数
	WorkerCount   int           // worker 池大小
	BatchSize     int           // CSV flush 阈值
}

// ExportService 导出任务服务
type ExportService struct {
	repo   repository.ExportRepository
	config Config
}

// NewExportService 创建导出服务
func NewExportService(repo repository.ExportRepository, cfg Config) *ExportService {
	return &ExportService{repo: repo, config: cfg}
}

// CreateExport 创建一条导出任务（状态 pending），返回领域模型。
// worker 池由 StartWorker 启动后异步消费。
func (s *ExportService) CreateExport(ctx context.Context, tenantID, userID string, req dto.CreateExportReq) (*model.ExportJob, error) {
	if !model.ValidResourceType(req.Resource) {
		return nil, ErrInvalidResource
	}
	filters := "{}"
	if req.Filters != nil {
		b, err := json.Marshal(req.Filters)
		if err != nil {
			return nil, err
		}
		filters = string(b)
	}
	retention := time.Duration(s.config.RetentionDays) * 24 * time.Hour
	if retention <= 0 {
		retention = 7 * 24 * time.Hour
	}
	now := time.Now()
	job := &model.ExportJob{
		ID:          ulid.Generate(),
		TenantID:    tenantID,
		UserID:      userID,
		Resource:    req.Resource,
		FiltersJSON: filters,
		Status:      model.ExportStatusPending,
		CreatedAt:   now,
		ExpiresAt:   now.Add(retention),
	}
	if err := s.repo.Create(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

// GetByID 按 ID 读取任务，校验归属：非 master 时只能读自己 tenant_id + user_id 的任务。
func (s *ExportService) GetByID(ctx context.Context, id, tenantID, userID string, isMaster bool) (*model.ExportJob, error) {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrExportJobNotFound) {
			return nil, ErrExportNotFound
		}
		return nil, err
	}
	if !isMaster && (job.TenantID != tenantID || job.UserID != userID) {
		return nil, ErrExportForbidden
	}
	return job, nil
}

// List 分页列出当前用户的导出任务
func (s *ExportService) List(ctx context.Context, tenantID, userID string, req dto.ListExportReq) ([]*model.ExportJob, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}
	return s.repo.ListByUser(ctx, tenantID, userID, req.Page, req.PageSize, req.Status, req.Resource)
}

// Delete 删除任务记录（同时删文件由 handler 层调用 RemoveFile 完成）
func (s *ExportService) Delete(ctx context.Context, id, tenantID, userID string, isMaster bool) (*model.ExportJob, error) {
	job, err := s.GetByID(ctx, id, tenantID, userID, isMaster)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return nil, err
	}
	return job, nil
}

// Config 返回服务配置（供 handler 层构造文件路径等）
func (s *ExportService) Config() Config { return s.config }

// Repo 暴露仓储给 worker 使用
func (s *ExportService) Repo() repository.ExportRepository { return s.repo }
