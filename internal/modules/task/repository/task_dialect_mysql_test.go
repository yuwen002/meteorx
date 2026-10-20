package repository

// 本文件验证**只能由真实 MySQL 执行**的方言查询：TIMESTAMPADD / JSON_CONTAINS+JSON_QUOTE /
// FIELD() / TIMESTAMPDIFF。常规仓储测试用 sqlmock，只断言生成的 SQL **文本**、从不执行；
// 而集成测试库是 sqlite（不支持这些函数，故 integration 里刻意绕开了 List 默认排序）。
// 结果是这几处方言的**语义**此前从未在任何地方被真正跑过——单位错位（分钟 vs 纳秒）、
// 对历史 TEXT 值报“Invalid JSON text”这类错误都能一路绿到生产。
//
// 门控策略：默认 t.Skip，需显式 METEORX_MYSQL_DIALECT_TEST=1 才执行。
// CI 的 Unit Tests job 本就已 provision MySQL 8.0 服务并注入同名 METEORX_DB_* 环境变量
// （此前无测试连接，属闲置），因此加一行开关即可让用例在 CI 真实运行；本地无 MySQL 时依旧全绿。
// 为防止误连业务库，另外要求 METEORX_DB_NAME 必须以 _test 结尾。

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"meteorx/internal/modules/task/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// openMySQLDialectDB 连接 CI 注入的真实 MySQL 并重建 tasks 表，返回可用仓储。
// 环境不满足（未开启 / 非测试库 / 连不上 / 无建表权限）时跳过，不当作失败。
func openMySQLDialectDB(t *testing.T) *taskRepository {
	t.Helper()

	if os.Getenv("METEORX_MYSQL_DIALECT_TEST") != "1" {
		t.Skip("跳过 MySQL 方言测试（设置 METEORX_MYSQL_DIALECT_TEST=1 并注入 METEORX_DB_* 后执行）")
	}
	name := os.Getenv("METEORX_DB_NAME")
	if !strings.HasSuffix(name, "_test") {
		t.Skipf("跳过 MySQL 方言测试：METEORX_DB_NAME=%q 不以 _test 结尾，拒绝操作疑似业务库", name)
	}
	port := 3306
	if p := os.Getenv("METEORX_DB_PORT"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil {
			t.Skipf("跳过 MySQL 方言测试：METEORX_DB_PORT=%q 非法", p)
		}
		port = v
	}

	// 与 bootstrap.dsn 保持一致的连接参数：charset/parseTime/loc 任一缺失都会让时间列
	// 以字符串返回或按 UTC 解析，从而让方言断言失真。
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("METEORX_DB_USER"), os.Getenv("METEORX_DB_PASSWORD"),
		os.Getenv("METEORX_DB_HOST"), port, name)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("跳过 MySQL 方言测试：无法连接 %s:%d/%s（%v）", os.Getenv("METEORX_DB_HOST"), port, name, err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	// 每个用例都从干净表开始，避免历史残留影响计数与排序断言。
	if err := db.Migrator().DropTable(&TaskPO{}); err != nil {
		t.Skipf("跳过 MySQL 方言测试：无权限重建 tasks 表（%v）", err)
	}
	require.NoError(t, db.AutoMigrate(&TaskPO{}), "MySQL 建表失败")

	return &taskRepository{db: db}
}

// mustCreate 走仓储真实写路径落一条任务——顺带验证 remind_before 由 Duration 到分钟的
// 换算与 JSON 标签列的写入在真实 MySQL 列类型上能落能读。
func mustCreate(t *testing.T, repo *taskRepository, task *model.Task) {
	t.Helper()
	require.NoError(t, repo.Create(context.Background(), task), "写入任务失败："+task.ID)
}

// dialectTask 构造一条测试用租户任务骨架。
func dialectTask(id, status string, due *time.Time) *model.Task {
	return &model.Task{
		ID: id, TenantID: "t1", CreatorID: "u1", AssigneeID: "u1",
		Title: "方言用例 " + id, Status: status, Priority: model.TaskPriorityNormal,
		Visibility: model.TaskVisibilityTenant, DueDate: due,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

// taskIDs 提取任务 ID 供集合断言。
func taskIDs(tasks []*model.Task) []string {
	ids := make([]string, 0, len(tasks))
	for _, t := range tasks {
		ids = append(ids, t.ID)
	}
	return ids
}

// TestTaskDialect_FindDueForReminder 在真实 MySQL 上执行含 TIMESTAMPADD 与 CASE 的提醒扫描 SQL，
// 锁定三件 sqlite/sqlmock 都给不出的结论：
//  1. remind_before 的单位确实是**分钟**（若 PO 误存了 Duration 纳秒原值，阈值会飞到万年后，
//     本不应命中的 lead5 会错误命中）；
//  2. CASE 分支选择与 <= 的边界包含性；
//  3. 逾期重复提醒的冷却基准比较。
func TestTaskDialect_FindDueForReminder(t *testing.T) {
	repo := openMySQLDialectDB(t)
	// 截断到毫秒：datetime 列与入参需在统一精度上比较，否则恰好相等的边界断言
	// 可能因纳秒位的舍入方向而抖动。
	now := time.Now().Truncate(time.Millisecond)
	horizon := now.Add(time.Hour)
	cutoff := now.Add(-24 * time.Hour)

	at := func(d time.Duration) *time.Time { t := now.Add(d); return &t }
	lead := func(min int) *time.Duration { d := time.Duration(min) * time.Minute; return &d }

	type seed struct {
		task *model.Task
		want bool
		why  string
	}
	seeds := []seed{
		{dialectTask("lead600", model.TaskStatusPending, at(6*time.Hour)), true, "全局窗口外但任务级 600 分钟提前量拉进窗口"},
		{dialectTask("lead5", model.TaskStatusPending, at(6*time.Hour)), false, "任务级仅 5 分钟提前量（单位错位防护）"},
		{dialectTask("nulllead", model.TaskStatusPending, at(30*time.Minute)), true, "无任务级提前量，落在全局 horizon 内"},
		{dialectTask("bound", model.TaskStatusPending, at(time.Hour)), true, "截止日恰好等于 horizon，验证 <= 包含边界"},
		{dialectTask("far", model.TaskStatusPending, at(2*time.Hour)), false, "超出 horizon"},
		{dialectTask("overdue-never", model.TaskStatusPending, at(-2*time.Hour)), true, "已逾期且从未提醒"},
		{dialectTask("overdue-cooled", model.TaskStatusPending, at(-2*time.Hour)), true, "距上次提醒恰好等于冷却期"},
		{dialectTask("overdue-hot", model.TaskStatusPending, at(-2*time.Hour)), false, "已逾期但仍在冷却内"},
		{dialectTask("done", model.TaskStatusCompleted, at(-2*time.Hour)), false, "已完成不参与提醒"},
	}
	// 逐个补齐差异字段：提前量与上次提醒时间。
	seeds[0].task.RemindBefore = lead(600)
	seeds[1].task.RemindBefore = lead(5)
	seeds[6].task.ReminderSentAt = &cutoff
	seeds[7].task.ReminderSentAt = func() *time.Time { t := now.Add(-time.Hour); return &t }()

	var want []string
	for _, s := range seeds {
		mustCreate(t, repo, s.task)
		if s.want {
			want = append(want, s.task.ID)
		}
	}

	tasks, err := repo.FindDueForReminder(context.Background(), horizon, now, cutoff, 100)
	require.NoError(t, err)
	got := taskIDs(tasks)

	// 逐例报告未符预期的任务及其语义，便于定位到具体方言表达式。
	gotSet := make(map[string]bool, len(got))
	for _, id := range got {
		gotSet[id] = true
	}
	for _, s := range seeds {
		assert.Equal(t, s.want, gotSet[s.task.ID], "用例 %s（%s）命中不符预期", s.task.ID, s.why)
	}
	assert.ElementsMatch(t, want, got)
}

// TestTaskDialect_List_TagFilter 在真实 MySQL 上执行 JSON_CONTAINS(tags, JSON_QUOTE(?))。
// tags 是 TEXT 列而非 JSON 列，且无标签的任务会写入字面量 null（json.Marshal(nil)）——
// 这种历史值到底返回 0 行还是直接报 Invalid JSON text，只有真引擎能给出答案。
func TestTaskDialect_List_TagFilter(t *testing.T) {
	repo := openMySQLDialectDB(t)
	now := time.Now()

	hit := dialectTask("tag-hit", model.TaskStatusPending, &now)
	hit.Tags = []string{"urgent", "周报"}
	emptyArr := dialectTask("tag-empty-array", model.TaskStatusPending, &now)
	emptyArr.Tags = []string{}
	nullTags := dialectTask("tag-null", model.TaskStatusPending, &now) // 写入为字面量 null
	other := dialectTask("tag-other", model.TaskStatusPending, &now)
	other.Tags = []string{"later"}
	for _, s := range []*model.Task{hit, emptyArr, nullTags, other} {
		mustCreate(t, repo, s)
	}
	// 历史态：tags 列引入前由 AutoMigrate 补列的老行，其值为 SQL NULL（而非 'null' 文本）。
	// JSON_CONTAINS(NULL, ...) 在 MySQL 下是报错还是仅不命中，决定一行老数据会不会把整个
	// 租户的标签筛选直接打成 500——这只能靠真引擎给出答案。
	require.NoError(t, repo.db.Exec(
		"INSERT INTO tasks (id,tenant_id,creator_id,assignee_id,title,status,priority,visibility,tags,created_at,updated_at)"+
			" VALUES (?,?,?,?,?,'pending','normal','tenant',NULL,?,?)",
		"legacy-null-col", "t1", "u1", "u1", "老数据无标签列", time.Now(), time.Now()).Error)

	tasks, total, err := repo.List(context.Background(), ListFilter{
		TenantID: "t1", UserID: "u1", Tag: "urgent", Page: 1, PageSize: 20,
	})
	require.NoError(t, err, "JSON_CONTAINS 遇到空标签历史值不得报错")
	assert.Equal(t, int64(1), total)
	assert.ElementsMatch(t, []string{"tag-hit"}, taskIDs(tasks))

	// 中文多字节标签同样靠 JSON_QUOTE 精确匹配
	tasks, _, err = repo.List(context.Background(), ListFilter{
		TenantID: "t1", UserID: "u1", Tag: "周报", Page: 1, PageSize: 20,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"tag-hit"}, taskIDs(tasks))
}

// TestTaskDialect_List_DefaultSort 验证默认排序里的 FIELD(priority,'urgent','high','normal','low')
// 真的按业务优先级而非字母序产出（sqlite 无 FIELD，集成测试已绕开此路径）。
func TestTaskDialect_List_DefaultSort(t *testing.T) {
	repo := openMySQLDialectDB(t)
	now := time.Now()

	for _, p := range []string{model.TaskPriorityLow, model.TaskPriorityUrgent,
		model.TaskPriorityNormal, model.TaskPriorityHigh} {
		task := dialectTask("sort-"+p, model.TaskStatusPending, &now)
		task.Priority = p
		mustCreate(t, repo, task)
	}

	tasks, _, err := repo.List(context.Background(), ListFilter{
		TenantID: "t1", UserID: "u1", Page: 1, PageSize: 20,
	})
	require.NoError(t, err)
	assert.Equal(t,
		[]string{"sort-urgent", "sort-high", "sort-normal", "sort-low"},
		taskIDs(tasks), "默认应按 urgent>high>normal>low 排序")
}

// TestTaskDialect_Stats_AvgTimestampdiff 验证 Stats 里的 AVG(TIMESTAMPDIFF(SECOND, ...))：
// 一是真实秒差均值必须正确，二是没有任何已完成任务时聚合返回 NULL 必须被容错
// （否则新租户调 /tasks/stats 直接 500）。
func TestTaskDialect_Stats_AvgTimestampdiff(t *testing.T) {
	repo := openMySQLDialectDB(t)
	completed := time.Now().Truncate(time.Second)

	oneHour := dialectTask("avg-1h", model.TaskStatusCompleted, &completed)
	start := completed.Add(-time.Hour)
	oneHour.StartedAt, oneHour.CompletedAt = &start, &completed
	twoHours := dialectTask("avg-2h", model.TaskStatusCompleted, &completed)
	start2 := completed.Add(-2 * time.Hour)
	twoHours.StartedAt, twoHours.CompletedAt = &start2, &completed
	// 缺 started_at 的完成任务不应参与均值，但仍计入 Completed 计数
	noStart := dialectTask("avg-no-start", model.TaskStatusCompleted, &completed)
	noStart.CompletedAt = &completed
	for _, s := range []*model.Task{oneHour, twoHours, noStart} {
		mustCreate(t, repo, s)
	}

	stats, err := repo.Stats(context.Background(), ListFilter{TenantID: "t1", UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.Completed)
	assert.Equal(t, int64(5400), stats.AvgHandleSeconds, "平均处理耗时应为 (3600+7200)/2")

	// 另建一个没有任何任务的租户：各计数归零且聚合行为 NULL，必须降级为 0 而非报错
	mustCreate(t, repo, func() *model.Task {
		t2 := dialectTask("other-tenant", model.TaskStatusPending, nil)
		t2.TenantID = "t2"
		return t2
	}())

	stats, err = repo.Stats(context.Background(), ListFilter{TenantID: "t9", UserID: "u1"})
	require.NoError(t, err, "无任何任务时 Stats 不得因聚合返回 NULL 而报错")
	assert.Equal(t, int64(0), stats.Total)
	assert.Equal(t, int64(0), stats.AvgHandleSeconds)
}
