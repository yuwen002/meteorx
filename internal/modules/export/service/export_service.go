// Package service 提供数据导出模块的业务逻辑。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"meteorx/internal/modules/export/dto"
	"meteorx/internal/modules/export/model"
	"meteorx/internal/modules/export/repository"
	"meteorx/pkg/logger"
	"meteorx/pkg/ulid"
)

// 业务错误
var (
	ErrExportNotFound  = errors.New("导出任务不存在")
	ErrExportForbidden = errors.New("无权访问该导出任务")
	ErrInvalidResource = errors.New("不支持的资源类型")
	ErrExportQueueFull = errors.New("导出队列已满，请稍后再试")
)

// MaxActiveJobsPerTenant 单租户未完结（pending/running）导出任务数的默认上限，
// 可通过配置 export.max_active_jobs 覆盖（见 Config.MaxActiveJobs）；
// 达到上限后新建任务返回 ErrExportQueueFull（429），防止队列被单租户占满。
const MaxActiveJobsPerTenant = 10

// Config 导出服务配置
type Config struct {
	Dir           string // 导出文件目录
	RetentionDays int    // 文件保留天数
	WorkerCount   int    // worker 池大小
	BatchSize     int    // CSV flush 阈值
	MaxActiveJobs int    // 单租户未完结任务上限，≤0 时回落 MaxActiveJobsPerTenant
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
	format := req.Format
	if !model.ValidFormat(format) {
		// 非法的格式（理论上已被 handler 校验拦截）回落默认 CSV，保证任务可正常创建
		format = model.DefaultFormat
	}
	// 队列上限准入：未完结任务达上限时拒绝新建，查询失败时不阻断创建（fail-open）
	limit := s.config.MaxActiveJobs
	if limit <= 0 {
		limit = MaxActiveJobsPerTenant
	}
	active, err := s.repo.CountActiveByTenant(ctx, tenantID)
	if err != nil {
		logger.Warnf("[Export] 统计未完结任务失败，跳过上限检查: tenant=%s err=%v", tenantID, err)
	} else if active >= int64(limit) {
		return nil, ErrExportQueueFull
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
		Format:      format,
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

// CleanupExpired 删除已过期（expires_at < now）的导出任务及其 CSV 文件，返回成功清理的记录数。
// limit 为单轮扫描上限；文件删除失败不阻断记录删除，仅告警。
func (s *ExportService) CleanupExpired(ctx context.Context, now time.Time, limit int) (int64, error) {
	jobs, err := s.repo.ListExpired(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	var n int64
	for _, j := range jobs {
		if j.FilePath != "" {
			if err := os.Remove(j.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
				logger.Warnf("[Export] 删除过期文件 %s 失败: %v", j.FilePath, err)
			}
		}
		if err := s.repo.Delete(ctx, j.ID); err != nil {
			logger.Warnf("[Export] 删除过期任务 %s 失败: %v", j.ID, err)
			continue
		}
		n++
	}
	return n, nil
}

// Repo 暴露仓储给 worker 使用
func (s *ExportService) Repo() repository.ExportRepository { return s.repo }
