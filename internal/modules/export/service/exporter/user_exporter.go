package exporter

import (
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"meteorx/internal/modules/export/model"

	"gorm.io/gorm"
)

// userExporter 导出租户下的用户列表（不含密码等敏感字段）。
type userExporter struct{ db *gorm.DB }

// NewUserExporter 创建用户导出器
func NewUserExporter(db *gorm.DB) Exporter { return &userExporter{db: db} }

// Resource 返回资源标识
func (e *userExporter) Resource() string { return model.ResourceUser }

// userRow 承载 users 表导出所需列，可空列用指针承接 NULL。
type userRow struct {
	ID          string     `gorm:"column:id"`
	Username    string     `gorm:"column:username"`
	Nickname    *string    `gorm:"column:nickname"`
	Email       *string    `gorm:"column:email"`
	Phone       *string    `gorm:"column:phone"`
	Department  *string    `gorm:"column:department"`
	Position    *string    `gorm:"column:position"`
	Status      int        `gorm:"column:status"`
	IsMaster    bool       `gorm:"column:is_master"`
	LastLoginAt *time.Time `gorm:"column:last_login_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
}

// Stream 分批流式写出用户 CSV，返回数据行数（不含表头）。
func (e *userExporter) Stream(ctx context.Context, tenantID string, filters map[string]any, w *csv.Writer) (int64, error) {
	header := []string{"ID", "用户名", "昵称", "邮箱", "手机号", "部门", "岗位", "状态", "是否主管理员", "最后登录时间", "创建时间"}
	if err := writeHeader(w, header); err != nil {
		return 0, err
	}

	var total int64
	q := e.db.WithContext(ctx).Table("users").
		Select("id, username, nickname, email, phone, department, position, status, is_master, last_login_at, created_at").
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
	if s := filterString(filters, "status"); s != "" {
		q = q.Where("status = ?", s)
	}

	var batch []userRow
	err := q.FindInBatches(&batch, exportBatchSize, func(res *gorm.DB, _ int) error {
		if res.Error != nil {
			return res.Error
		}
		records := make([][]string, 0, len(batch))
		for _, r := range batch {
			records = append(records, []string{
				r.ID, r.Username, fmtStr(r.Nickname), fmtStr(r.Email), fmtStr(r.Phone),
				fmtStr(r.Department), fmtStr(r.Position), fmt.Sprintf("%d", r.Status),
				boolStr(r.IsMaster), fmtNullTime(r.LastLoginAt), fmtTime(r.CreatedAt),
			})
		}
		n, err := writeRows(w, records)
		total += n
		return err
	}).Error
	if err != nil {
		return total, err
	}
	return total, nil
}
