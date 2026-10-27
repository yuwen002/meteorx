package service

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/config"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// newIntegrityGuard 构造一个启用态、固定 pepper 的摘要器。
func newIntegrityGuard(secret string) *IntegrityGuard {
	return NewIntegrityGuard(config.Config{Audit: config.AuditConfig{
		Integrity: config.AuditIntegrityConfig{Enabled: true, SecretKey: secret},
	}})
}

func sampleLog(id string) *model.AuditLog {
	return &model.AuditLog{
		ID:         id,
		UserID:     "u-1",
		Username:   "alice",
		TenantID:   "t-1",
		Module:     "user",
		Action:     model.ActionTypeCreate,
		Path:       "/api/v1/users",
		Method:     "POST",
		Result:     model.ResultSuccess,
		RiskLevel:  model.RiskLow,
		StatusCode: 200,
		Duration:   10,
		CreatedAt:  time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}
}

// TestCanonicalDeterministicAndSensitive 摘要须对同输入稳定、对任一被覆盖字段变化敏感、
// 对 RecordHash 自身不敏感（否则无法自证）。
func TestCanonicalDeterministicAndSensitive(t *testing.T) {
	g := newIntegrityGuard("pepper-A")

	base := sampleLog("id-1")
	h1 := g.Compute(base)
	h2 := g.Compute(sampleLog("id-1"))
	require.NotEmpty(t, h1)
	require.Equal(t, h1, h2, "同输入必须得到同摘要")

	// 字段敏感性：逐个改动任一被覆盖字段都应改变摘要
	cases := []struct {
		name string
		mut  func(*model.AuditLog)
	}{
		{"UserID", func(l *model.AuditLog) { l.UserID = "u-2" }},
		{"Module", func(l *model.AuditLog) { l.Module = "tenant" }},
		{"Action", func(l *model.AuditLog) { l.Action = model.ActionTypeDelete }},
		{"Path", func(l *model.AuditLog) { l.Path = "/api/v1/evil" }},
		{"RequestBody", func(l *model.AuditLog) { l.RequestBody = `{"x":1}` }},
		{"StatusCode", func(l *model.AuditLog) { l.StatusCode = 500 }},
		{"Duration", func(l *model.AuditLog) { l.Duration = 999 }},
		{"CreatedAt", func(l *model.AuditLog) { l.CreatedAt = l.CreatedAt.Add(time.Second) }},
	}
	for _, c := range cases {
		l := sampleLog("id-1")
		c.mut(l)
		require.NotEqual(t, h1, g.Compute(l), "改动 %s 必须改变摘要", c.name)
	}

	// RecordHash 不参与自摘要
	l := sampleLog("id-1")
	l.RecordHash = "whatever"
	require.Equal(t, h1, g.Compute(l), "RecordHash 自身不应改变计算结果")

	// 不同 pepper 得到不同摘要
	require.NotEqual(t, h1, newIntegrityGuard("pepper-B").Compute(base), "pepper 必须参与密钥派生")
}

// TestFoldRecordHashesSensitive 窗口折叠须对顺序与条数均敏感（否则删/插/重序无法检出）：
// 累加折叠下 fold(a,b) 既不等于换序 fold(b,a)，也不等于丢前缀 fold(b)。
func TestFoldRecordHashesSensitive(t *testing.T) {
	g := newIntegrityGuard("pepper-A")
	ab := g.FoldRecordHashes([]string{"h1", "h2"})
	require.NotEmpty(t, ab)

	require.Equal(t, ab, g.FoldRecordHashes([]string{"h1", "h2"}), "同输入须稳定")
	require.NotEqual(t, ab, g.FoldRecordHashes([]string{"h2", "h1"}), "重序必须改变摘要")
	require.NotEqual(t, ab, g.FoldRecordHashes([]string{"h2"}), "丢弃前缀行（删行）必须改变摘要")
	require.NotEqual(t, ab, g.FoldRecordHashes([]string{"h1", "h2", "h3"}), "追加行（插行）必须改变摘要")
	require.NotEqual(t, g.FoldRecordHashes([]string{"h1"}), g.FoldRecordHashes([]string{"h1", "h1"}), "重复条数必须敏感")
}

// TestIntegrityGuardDisabled 未启用/无 pepper 时 Compute 返回空、Enabled=false。
func TestIntegrityGuardDisabled(t *testing.T) {
	cases := []config.AuditIntegrityConfig{
		{Enabled: false, SecretKey: "x"},
		{Enabled: true, SecretKey: "   "},
		{Enabled: true, SecretKey: ""},
	}
	for _, c := range cases {
		g := NewIntegrityGuard(config.Config{Audit: config.AuditConfig{Integrity: c}})
		require.False(t, g.Enabled())
		require.Empty(t, g.Compute(sampleLog("id-1")))
		require.Empty(t, g.ComputeDigest([]byte("z")))
	}
}

// newIntegrityTestDB 具名 shared-cache 内存库，迁移审计全部表。
func newIntegrityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, repository.AutoMigrate(db))
	return db
}

// TestIntegrityServiceWriteThenVerify 启用完整性时经 service 写入应落 RecordHash，verify 应通过。
func TestIntegrityServiceWriteThenVerify(t *testing.T) {
	db := newIntegrityTestDB(t)
	guard := newIntegrityGuard("pepper-A")

	svc := NewAuditService(repository.NewAuditLogRepository(db), WithIntegrity(guard))
	_, err := svc.CreateLog(context.Background(), dto.CreateAuditLogReq{
		UserID: "u-1", Username: "alice", Module: "user",
		Action: model.ActionTypeCreate, Result: model.ResultSuccess,
	})
	require.NoError(t, err)
	require.NoError(t, svc.BatchCreateLogs(context.Background(), []dto.CreateAuditLogReq{
		{UserID: "u-2", Username: "bob", Module: "auth", Action: model.ActionTypeLogin, Result: model.ResultSuccess},
	}))

	// 落库行应带非空 RecordHash
	var rows []repository.AuditLogPO
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		require.NotEmpty(t, r.RecordHash, "启用完整性后写入行必须带行摘要")
	}

	ires := NewIntegrityService(guard, repository.NewIntegrityRepository(db))
	res, err := ires.Verify(context.Background())
	require.NoError(t, err)
	require.Equal(t, IntegrityVerdictVerified, res.Verdict)
	require.EqualValues(t, 2, res.CheckedRows)
	require.Zero(t, res.TamperedRows)
	require.Zero(t, res.LegacyRows)
}

// TestIntegrityServiceDetectsTamperedRow 直接改库内容而不动 record_hash，verify 须报 tampered 并定位到该行。
func TestIntegrityServiceDetectsTamperedRow(t *testing.T) {
	db := newIntegrityTestDB(t)
	guard := newIntegrityGuard("pepper-A")
	svc := NewAuditService(repository.NewAuditLogRepository(db), WithIntegrity(guard))
	created, err := svc.CreateLog(context.Background(), dto.CreateAuditLogReq{
		UserID: "u-1", Module: "user", Action: model.ActionTypeCreate, Result: model.ResultSuccess,
	})
	require.NoError(t, err)

	// 篡改：改内容、保摘要（攻击者无 pepper 无法同步重算）
	require.NoError(t, db.Model(&repository.AuditLogPO{}).Where("id = ?", created.ID).
		Update("path", "/api/v1/tampered").Error)

	res, err := NewIntegrityService(guard, repository.NewIntegrityRepository(db)).Verify(context.Background())
	require.NoError(t, err)
	require.Equal(t, IntegrityVerdictTampered, res.Verdict)
	require.EqualValues(t, 1, res.TamperedRows)
	require.Contains(t, res.TamperedSamples, created.ID)
}

// TestIntegrityServiceLegacyNotCountedAsFailure RecordHash 为空的旧行视为 legacy，不计入失败。
func TestIntegrityServiceLegacyNotCountedAsFailure(t *testing.T) {
	db := newIntegrityTestDB(t)
	guard := newIntegrityGuard("pepper-A")
	require.NoError(t, db.Create(&repository.AuditLogPO{
		ID: "legacy-1", UserID: "u-1", Module: "user", Action: model.ActionTypeCreate,
		Result: model.ResultSuccess, CreatedAt: time.Now(), // RecordHash 留空
	}).Error)

	res, err := NewIntegrityService(guard, repository.NewIntegrityRepository(db)).Verify(context.Background())
	require.NoError(t, err)
	require.Equal(t, IntegrityVerdictVerified, res.Verdict)
	require.EqualValues(t, 1, res.LegacyRows)
	require.Zero(t, res.TamperedRows)
}

// TestIntegrityServiceNotConfigured 未配置 pepper 时 verify 返回 not_configured 且不阻断。
func TestIntegrityServiceNotConfigured(t *testing.T) {
	db := newIntegrityTestDB(t)
	disabled := NewIntegrityGuard(config.Config{})
	res, err := NewIntegrityService(disabled, repository.NewIntegrityRepository(db)).Verify(context.Background())
	require.NoError(t, err)
	require.Equal(t, IntegrityVerdictNotConfigured, res.Verdict)
	require.False(t, res.Enabled)
}
