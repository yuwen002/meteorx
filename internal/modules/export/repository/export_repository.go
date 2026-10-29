// Package repository 提供数据导出任务的数据访问层。
package repository

import (
	"context"
	"errors"
	"time"

	"meteorx/internal/modules/export/model"

	"gorm.io/gorm"
)

// ExportJobPO 导出任务的 GORM 持久化模型
type ExportJobPO struct {
	ID          string     `gorm:"primaryKey;size:26;comment:'任务ID'"`
	TenantID    string     `gorm:"index;size:26;comment:'租户ID'"`
	UserID      string     `gorm:"index;size:26;comment:'创建用户ID'"`
	Resource    string     `gorm:"size:32;index;comment:'资源类型：user/task/audit_log/wiki_document'"`
	Format      string     `gorm:"size:8;comment:'导出格式：csv/xlsx/json/pdf，空为csv'"`
	FiltersJSON string     `gorm:"type:text;comment:'筛选条件 JSON'"`
	Status      string     `gorm:"size:16;index;comment:'状态：pending/running/completed/failed'"`
	FilePath    string     `gorm:"size:512;comment:'生成的 CSV 文件路径'"`
	Rows        int64      `gorm:"comment:'导出总行数'"`
	ErrorMsg    string     `gorm:"size:1000;comment:'失败原因'"`
	CreatedAt   time.Time  `gorm:"autoCreateTime;index;comment:'创建时间'"`
	StartedAt   *time.Time `gorm:"comment:'开始执行时间'"`
	CompletedAt *time.Time `gorm:"comment:'完成时间'"`
	ExpiresAt   time.Time  `gorm:"index;comment:'文件过期时间'"`
	// LeaseExpiresAt 执行租约到期时间，worker 心跳续期；索引使回收扫描不走全表
	LeaseExpiresAt *time.Time `gorm:"index;comment:'执行租约到期时间，过期表示持有进程已死'"`
	LeaseToken     string     `gorm:"size:36;comment:'本次执行的归属令牌，防止被回收后的旧 worker 覆写'"`
	Attempts       int        `gorm:"comment:'已执行次数'"`
	DeletedAt      gorm.DeletedAt
}

// TableName 表名
func (ExportJobPO) TableName() string { return "export_jobs" }

func (po ExportJobPO) toDomain() *model.ExportJob {
	return &model.ExportJob{
		ID:             po.ID,
		TenantID:       po.TenantID,
		UserID:         po.UserID,
		Resource:       po.Resource,
		Format:         po.Format,
		FiltersJSON:    po.FiltersJSON,
		Status:         po.Status,
		FilePath:       po.FilePath,
		Rows:           po.Rows,
		ErrorMsg:       po.ErrorMsg,
		CreatedAt:      po.CreatedAt,
		StartedAt:      po.StartedAt,
		CompletedAt:    po.CompletedAt,
		ExpiresAt:      po.ExpiresAt,
		LeaseExpiresAt: po.LeaseExpiresAt,
		LeaseToken:     po.LeaseToken,
		Attempts:       po.Attempts,
	}
}

func fromDomain(m *model.ExportJob) *ExportJobPO {
	return &ExportJobPO{
		ID:             m.ID,
		TenantID:       m.TenantID,
		UserID:         m.UserID,
		Resource:       m.Resource,
		Format:         m.Format,
		FiltersJSON:    m.FiltersJSON,
		Status:         m.Status,
		FilePath:       m.FilePath,
		Rows:           m.Rows,
		ErrorMsg:       m.ErrorMsg,
		CreatedAt:      m.CreatedAt,
		StartedAt:      m.StartedAt,
		CompletedAt:    m.CompletedAt,
		ExpiresAt:      m.ExpiresAt,
		LeaseExpiresAt: m.LeaseExpiresAt,
		LeaseToken:     m.LeaseToken,
		Attempts:       m.Attempts,
	}
}

// ExportRepository 导出任务仓储接口
type ExportRepository interface {
	Create(ctx context.Context, job *model.ExportJob) error
	GetByID(ctx context.Context, id string) (*model.ExportJob, error)
	// ListByUser 按创建人列出任务；tenantID 为空表示不限租户（仅系统管理员跨租户查看时使用）
	ListByUser(ctx context.Context, tenantID, userID string, page, pageSize int, status, resource string) ([]*model.ExportJob, int64, error)
	// ListPending 拉取待执行任务（worker 消费），按创建时间升序
	ListPending(ctx context.Context, limit int) ([]*model.ExportJob, error)
	// MarkRunning 将任务置为 running 并写入本次执行凭据（开始时间、租约、归属令牌），
	// 同时累加执行次数；返回 true 表示本次调用成功从 pending 抢占。
	MarkRunning(ctx context.Context, id string, claim Claim) (bool, error)
	// Heartbeat 续租执行中的任务；仅当任务仍为 running 且令牌匹配时生效，返回 true 表示续租成功。
	Heartbeat(ctx context.Context, id, token string, leaseExpiresAt time.Time) (bool, error)
	// ReclaimExpiredLeases 回收租约过期的 running 任务（执行进程已被强杀/OOM，无任何收尾）：
	// 未达 maxAttempts 者回队 pending（并清空令牌），达上限者置 failed。leaseTTL 用于兼容无租约列的历史行（按 started_at 判定）。
	// 多副本安全：持有效租约的任务属其他存活 worker，绝不会被抢。
	ReclaimExpiredLeases(ctx context.Context, now time.Time, leaseTTL time.Duration, maxAttempts int) (requeued, aborted int64, err error)
	// MarkCompleted 更新为 completed，附带文件路径与行数；仅当前令牌持有者可写。
	MarkCompleted(ctx context.Context, id, token, filePath string, rows int64, completedAt time.Time) error
	// MarkFailed 更新为 failed 并记录错误消息；仅当前令牌持有者可写。
	MarkFailed(ctx context.Context, id, token, errMsg string, completedAt time.Time) error
	// ListExpired 查询已过期待清理的任务
	ListExpired(ctx context.Context, now time.Time, limit int) ([]*model.ExportJob, error)
	// CountActiveByTenant 统计指定租户未完结（pending/running）的任务数，用于队列上限准入检查
	CountActiveByTenant(ctx context.Context, tenantID string) (int64, error)
	Delete(ctx context.Context, id string) error
}

type exportRepository struct {
	db *gorm.DB
}

// Claim 一次抢占执行的凭据。Token 标识“哪一次执行”在持有该任务：
// 若仅按 id 回写，当一个任务因心跳断供（进程暂停、DB 超时）被回收并重驱后，
// 旧执行完成时的迟到回写会覆写新执行的租约与终态。带上令牌后，这些回写自然空转。
type Claim struct {
	StartedAt      time.Time
	LeaseExpiresAt time.Time
	Token          string
}

// NewExportRepository 创建导出任务仓储
func NewExportRepository(db *gorm.DB) ExportRepository {
	return &exportRepository{db: db}
}

func (r *exportRepository) Create(ctx context.Context, job *model.ExportJob) error {
	return r.db.WithContext(ctx).Create(fromDomain(job)).Error
}

func (r *exportRepository) GetByID(ctx context.Context, id string) (*model.ExportJob, error) {
	var po ExportJobPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExportJobNotFound
		}
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *exportRepository) ListByUser(ctx context.Context, tenantID, userID string, page, pageSize int, status, resource string) ([]*model.ExportJob, int64, error) {
	var pos []*ExportJobPO
	var total int64

	query := r.db.WithContext(ctx).Model(&ExportJobPO{}).Where("user_id = ?", userID)
	// tenantID 为空表示不限租户：系统管理员需要看到自己为其它租户创建的任务
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*model.ExportJob, len(pos))
	for i, po := range pos {
		items[i] = po.toDomain()
	}
	return items, total, nil
}

func (r *exportRepository) ListPending(ctx context.Context, limit int) ([]*model.ExportJob, error) {
	var pos []*ExportJobPO
	if err := r.db.WithContext(ctx).
		Where("status = ?", model.ExportStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&pos).Error; err != nil {
		return nil, err
	}
	items := make([]*model.ExportJob, len(pos))
	for i, po := range pos {
		items[i] = po.toDomain()
	}
	return items, nil
}

func (r *exportRepository) MarkRunning(ctx context.Context, id string, claim Claim) (bool, error) {
	res := r.db.WithContext(ctx).Model(&ExportJobPO{}).
		Where("id = ? AND status = ?", id, model.ExportStatusPending).
		Updates(map[string]any{
			"status":           model.ExportStatusRunning,
			"started_at":       claim.StartedAt,
			"lease_expires_at": claim.LeaseExpiresAt,
			"lease_token":      claim.Token,
			"attempts":         gorm.Expr("attempts + 1"),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// Heartbeat 仅在任务仍为 running 且令牌匹配时续租。令牌不匹配说明该任务已被回收并重驱，
// 本执行已成僵尸：若仍按 id 续租，会把新持有者的租约拉到未来，使其逃离回收。
func (r *exportRepository) Heartbeat(ctx context.Context, id, token string, leaseExpiresAt time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Model(&ExportJobPO{}).
		Where("id = ? AND status = ? AND lease_token = ?", id, model.ExportStatusRunning, token).
		Update("lease_expires_at", leaseExpiresAt)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// ErrMsgInterruptExhausted 执行次数超限后回写的失败原因（可达上限说明该任务会反复撞死进程）。
const ErrMsgInterruptExhausted = "导出多次被中断（服务崩溃或超时），已停止重试，请重新发起"

// staleLeaseWhere 租约过期判定：
//  1. 本次变更后写入的租约已过期；
//  2. 升级前遗留的 running 行没有租约列（AutoMigrate 补列为 NULL），否则它们永远得不到回收，
//     退而按 started_at 旧于一个 TTL 判定。
func staleLeaseWhere() string {
	return "(lease_expires_at IS NOT NULL AND lease_expires_at < ?)" +
		" OR (lease_expires_at IS NULL AND (started_at IS NULL OR started_at < ?))"
}

func (r *exportRepository) ReclaimExpiredLeases(ctx context.Context, now time.Time, leaseTTL time.Duration, maxAttempts int) (int64, int64, error) {
	stale := func() *gorm.DB {
		return r.db.WithContext(ctx).Model(&ExportJobPO{}).
			Where("status = ?", model.ExportStatusRunning).
			Where(staleLeaseWhere(), now, now.Add(-leaseTTL))
	}
	// 先探测再回收：绝大多数轮次并无僵死行，空转 UPDATE 除产生写负担外，
	// 在 SQL 日志（db.debug）开启时每 2s 刷两条噪声，探测命中 0 即直接返回。
	var staleCount int64
	if err := stale().Count(&staleCount).Error; err != nil {
		return 0, 0, err
	}
	if staleCount == 0 {
		return 0, 0, nil
	}
	// 先处理超限行：不回 pending，避免必然撞死进程的导出形成崩溃循环
	abortedRes := stale().Where("attempts >= ?", maxAttempts).Updates(map[string]any{
		"status":           model.ExportStatusFailed,
		"error_msg":        ErrMsgInterruptExhausted,
		"completed_at":     now,
		"lease_expires_at": nil,
	})
	if abortedRes.Error != nil {
		return 0, 0, abortedRes.Error
	}
	requeuedRes := stale().Where("attempts < ?", maxAttempts).Updates(map[string]any{
		"status":           model.ExportStatusPending,
		"started_at":       nil,
		"lease_expires_at": nil,
		"lease_token":      "",
	})
	if requeuedRes.Error != nil {
		return 0, 0, requeuedRes.Error
	}
	return requeuedRes.RowsAffected, abortedRes.RowsAffected, nil
}

// MarkCompleted 仅当前令牌持有者可写：任务被回收重驱后，旧执行的迟到回写不得抢走新执行的结果。
func (r *exportRepository) MarkCompleted(ctx context.Context, id, token, filePath string, rows int64, completedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&ExportJobPO{}).
		Where("id = ? AND status = ? AND lease_token = ?", id, model.ExportStatusRunning, token).
		Updates(map[string]any{
			"status":           model.ExportStatusCompleted,
			"file_path":        filePath,
			"rows":             rows,
			"completed_at":     completedAt,
			"lease_expires_at": nil,
			"lease_token":      "",
		}).Error
}

// MarkFailed 仅当前令牌持有者可写（理由同 MarkCompleted）：旧 worker 被 ctx 取消后
// 回写的 failed 不能盖掉已重驱新执行的状态。
func (r *exportRepository) MarkFailed(ctx context.Context, id, token, errMsg string, completedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&ExportJobPO{}).
		Where("id = ? AND status = ? AND lease_token = ?", id, model.ExportStatusRunning, token).
		Updates(map[string]any{
			"status":           model.ExportStatusFailed,
			"error_msg":        errMsg,
			"completed_at":     completedAt,
			"lease_expires_at": nil,
			"lease_token":      "",
		}).Error
}

func (r *exportRepository) ListExpired(ctx context.Context, now time.Time, limit int) ([]*model.ExportJob, error) {
	var pos []*ExportJobPO
	if err := r.db.WithContext(ctx).
		Where("expires_at < ?", now).
		Order("expires_at ASC").
		Limit(limit).
		Find(&pos).Error; err != nil {
		return nil, err
	}
	items := make([]*model.ExportJob, len(pos))
	for i, po := range pos {
		items[i] = po.toDomain()
	}
	return items, nil
}

func (r *exportRepository) CountActiveByTenant(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&ExportJobPO{}).
		Where("tenant_id = ? AND status IN ?", tenantID, []string{model.ExportStatusPending, model.ExportStatusRunning}).
		Count(&count).Error
	return count, err
}

func (r *exportRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&ExportJobPO{}, "id = ?", id).Error
}

// ErrExportJobNotFound 导出任务不存在
var ErrExportJobNotFound = errors.New("export job not found")

// Models 返回本模块全部待迁移模型，作为 AutoMigrate 与脏列巡检的单一模型来源。
func Models() []interface{} {
	return []interface{}{&ExportJobPO{}}
}

// AutoMigrate 迁移导出任务表
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(Models()...)
}
