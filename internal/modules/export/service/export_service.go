// Package service 提供数据导出模块的业务逻辑。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
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
	// ErrInvalidFilters 筛选条件不合法（如 ids 不是字符串数组、为空或超出条数上限），对应 HTTP 400
	ErrInvalidFilters = fmt.Errorf("筛选条件无效：ids 需为非空字符串数组且不超过 %d 条", MaxFilterIDs)
	// ErrExportFileInUse 导出文件无法删除（正被占用或权限不足），此时保留记录供重试，对应 HTTP 409
	ErrExportFileInUse = errors.New("导出文件正被占用，请稍后重试")
)

// MaxActiveJobsPerTenant 单租户未完结（pending/running）导出任务数的默认上限，
// 可通过配置 export.max_active_jobs 覆盖（见 Config.MaxActiveJobs）；
// 达到上限后新建任务返回 ErrExportQueueFull（429），防止队列被单租户占满。
const MaxActiveJobsPerTenant = 10

// MaxFilterIDs 单次「导出所选」允许携带的 ID 条数上限。
// 派生任务会把它拼成 IN 条件，过大的集合既无前端使用场景，也会拖慢查询。
const MaxFilterIDs = 200

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
	// 导出所选等场景会带 ids 过滤：先规范化并丢弃脏数据，避免任务已创建却注定失败
	if err := normalizeIDFilter(req.Filters); err != nil {
		return nil, err
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

// List 分页列出当前用户的导出任务。
// 系统管理员（isMaster）不以租户限定范围：其为其它租户创建的任务归属目标租户，
// 若仍按当前租户（SYSTEM_ROOT）过滤，下载中心将看不到这些任务。
func (s *ExportService) List(ctx context.Context, tenantID, userID string, req dto.ListExportReq, isMaster bool) ([]*model.ExportJob, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}
	if isMaster {
		tenantID = "" // 空租户ID 表示不限租户，见 repository.ListByUser
	}
	return s.repo.ListByUser(ctx, tenantID, userID, req.Page, req.PageSize, req.Status, req.Resource)
}

// Delete 删除任务及其导出文件。
// 顺序必须是“先删文件、再删记录”：文件删不掉时报错并保留记录，由调用方重试；
// 若先删记录，这个文件就再也无人引用（过期清理只扫数据库行），会永久占用磁盘。
// 文件本就不存在视为成功，避免记录删不掉。
func (s *ExportService) Delete(ctx context.Context, id, tenantID, userID string, isMaster bool) (*model.ExportJob, error) {
	job, err := s.GetByID(ctx, id, tenantID, userID, isMaster)
	if err != nil {
		return nil, err
	}
	if job.FilePath != "" {
		if err := os.Remove(job.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			logger.Warnf("[Export] 删除任务 %s 的文件 %s 失败，保留记录待重试: %v", id, job.FilePath, err)
			return nil, ErrExportFileInUse
		}
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return nil, err
	}
	return job, nil
}

// Config 返回服务配置（供 handler 层构造文件路径等）
func (s *ExportService) Config() Config { return s.config }

// CleanupExpired 删除已过期（expires_at < now）的导出任务及其导出文件，返回成功清理的记录数。
// limit 为单轮扫描上限。文件确实删不掉时（被下载句柄占用、权限不足）保留记录等下一轮重试，
// 以免留下无任何记录引用、后续扫不到的孤儿文件；文件已不存在（ErrNotExist）则照常删记录。
func (s *ExportService) CleanupExpired(ctx context.Context, now time.Time, limit int) (int64, error) {
	jobs, err := s.repo.ListExpired(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	var n int64
	for _, j := range jobs {
		if j.FilePath != "" {
			if err := os.Remove(j.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
				// 先留记录：过期清理只扫数据库行，记录一删，这个文件就永久没人管了。
				logger.Warnf("[Export] 删除过期文件 %s 失败，保留任务 %s 待下轮重试: %v", j.FilePath, j.ID, err)
				continue
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

// normalizeIDFilter 规范化 filters.ids：剔空、去重、限制条数，并回写为 []string。
// 未携带 ids 时直接返回；携带但类型不符、剔空后为空或超上限时返回 ErrInvalidFilters，
// 让调用方在创建任务阶段就得到明确反馈，而不是任务跑起来后才发现条件无效。
func normalizeIDFilter(filters map[string]any) error {
	if filters == nil {
		return nil
	}
	raw, ok := filters["ids"]
	if !ok {
		return nil
	}
	var ids []string
	switch v := raw.(type) {
	case []string:
		ids = v
	case []any:
		for _, item := range v {
			s, isStr := item.(string)
			if !isStr {
				return ErrInvalidFilters
			}
			ids = append(ids, s)
		}
	default:
		return ErrInvalidFilters
	}
	seen := make(map[string]struct{}, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 || len(unique) > MaxFilterIDs {
		return ErrInvalidFilters
	}
	filters["ids"] = unique
	return nil
}
