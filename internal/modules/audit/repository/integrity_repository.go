package repository

import (
	"context"
	"time"

	"meteorx/internal/modules/audit/model"

	"gorm.io/gorm"
)

// AuditIntegrityAnchorPO 审计完整性锚点（append-only 表 audit_integrity_anchors）。
// 每个窗口折叠该窗口内全部行摘要为 WindowDigest，并以 PrevAnchorHash→AnchorHash 链向上一条锚点，
// 形成不可抵赖的时间线：改动/删除/插入/重序任一行都会破坏所在窗口的累加摘要，
// 断开链则破坏后续所有锚点的 AnchorHash。
type AuditIntegrityAnchorPO struct {
	ID             string    `gorm:"primaryKey;size:26;comment:锚点ID(ULID)"`
	WindowStart    time.Time `gorm:"index;comment:窗口起始(不含前一条end，含本条start)"`
	WindowEnd      time.Time `gorm:"index;comment:窗口结束时间(含)"`
	RowCount       int64     `gorm:"comment:窗口内审计日志行数"`
	WindowDigest   string    `gorm:"size:64;comment:窗口累加摘要(HMAC hex)"`
	AnchorHash     string    `gorm:"size:64;index;comment:锚点哈希(链接上一条)"`
	PrevAnchorHash string    `gorm:"size:64;comment:上一锚点哈希，首条为空"`
	CreatedAt      time.Time `gorm:"autoCreateTime;index;comment:创建时间"`
}

// TableName 返回锚点表名 audit_integrity_anchors。
func (AuditIntegrityAnchorPO) TableName() string {
	return "audit_integrity_anchors"
}

// IntegrityAnchor 锚点领域视图（面向 service / job / handler）。
type IntegrityAnchor struct {
	ID             string
	WindowStart    time.Time
	WindowEnd      time.Time
	RowCount       int64
	WindowDigest   string
	AnchorHash     string
	PrevAnchorHash string
	CreatedAt      time.Time
}

func (po AuditIntegrityAnchorPO) toAnchorDomain() *IntegrityAnchor {
	return &IntegrityAnchor{
		ID:             po.ID,
		WindowStart:    po.WindowStart,
		WindowEnd:      po.WindowEnd,
		RowCount:       po.RowCount,
		WindowDigest:   po.WindowDigest,
		AnchorHash:     po.AnchorHash,
		PrevAnchorHash: po.PrevAnchorHash,
		CreatedAt:      po.CreatedAt,
	}
}

// RecordHashEntry 窗口内一行的摘要快照，按 (created_at, id) 有序返回，供锚点折叠与校验重放。
type RecordHashEntry struct {
	ID         string
	RecordHash string
	CreatedAt  time.Time
}

// IntegrityRepository 审计完整性数据访问接口（锚点表 + 审计日志窗口读取）。
type IntegrityRepository interface {
	// CreateAnchor 追加一条锚点（append-only，永不更新/删除）。
	CreateAnchor(ctx context.Context, anchor *IntegrityAnchor) error

	// ListAnchors 按窗口结束时间升序返回全部锚点（校验重放整条链）。
	ListAnchors(ctx context.Context) ([]*IntegrityAnchor, error)

	// LatestAnchor 返回链尾锚点；无锚点时返回 (nil, nil)。
	LatestAnchor(ctx context.Context) (*IntegrityAnchor, error)

	// ListRecordHashesInRange 返回 [start, end) 内审计日志的 (id, record_hash)，按 (created_at, id) 有序。
	// start 为零值时表示不设下界（用于创世窗口）。
	ListRecordHashesInRange(ctx context.Context, start, end time.Time) ([]RecordHashEntry, error)

	// ListLogsInRange 按 (created_at, id) 有序返回 [start, end) 内审计日志整行（含全部字段与存储的 record_hash），
	// 供完整性校验逐行重算与窗口重放。start/end 任一为零值表示该侧不设界。
	ListLogsInRange(ctx context.Context, start, end time.Time) ([]*model.AuditLog, error)
}

// integrityRepository IntegrityRepository 的 GORM 实现。
type integrityRepository struct {
	db *gorm.DB
}

// NewIntegrityRepository 创建审计完整性仓库。
func NewIntegrityRepository(db *gorm.DB) IntegrityRepository {
	return &integrityRepository{db: db}
}

func (r *integrityRepository) CreateAnchor(ctx context.Context, anchor *IntegrityAnchor) error {
	po := AuditIntegrityAnchorPO{
		ID:             anchor.ID,
		WindowStart:    anchor.WindowStart,
		WindowEnd:      anchor.WindowEnd,
		RowCount:       anchor.RowCount,
		WindowDigest:   anchor.WindowDigest,
		AnchorHash:     anchor.AnchorHash,
		PrevAnchorHash: anchor.PrevAnchorHash,
	}
	return r.db.WithContext(ctx).Create(&po).Error
}

func (r *integrityRepository) ListAnchors(ctx context.Context) ([]*IntegrityAnchor, error) {
	var pos []AuditIntegrityAnchorPO
	if err := r.db.WithContext(ctx).
		Order("window_end ASC").
		Order("id ASC").
		Find(&pos).Error; err != nil {
		return nil, err
	}
	anchors := make([]*IntegrityAnchor, len(pos))
	for i, po := range pos {
		anchors[i] = po.toAnchorDomain()
	}
	return anchors, nil
}

func (r *integrityRepository) LatestAnchor(ctx context.Context) (*IntegrityAnchor, error) {
	var po AuditIntegrityAnchorPO
	err := r.db.WithContext(ctx).
		Order("window_end DESC").
		Order("id DESC").
		First(&po).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return po.toAnchorDomain(), nil
}

func (r *integrityRepository) ListRecordHashesInRange(ctx context.Context, start, end time.Time) ([]RecordHashEntry, error) {
	db := r.db.WithContext(ctx).Model(&AuditLogPO{}).
		Select("id, record_hash, created_at")
	if !start.IsZero() {
		db = db.Where("created_at >= ?", start)
	}
	db = db.Where("created_at < ?", end).
		Order("created_at ASC").
		Order("id ASC")

	var pos []AuditLogPO
	if err := db.Find(&pos).Error; err != nil {
		return nil, err
	}
	entries := make([]RecordHashEntry, len(pos))
	for i, po := range pos {
		entries[i] = RecordHashEntry{ID: po.ID, RecordHash: po.RecordHash, CreatedAt: po.CreatedAt}
	}
	return entries, nil
}

func (r *integrityRepository) ListLogsInRange(ctx context.Context, start, end time.Time) ([]*model.AuditLog, error) {
	db := r.db.WithContext(ctx).Model(&AuditLogPO{})
	if !start.IsZero() {
		db = db.Where("created_at >= ?", start)
	}
	if !end.IsZero() {
		db = db.Where("created_at < ?", end)
	}
	db = db.Order("created_at ASC").Order("id ASC")

	var pos []AuditLogPO
	if err := db.Find(&pos).Error; err != nil {
		return nil, err
	}
	logs := make([]*model.AuditLog, len(pos))
	for i, po := range pos {
		logs[i] = po.toDomain()
	}
	return logs, nil
}
