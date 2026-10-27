package service

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/config"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"
	"meteorx/pkg/idgen"

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

// newRotatingGuard 构造处于密钥轮换过渡期的摘要器：当前 pepper=cur、旧 pepper=prev。
func newRotatingGuard(cur, prev string) *IntegrityGuard {
	return NewIntegrityGuard(config.Config{Audit: config.AuditConfig{
		Integrity: config.AuditIntegrityConfig{Enabled: true, SecretKey: cur, PreviousSecretKey: prev},
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

// seedKeyedRow 以指定 guard 计算 record_hash，把一条带显式 created_at 的历史行直接落库
// （模拟密钥轮换前的旧数据，created_at 可控以便验证封口边界）。
func seedKeyedRow(t *testing.T, db *gorm.DB, g *IntegrityGuard, m *model.AuditLog) {
	t.Helper()
	m.RecordHash = g.Compute(m)
	po := repository.AuditLogPO{
		ID: m.ID, UserID: m.UserID, Username: m.Username, TenantID: m.TenantID,
		Module: m.Module, Action: m.Action, Resource: m.Resource, ResourceID: m.ResourceID,
		Method: m.Method, Path: m.Path, RequestBody: m.RequestBody, ResponseBody: m.ResponseBody,
		StatusCode: m.StatusCode, Result: m.Result, ErrorMessage: m.ErrorMessage,
		ClientIP: m.ClientIP, IPLocation: m.IPLocation, UserAgent: m.UserAgent, DeviceInfo: m.DeviceInfo,
		Duration: m.Duration, SessionID: m.SessionID, RequestID: m.RequestID, TraceID: m.TraceID,
		Referer: m.Referer, RiskLevel: m.RiskLevel, Tags: m.Tags,
		RecordHash: m.RecordHash, CreatedAt: m.CreatedAt,
	}
	require.NoError(t, db.Create(&po).Error)
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

// TestClassifyRowDualKey 双密钥下逐行分类：命中当前=Current、仅命中旧=Previous、
// 两把都不命中=None、空摘要=Legacy。
func TestClassifyRowDualKey(t *testing.T) {
	g := newRotatingGuard("pepper-NEW", "pepper-OLD")
	require.True(t, g.HasPrevious())

	// 旧 key 下的历史行：仅匹配旧 key
	oldGuard := newIntegrityGuard("pepper-OLD")
	oldRow := sampleLog("row-old")
	oldRow.RecordHash = oldGuard.Compute(oldRow)
	require.Equal(t, RowPrevious, g.ClassifyRow(oldRow))

	// 新 key 下的行：匹配当前
	newRow := sampleLog("row-new")
	newRow.RecordHash = g.Compute(newRow)
	require.Equal(t, RowCurrent, g.ClassifyRow(newRow))

	// 两把 key 都对不上：真篡改
	evil := sampleLog("row-evil")
	evil.RecordHash = newIntegrityGuard("pepper-EVIL").Compute(evil)
	require.Equal(t, RowNone, g.ClassifyRow(evil))

	// 空摘要：legacy
	legacy := sampleLog("row-legacy")
	require.Equal(t, RowLegacy, g.ClassifyRow(legacy))

	// 非轮换期（无旧 key）时旧 key 行应判 None（无从识别为待重算）
	single := newIntegrityGuard("pepper-NEW")
	require.False(t, single.HasPrevious())
	require.Equal(t, RowNone, single.ClassifyRow(oldRow))
}

// TestRehashMigratesPreviousRows 轮换过渡期：旧 key 历史行 verify 为 pending_rehash，
// rehash 后全部迁到新 key、verify 回到 verified，锚点链按新 key 重建。
func TestRehashMigratesPreviousRows(t *testing.T) {
	db := newIntegrityTestDB(t)
	irepo := repository.NewIntegrityRepository(db)

	// 以旧 pepper 写入两行历史数据，created_at 落在久远过去（早于 grace，重建时会被封口）
	oldGuard := newIntegrityGuard("pepper-OLD")
	ra := sampleLog("row-a")
	rb := sampleLog("row-b")
	rb.CreatedAt = ra.CreatedAt.Add(time.Minute)
	seedKeyedRow(t, db, oldGuard, ra)
	seedKeyedRow(t, db, oldGuard, rb)

	// 轮换：当前=NEW、旧=OLD
	rot := newRotatingGuard("pepper-NEW", "pepper-OLD")
	ires := NewIntegrityService(rot, irepo)

	res, err := ires.Verify(context.Background())
	require.NoError(t, err)
	require.Equal(t, IntegrityVerdictPendingRehash, res.Verdict)
	require.EqualValues(t, 2, res.PendingRehashRows)
	require.Zero(t, res.TamperedRows)

	rr, err := ires.Rehash(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 2, rr.RehashedRows)
	require.Greater(t, rr.AnchorsRebuilt, int64(0), "历史行早已越过 grace，rehash 应重建锚点链")

	// 重建的锚点窗口右端都不得越过当前时刻（防未来窗吞并后续写入行，见 B）
	var future int64
	require.NoError(t, db.Model(&repository.AuditIntegrityAnchorPO{}).
		Where("window_end > ?", time.Now()).Count(&future).Error)
	require.Zero(t, future, "重建锚点的窗口右端不得进入未来")

	// 迁移后单看新 key 即应 verified（旧行已重算）
	res2, err := NewIntegrityService(newIntegrityGuard("pepper-NEW"), irepo).Verify(context.Background())
	require.NoError(t, err)
	require.Equal(t, IntegrityVerdictVerified, res2.Verdict)
	require.Zero(t, res2.TamperedRows)
	require.Zero(t, res2.PendingRehashRows)
}

// TestRehashNoFutureWindowSwallowsLaterRow 锁死 B：rehash 重建不得产生以未来时刻收尾的窗口。
// 历史行落在 now-30min（早于 grace 会被封口，但其 +1h 窗口右端会越入未来）；若产生未来窗，
// rehash 后新写入的行会落进该已封窗口，令校验重放时窗口行数/摘要漂移而误报 tampered。
func TestRehashNoFutureWindowSwallowsLaterRow(t *testing.T) {
	db := newIntegrityTestDB(t)
	irepo := repository.NewIntegrityRepository(db)

	oldGuard := newIntegrityGuard("pepper-OLD")
	recent := sampleLog("row-recent")
	recent.CreatedAt = time.Now().UTC().Truncate(time.Second).Add(-30 * time.Minute)
	seedKeyedRow(t, db, oldGuard, recent)

	rot := newRotatingGuard("pepper-NEW", "pepper-OLD")
	_, err := NewIntegrityService(rot, irepo).Rehash(context.Background())
	require.NoError(t, err)

	// 重建后任何锚点窗口右端均不得超过当前时刻
	var future int64
	require.NoError(t, db.Model(&repository.AuditIntegrityAnchorPO{}).
		Where("window_end > ?", time.Now()).Count(&future).Error)
	require.Zero(t, future, "封口窗口右端不得进入未来")

	// rehash 后以新 key 再写一行（created_at≈now）：不得落入任何已封窗口
	newSvc := NewAuditService(repository.NewAuditLogRepository(db), WithIntegrity(newIntegrityGuard("pepper-NEW")))
	_, err = newSvc.CreateLog(context.Background(), dto.CreateAuditLogReq{
		UserID: "u-2", Module: "user", Action: model.ActionTypeCreate, Result: model.ResultSuccess,
	})
	require.NoError(t, err)

	res, err := NewIntegrityService(newIntegrityGuard("pepper-NEW"), irepo).Verify(context.Background())
	require.NoError(t, err)
	require.Equal(t, IntegrityVerdictVerified, res.Verdict, "rehash 后新写入行不得落入未来窗导致误报")
	require.Zero(t, res.TamperedRows)
	require.Zero(t, len(res.AnomalyAnchorIDs))
}

// TestReplaceAnchorsReplacesAtomically 锁死 A：ReplaceAnchors 在单事务内原子地删旧+建新，
// 不会残留旧链、也不会因建新失败而丢旧链（空切片=仅清空）。
func TestReplaceAnchorsReplacesAtomically(t *testing.T) {
	db := newIntegrityTestDB(t)
	irepo := repository.NewIntegrityRepository(db)
	base := time.Now().UTC().Truncate(time.Second)
	mk := func(i int, end time.Time) *repository.IntegrityAnchor {
		return &repository.IntegrityAnchor{
			ID: idgen.New(), WindowStart: end.Add(-time.Hour), WindowEnd: end,
			RowCount: int64(i + 1), WindowDigest: "d", AnchorHash: "a", PrevAnchorHash: "",
		}
	}

	old := []*repository.IntegrityAnchor{mk(0, base), mk(1, base.Add(time.Hour)), mk(2, base.Add(2*time.Hour))}
	require.NoError(t, irepo.ReplaceAnchors(context.Background(), old))
	var n int64
	require.NoError(t, db.Model(&repository.AuditIntegrityAnchorPO{}).Count(&n).Error)
	require.EqualValues(t, 3, n)

	fresh := []*repository.IntegrityAnchor{mk(0, base.Add(3*time.Hour)), mk(1, base.Add(4*time.Hour))}
	require.NoError(t, irepo.ReplaceAnchors(context.Background(), fresh))
	var pos []repository.AuditIntegrityAnchorPO
	require.NoError(t, db.Order("window_end ASC").Find(&pos).Error)
	require.Len(t, pos, 2)
	require.Equal(t, fresh[0].ID, pos[0].ID)
	require.Equal(t, fresh[1].ID, pos[1].ID)

	require.NoError(t, irepo.ReplaceAnchors(context.Background(), nil))
	require.NoError(t, db.Model(&repository.AuditIntegrityAnchorPO{}).Count(&n).Error)
	require.Zero(t, n)
}

// TestRehashDoesNotWhitelistTampered 真篡改行（两把 key 都不命中）不会被 rehash 洗白。
func TestRehashDoesNotWhitelistTampered(t *testing.T) {
	db := newIntegrityTestDB(t)
	irepo := repository.NewIntegrityRepository(db)

	oldGuard := newIntegrityGuard("pepper-OLD")
	svc := NewAuditService(repository.NewAuditLogRepository(db), WithIntegrity(oldGuard))
	created, err := svc.CreateLog(context.Background(), dto.CreateAuditLogReq{
		UserID: "u-1", Module: "user", Action: model.ActionTypeCreate, Result: model.ResultSuccess,
	})
	require.NoError(t, err)
	// 篡改：改内容、保留旧 key 算出的摘要 → 两把 key 都对不上
	require.NoError(t, db.Model(&repository.AuditLogPO{}).Where("id = ?", created.ID).
		Update("path", "/api/v1/tampered").Error)

	rot := newRotatingGuard("pepper-NEW", "pepper-OLD")
	ires := NewIntegrityService(rot, irepo)

	res, err := ires.Verify(context.Background())
	require.NoError(t, err)
	require.Equal(t, IntegrityVerdictTampered, res.Verdict)
	require.EqualValues(t, 1, res.TamperedRows)

	rr, err := ires.Rehash(context.Background())
	require.NoError(t, err)
	require.Zero(t, rr.RehashedRows, "篡改行不属于待重算，不应被重写")

	// rehash 后仍应报 tampered（篡改未被洗白）
	res2, err := ires.Verify(context.Background())
	require.NoError(t, err)
	require.Equal(t, IntegrityVerdictTampered, res2.Verdict)
	require.EqualValues(t, 1, res2.TamperedRows)
}

// TestRehashRequiresPrevious 非轮换期（未配旧 pepper）时 Rehash 返回前置错误。
func TestRehashRequiresPrevious(t *testing.T) {
	db := newIntegrityTestDB(t)
	single := newIntegrityGuard("pepper-NEW")
	_, err := NewIntegrityService(single, repository.NewIntegrityRepository(db)).Rehash(context.Background())
	require.ErrorIs(t, err, ErrIntegrityNoPreviousKey)

	disabled := NewIntegrityGuard(config.Config{})
	_, err = NewIntegrityService(disabled, repository.NewIntegrityRepository(db)).Rehash(context.Background())
	require.ErrorIs(t, err, ErrIntegrityNotEnabled)
}
