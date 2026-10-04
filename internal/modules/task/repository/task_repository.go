// Package repository 实现任务模块的 GORM 仓储层。
// 提供任务的 CRUD、多条件分页查询与状态统计，查询自动遵循多租户隔离与可见范围规则。
package repository

import (
	"context"
	"encoding/json"
	"time"

	"meteorx/internal/modules/task/model"

	"gorm.io/gorm"
)

// TaskPO 任务持久化对象，映射 tasks 表。
// Tags 以 JSON 数组字符串存储；DeletedAt 使用 GORM 软删除。
type TaskPO struct {
	ID           string         `gorm:"primaryKey;size:26;comment:任务ID"`
	TenantID     string         `gorm:"index;size:26;not null;comment:租户ID"`
	CreatorID    string         `gorm:"index;size:26;not null;comment:创建人ID"`
	AssigneeID   string         `gorm:"index;size:26;comment:负责人ID"`
	Title        string         `gorm:"size:200;not null;comment:标题"`
	Description  string         `gorm:"type:text;comment:描述"`
	Status       string         `gorm:"size:20;default:pending;index;comment:状态"`
	Priority     string         `gorm:"size:20;default:normal;index;comment:优先级"`
	DueDate      *time.Time     `gorm:"comment:截止时间"`
	RemindBefore *int           `gorm:"comment:提醒提前量(分钟)"`
	Tags         string         `gorm:"type:text;comment:标签(JSON数组)"`
	Visibility   string         `gorm:"size:20;default:personal;index;comment:可见范围"`
	Recurrence   string         `gorm:"size:20;comment:重复周期(daily/weekly/monthly)"`
	StartedAt    *time.Time     `gorm:"comment:开始时间"`
	CompletedAt  *time.Time     `gorm:"comment:完成时间"`
	ReminderSent *time.Time     `gorm:"column:reminder_sent_at;comment:到期提醒发送时间"`
	CreatedAt    time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt    gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

// TableName 返回任务表名 tasks。
func (TaskPO) TableName() string {
	return "tasks"
}

// toDomain 将持久化对象转换为领域模型 model.Task。
func (p TaskPO) toDomain() *model.Task {
	var tags []string
	if p.Tags != "" {
		_ = json.Unmarshal([]byte(p.Tags), &tags)
	}
	t := &model.Task{
		ID:             p.ID,
		TenantID:       p.TenantID,
		CreatorID:      p.CreatorID,
		AssigneeID:     p.AssigneeID,
		Title:          p.Title,
		Description:    p.Description,
		Status:         p.Status,
		Priority:       p.Priority,
		DueDate:        p.DueDate,
		Tags:           tags,
		Visibility:     p.Visibility,
		Recurrence:     p.Recurrence,
		StartedAt:      p.StartedAt,
		CompletedAt:    p.CompletedAt,
		ReminderSentAt: p.ReminderSent,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
	if p.RemindBefore != nil {
		d := time.Duration(*p.RemindBefore) * time.Minute
		t.RemindBefore = &d
	}
	if p.DeletedAt.Valid {
		dt := p.DeletedAt.Time
		t.DeletedAt = &dt
	}
	return t
}

// toPO 将领域模型转换为持久化对象。
func fromDomain(t *model.Task) TaskPO {
	tagsJSON, _ := json.Marshal(t.Tags)
	return TaskPO{
		ID:           t.ID,
		TenantID:     t.TenantID,
		CreatorID:    t.CreatorID,
		AssigneeID:   t.AssigneeID,
		Title:        t.Title,
		Description:  t.Description,
		Status:       t.Status,
		Priority:     t.Priority,
		DueDate:      t.DueDate,
		RemindBefore: remindBeforeMinutes(t.RemindBefore),
		Tags:         string(tagsJSON),
		Visibility:   t.Visibility,
		Recurrence:   t.Recurrence,
		StartedAt:    t.StartedAt,
		CompletedAt:  t.CompletedAt,
		ReminderSent: t.ReminderSentAt,
	}
}

// remindBeforeMinutes 将提醒提前量 Duration 转为分钟数供持久化（截断到整分钟）；nil 保持 nil。
func remindBeforeMinutes(d *time.Duration) *int {
	if d == nil {
		return nil
	}
	m := int(d.Minutes())
	return &m
}

// taskRepository TaskRepository 的 GORM 实现。
type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository 创建任务仓储实例。
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

// AutoMigrate 自动迁移任务表结构。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&TaskPO{})
}

// Create 新建任务记录。
func (r *taskRepository) Create(ctx context.Context, task *model.Task) error {
	record := fromDomain(task)
	return r.db.WithContext(ctx).Create(&record).Error
}

// GetByID 按主键查询未删除的任务。
func (r *taskRepository) GetByID(ctx context.Context, id string) (*model.Task, error) {
	var p TaskPO
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return p.toDomain(), nil
}

// Update 更新任务的可变字段（使用 map 以支持将截止日/负责人清空为空）。
func (r *taskRepository) Update(ctx context.Context, task *model.Task) error {
	tagsJSON, _ := json.Marshal(task.Tags)
	updates := map[string]interface{}{
		"title":            task.Title,
		"description":      task.Description,
		"status":           task.Status,
		"priority":         task.Priority,
		"due_date":         task.DueDate,
		"remind_before":    remindBeforeMinutes(task.RemindBefore),
		"tags":             string(tagsJSON),
		"assignee_id":      task.AssigneeID,
		"visibility":       task.Visibility,
		"recurrence":       task.Recurrence,
		"started_at":       task.StartedAt,
		"completed_at":     task.CompletedAt,
		"reminder_sent_at": task.ReminderSentAt,
		"updated_at":       time.Now(),
	}
	return r.db.WithContext(ctx).Model(&TaskPO{}).Where("id = ?", task.ID).Updates(updates).Error
}

// Delete 软删除任务。
func (r *taskRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&TaskPO{}, "id = ?", id).Error
}

// GetByIDUnscoped 按主键查询任务，包含已软删除记录（供恢复/永久删除校验归属）。
func (r *taskRepository) GetByIDUnscoped(ctx context.Context, id string) (*model.Task, error) {
	var p TaskPO
	if err := r.db.WithContext(ctx).Unscoped().Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return p.toDomain(), nil
}

// ListDeleted 分页查询回收站（已软删除）任务，仅列出当前用户创建的任务。
func (r *taskRepository) ListDeleted(ctx context.Context, f ListFilter) ([]*model.Task, int64, error) {
	q := r.db.WithContext(ctx).Unscoped().Model(&TaskPO{}).
		Where("deleted_at IS NOT NULL").
		Where("tenant_id = ?", f.TenantID)
	if f.UserID != "" {
		// 回收站仅展示自己删除的任务，避免跨成员泄露
		q = q.Where("creator_id = ?", f.UserID)
	}
	if f.Keyword != "" {
		kw := "%" + f.Keyword + "%"
		q = q.Where("title LIKE ? OR description LIKE ?", kw, kw)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if f.PageSize > 0 {
		q = q.Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize)
	}

	var records []TaskPO
	if err := q.Order("deleted_at DESC").Find(&records).Error; err != nil {
		return nil, 0, err
	}
	tasks := make([]*model.Task, 0, len(records))
	for _, rec := range records {
		tasks = append(tasks, rec.toDomain())
	}
	return tasks, total, nil
}

// Restore 恢复已软删除的任务（将 deleted_at 置空）。
func (r *taskRepository) Restore(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Unscoped().Model(&TaskPO{}).
		Where("id = ?", id).Update("deleted_at", nil).Error
}

// PermanentDelete 永久（物理）删除任务，不可恢复。
func (r *taskRepository) PermanentDelete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&TaskPO{}).Error
}

// FindDueForReminder 查询需要发送到期/逾期提醒的未完成任务，供定时提醒任务扫描。
// 命中条件：
//   - 提醒到点：任务自带 remind_before 时以 now+remind_before 为阈值，否则使用全局 horizon；
//   - 尚未提醒过（reminder_sent_at 为空）；或已逾期（due<now）且距上次提醒已超过冷却期（reminder_sent_at<=cutoff）。
func (r *taskRepository) FindDueForReminder(ctx context.Context, horizon, now, cutoff time.Time, limit int) ([]*model.Task, error) {
	q := r.db.WithContext(ctx).Model(&TaskPO{}).
		Where("status IN ?", []string{model.TaskStatusPending, model.TaskStatusInProgress}).
		Where("due_date IS NOT NULL AND due_date <= CASE WHEN remind_before IS NOT NULL THEN TIMESTAMPADD(MINUTE, remind_before, ?) ELSE ? END", now, horizon).
		Where("reminder_sent_at IS NULL OR (due_date < ? AND reminder_sent_at <= ?)", now, cutoff).
		Order("due_date ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}

	var records []TaskPO
	if err := q.Find(&records).Error; err != nil {
		return nil, err
	}
	tasks := make([]*model.Task, 0, len(records))
	for _, rec := range records {
		tasks = append(tasks, rec.toDomain())
	}
	return tasks, nil
}

// MarkReminderSent 标记任务已发送到期提醒，避免重复提醒。
func (r *taskRepository) MarkReminderSent(ctx context.Context, id string, t time.Time) error {
	return r.db.WithContext(ctx).Model(&TaskPO{}).
		Where("id = ?", id).Update("reminder_sent_at", t).Error
}

// applyVisibility 依据可见范围与用户身份追加过滤条件，返回新查询。
func applyVisibility(q *gorm.DB, f ListFilter) *gorm.DB {
	q = q.Where("tenant_id = ?", f.TenantID)
	switch f.Visibility {
	case model.TaskVisibilityPersonal:
		q = q.Where("visibility = ? AND creator_id = ?", model.TaskVisibilityPersonal, f.UserID)
	case model.TaskVisibilityTenant:
		q = q.Where("visibility = ?", model.TaskVisibilityTenant)
	default:
		// 全部视图：租户协作任务 + 自己创建/负责的个人任务
		q = q.Where("visibility = ? OR creator_id = ? OR assignee_id = ?",
			model.TaskVisibilityTenant, f.UserID, f.UserID)
	}
	return q
}

// List 按过滤条件分页查询任务列表。
func (r *taskRepository) List(ctx context.Context, f ListFilter) ([]*model.Task, int64, error) {
	q := applyVisibility(r.db.WithContext(ctx).Model(&TaskPO{}), f)

	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Priority != "" {
		q = q.Where("priority = ?", f.Priority)
	}
	if f.AssigneeID != "" {
		q = q.Where("assignee_id = ?", f.AssigneeID)
	}
	if f.Keyword != "" {
		kw := "%" + f.Keyword + "%"
		q = q.Where("title LIKE ? OR description LIKE ?", kw, kw)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q = applySort(q, f)
	if f.PageSize > 0 {
		q = q.Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize)
	}

	var records []TaskPO
	if err := q.Find(&records).Error; err != nil {
		return nil, 0, err
	}
	tasks := make([]*model.Task, 0, len(records))
	for _, rec := range records {
		tasks = append(tasks, rec.toDomain())
	}
	return tasks, total, nil
}

// applySort 依据排序字段追加 ORDER BY，仅允许白名单列，默认按优先级与创建时间排序。
func applySort(q *gorm.DB, f ListFilter) *gorm.DB {
	allowed := map[string]string{
		"created_at": "created_at",
		"updated_at": "updated_at",
		"due_date":   "due_date",
		"priority":   "priority",
		"title":      "title",
	}
	col, ok := allowed[f.SortBy]
	if !ok {
		// 默认：未完成在前，其次按优先级（urgent>high>normal>low），再按创建时间倒序
		return q.Order("CASE WHEN status = 'completed' THEN 1 ELSE 0 END ASC").
			Order("FIELD(priority,'urgent','high','normal','low') ASC").
			Order("created_at DESC")
	}
	order := "ASC"
	if f.SortOrder != "" {
		order = f.SortOrder
	}
	return q.Order(col + " " + order)
}

// Stats 统计各状态任务数量及逾期数量。
func (r *taskRepository) Stats(ctx context.Context, f ListFilter) (*TaskStatusStats, error) {
	base := func() *gorm.DB {
		return applyVisibility(r.db.WithContext(ctx).Model(&TaskPO{}), f)
	}

	stats := &TaskStatusStats{}
	if err := base().Where("status = ?", model.TaskStatusPending).Count(&stats.Pending).Error; err != nil {
		return nil, err
	}
	if err := base().Where("status = ?", model.TaskStatusInProgress).Count(&stats.InProgress).Error; err != nil {
		return nil, err
	}
	if err := base().Where("status = ?", model.TaskStatusCompleted).Count(&stats.Completed).Error; err != nil {
		return nil, err
	}
	if err := base().Where("status <> ? AND due_date IS NOT NULL AND due_date < ?",
		model.TaskStatusCompleted, time.Now()).Count(&stats.Overdue).Error; err != nil {
		return nil, err
	}
	if err := base().Count(&stats.Total).Error; err != nil {
		return nil, err
	}
	// 平均处理耗时：对已完成且 started_at/completed_at 均有值的任务取秒级平均（依赖 MySQL TIMESTAMPDIFF）
	var dur struct {
		AvgHandleSeconds float64
	}
	if err := base().
		Where("status = ? AND started_at IS NOT NULL AND completed_at IS NOT NULL", model.TaskStatusCompleted).
		Select("AVG(TIMESTAMPDIFF(SECOND, started_at, completed_at)) AS avg_handle_seconds").
		Scan(&dur).Error; err != nil {
		return nil, err
	}
	stats.AvgHandleSeconds = int64(dur.AvgHandleSeconds)
	return stats, nil
}
