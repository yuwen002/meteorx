package repository

import (
	"context"
	"strings"
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

// RecordHashUpdate 一行 record_hash 的重写目标（rehash 用）。
type RecordHashUpdate struct {
	ID         string
	RecordHash string
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

	// ListLogsPage 以 (created_at, id) 复合游标向后至多取 limit 行（按 (created_at, id) 升序），
	// 供大表 rehash 分页流式遍历，避免一次性把全表载入内存。afterCreatedAt 为零值且 afterID 为空时
	// 从最早行开始；limit<=0 回落默认页大小。重写 record_hash 不改变 (created_at, id)，游标迭代稳定。
	ListLogsPage(ctx context.Context, afterCreatedAt time.Time, afterID string, limit int) ([]*model.AuditLog, error)

	// BatchUpdateRecordHash 按 id 批量重写审计日志的 record_hash（rehash 迁移用），分块提交。
	BatchUpdateRecordHash(ctx context.Context, updates []RecordHashUpdate) error

	// ReplaceAnchors 在单个事务内清空全部旧锚点并写入新链（rehash 全量重建的原子收口）。
	// 传空切片等价于仅清空。整体原子：中途失败则旧锚点不会被丢。
	ReplaceAnchors(ctx context.Context, anchors []*IntegrityAnchor) error
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

// rehashPageDefault ListLogsPage 未显式传 limit 时的默认页大小（行数），平衡单页内存与往返次数。
const rehashPageDefault = 1000

// ListLogsPage 以 (created_at, id) 复合游标向后分页读取审计日志整行，按 (created_at, id) 升序。
// 首页（afterCreatedAt 零值且 afterID 为空）不加游标条件，从最早行起；后续页取严格大于游标的行。
func (r *integrityRepository) ListLogsPage(ctx context.Context, afterCreatedAt time.Time, afterID string, limit int) ([]*model.AuditLog, error) {
	if limit <= 0 {
		limit = rehashPageDefault
	}
	db := r.db.WithContext(ctx).Model(&AuditLogPO{})
	if !afterCreatedAt.IsZero() || afterID != "" {
		db = db.Where("created_at > ? OR (created_at = ? AND id > ?)", afterCreatedAt, afterCreatedAt, afterID)
	}
	db = db.Order("created_at ASC").Order("id ASC").Limit(limit)

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

// rehashUpdateChunk 单批 UPDATE 的行数上限，避免一次拼接过长的 CASE WHEN 语句。
const rehashUpdateChunk = 500

// BatchUpdateRecordHash 按 id 批量重写 record_hash：以 CASE WHEN 单语句更新一块 id，分块提交。
// 空入参直接返回 nil（不触库）。仅更新 record_hash，不改动 created_at 等被摘要覆盖的字段。
func (r *integrityRepository) BatchUpdateRecordHash(ctx context.Context, updates []RecordHashUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	for start := 0; start < len(updates); start += rehashUpdateChunk {
		end := start + rehashUpdateChunk
		if end > len(updates) {
			end = len(updates)
		}
		chunk := updates[start:end]
		caseExpr := "CASE id "
		args := make([]interface{}, 0, len(chunk)*3)
		for _, u := range chunk {
			caseExpr += "WHEN ? THEN ? "
			args = append(args, u.ID, u.RecordHash)
		}
		caseExpr += "END"
		// WHERE id IN (?,?,...)：逐 id 一个占位符（raw Exec 不会把切片展开为多值）
		placeholders := make([]string, len(chunk))
		for i, u := range chunk {
			placeholders[i] = "?"
			args = append(args, u.ID)
		}
		sql := "UPDATE audit_logs SET record_hash = " + caseExpr + " WHERE id IN (" + strings.Join(placeholders, ",") + ")"
		if err := r.db.WithContext(ctx).Exec(sql, args...).Error; err != nil {
			return err
		}
	}
	return nil
}

// replaceAnchorChunk 单条 INSERT 语句携带的锚点数上限，避免一次拼接过长。
const replaceAnchorChunk = 500

// ReplaceAnchors 在单事务内先清空旧锚点链、再批量写入新链，使「删旧+建新」原子化。
// 中途任何失败整体回滚，不会出现“旧链已删、新链未建”的空窗（rehash 重建的收口保护）。
func (r *integrityRepository) ReplaceAnchors(ctx context.Context, anchors []*IntegrityAnchor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&AuditIntegrityAnchorPO{}).Error; err != nil {
			return err
		}
		if len(anchors) == 0 {
			return nil
		}
		pos := make([]*AuditIntegrityAnchorPO, len(anchors))
		for i, a := range anchors {
			pos[i] = &AuditIntegrityAnchorPO{
				ID:             a.ID,
				WindowStart:    a.WindowStart,
				WindowEnd:      a.WindowEnd,
				RowCount:       a.RowCount,
				WindowDigest:   a.WindowDigest,
				AnchorHash:     a.AnchorHash,
				PrevAnchorHash: a.PrevAnchorHash,
			}
		}
		for start := 0; start < len(pos); start += replaceAnchorChunk {
			end := start + replaceAnchorChunk
			if end > len(pos) {
				end = len(pos)
			}
			if err := tx.Create(pos[start:end]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
