package integration

import (
	"net/http"
	"testing"
	"time"

	"meteorx/internal/config"
	"meteorx/internal/modules/audit/model"
	auditrepo "meteorx/internal/modules/audit/repository"
	"meteorx/internal/modules/audit/service"
	"meteorx/internal/testkit"

	"github.com/stretchr/testify/require"
)

// verifyResp 与 dto.IntegrityVerifyResp 对齐的最小解析结构。
type verifyResp struct {
	Enabled           bool   `json:"enabled"`
	Verdict           string `json:"verdict"`
	CheckedRows       int64  `json:"checked_rows"`
	LegacyRows        int64  `json:"legacy_rows"`
	TamperedRows      int64  `json:"tampered_rows"`
	PendingRehashRows int64  `json:"pending_rehash_rows"`
}

// rehashResp 与 dto.IntegrityRehashResp 对齐的最小解析结构。
type rehashResp struct {
	RehashedRows   int64  `json:"rehashed_rows"`
	AnchorsRebuilt int64  `json:"anchors_rebuilt"`
	Message        string `json:"message"`
}

// TestInteg_AuditIntegrity_DisabledByDefault 默认配置（未配 pepper）下：
// 超管触发校验返回 not_configured，非超管一律 403。锁住路由 + 权限 + 默认关。
func TestInteg_AuditIntegrity_DisabledByDefault(t *testing.T) {
	app := testkit.NewTestApp(t)
	sa := app.SuperadminHeader("sa", "t-1")
	member := app.AuthHeader("m", "t-1", []string{"member"})

	w := app.Do(t, http.MethodPost, "/api/v1/audit/integrity/verify", "", sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var res verifyResp
	testkit.DecodeData(t, w, &res)
	require.False(t, res.Enabled)
	require.Equal(t, "not_configured", res.Verdict)

	// 非超管无 audit:integrity:verify 权限 → 403
	w = app.Do(t, http.MethodPost, "/api/v1/audit/integrity/verify", "", member)
	require.Equal(t, http.StatusForbidden, w.Code, "非超管触发完整性校验应 403")
}

// TestInteg_AuditIntegrity_WriteHashThenVerifyAndTamper 端到端锁住生产接线：
// 开启 pepper 后，真实 HTTP 写路径（审计中间件 → 批量 service）必须落非空行摘要
// （证明 router 注入的 guard 确实用在了写侧），只读校验应 verified；直接改库一行内容
// 而不同步 record_hash（攻击者无 pepper）后，校验须报 tampered。
func TestInteg_AuditIntegrity_WriteHashThenVerifyAndTamper(t *testing.T) {
	app := testkit.NewTestAppOpt(t, func(cfg *config.Config) {
		cfg.Audit = config.AuditConfig{Integrity: config.AuditIntegrityConfig{
			Enabled: true, SecretKey: "integration-pepper-key",
		}}
	})
	sa := app.SuperadminHeader("sa", "t-1")

	// 触发一条会被审计中间件记录的写操作（DELETE → critical 审计）
	app.Do(t, http.MethodDelete, "/api/v1/users/nonexistent", "", sa)

	// 审计批量异步 flush，轮询直等真实写路径落下带摘要的行
	require.Eventually(t, func() bool {
		var n int64
		if err := app.DB.Model(&auditrepo.AuditLogPO{}).
			Where("record_hash <> ?", "").Count(&n).Error; err != nil {
			return false
		}
		return n >= 1
	}, 15*time.Second, 200*time.Millisecond, "开启完整性后真实写路径应落下带行摘要的审计日志")

	// 校验通过：所有带摘要行均可重算自证
	w := app.Do(t, http.MethodPost, "/api/v1/audit/integrity/verify", "", sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var res verifyResp
	testkit.DecodeData(t, w, &res)
	require.True(t, res.Enabled)
	require.Equal(t, "verified", res.Verdict, "写入未篡改时校验须通过")
	require.Positive(t, res.CheckedRows)
	require.Zero(t, res.TamperedRows)

	// 篡改：改一条已带摘要行的内容，保摘要不动（无 pepper 无法同步重算）
	var target auditrepo.AuditLogPO
	require.NoError(t, app.DB.Where("record_hash <> ?", "").First(&target).Error)
	require.NoError(t, app.DB.Model(&auditrepo.AuditLogPO{}).
		Where("id = ?", target.ID).Update("path", "/api/v1/tampered").Error)

	w = app.Do(t, http.MethodPost, "/api/v1/audit/integrity/verify", "", sa)
	require.Equal(t, http.StatusOK, w.Code)
	var res2 verifyResp
	testkit.DecodeData(t, w, &res2)
	require.Equal(t, "tampered", res2.Verdict, "改内容不改摘要须检出篡改")
	require.Positive(t, res2.TamperedRows)
}

// TestInteg_AuditIntegrity_KeyRotationRehash 端到端锁住密钥轮换迁移：
// 配当前=NEW、旧=OLD 的轮换态，手工植入一条旧 key 历史行→verify 为 pending_rehash；
// 超管调 rehash→行迁到新 key、verify 回到 verified；非超管调 rehash 一律 403。
func TestInteg_AuditIntegrity_KeyRotationRehash(t *testing.T) {
	app := testkit.NewTestAppOpt(t, func(cfg *config.Config) {
		cfg.Audit = config.AuditConfig{Integrity: config.AuditIntegrityConfig{
			Enabled: true, SecretKey: "pepper-NEW", PreviousSecretKey: "pepper-OLD",
		}}
	})
	sa := app.SuperadminHeader("sa", "t-1")
	member := app.AuthHeader("m", "t-1", []string{"member"})

	// 以旧 pepper 手工算一行摘要并直接入库（模拟轮换前历史数据）
	oldGuard := service.NewIntegrityGuard(config.Config{Audit: config.AuditConfig{
		Integrity: config.AuditIntegrityConfig{Enabled: true, SecretKey: "pepper-OLD"},
	}})
	// 以秒精度 UTC 时间戳落库，保证重算时的 CreatedAt 序列化与读回值逐字节一致。
	createdAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	legacy := &model.AuditLog{
		ID: "old-key-row-1", UserID: "u-1", Username: "legacy", TenantID: "t-1",
		Module: "user", Action: "create", Result: "success", StatusCode: 200,
		CreatedAt: createdAt,
	}
	// 用旧 pepper 对同一领域模型算摘要，再把相同字段值落 PO（模拟轮换前的历史数据）。
	row := &auditrepo.AuditLogPO{
		ID: legacy.ID, UserID: legacy.UserID, Username: legacy.Username, TenantID: legacy.TenantID,
		Module: legacy.Module, Action: legacy.Action, Result: legacy.Result,
		StatusCode: legacy.StatusCode, CreatedAt: legacy.CreatedAt,
		RecordHash: oldGuard.Compute(legacy),
	}
	require.NotEmpty(t, row.RecordHash)
	require.NoError(t, app.DB.Create(row).Error)

	// 非超管无 audit:integrity:rehash 权限 → 403
	w := app.Do(t, http.MethodPost, "/api/v1/audit/integrity/rehash", "", member)
	require.Equal(t, http.StatusForbidden, w.Code, "非超管触发 rehash 应 403")

	// 轮换态校验：旧 key 行判为待重算，非篡改
	w = app.Do(t, http.MethodPost, "/api/v1/audit/integrity/verify", "", sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var pre verifyResp
	testkit.DecodeData(t, w, &pre)
	require.Equal(t, "pending_rehash", pre.Verdict, "旧 key 历史行应判为待重算")
	require.Positive(t, pre.PendingRehashRows)
	require.Zero(t, pre.TamperedRows)

	// 执行 rehash
	w = app.Do(t, http.MethodPost, "/api/v1/audit/integrity/rehash", "", sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var rr rehashResp
	testkit.DecodeData(t, w, &rr)
	require.Positive(t, rr.RehashedRows, "rehash 应重算旧 key 行")

	// 迁移后再校验：回到 verified
	w = app.Do(t, http.MethodPost, "/api/v1/audit/integrity/verify", "", sa)
	require.Equal(t, http.StatusOK, w.Code)
	var post verifyResp
	testkit.DecodeData(t, w, &post)
	require.Equal(t, "verified", post.Verdict, "rehash 后应 verified")
	require.Zero(t, post.TamperedRows)
	require.Zero(t, post.PendingRehashRows)
}

// TestInteg_AuditIntegrity_RehashRequiresPrevious 非轮换期（未配旧 pepper）时 rehash 返回 400。
func TestInteg_AuditIntegrity_RehashRequiresPrevious(t *testing.T) {
	app := testkit.NewTestAppOpt(t, func(cfg *config.Config) {
		cfg.Audit = config.AuditConfig{Integrity: config.AuditIntegrityConfig{
			Enabled: true, SecretKey: "pepper-ONLY",
		}}
	})
	sa := app.SuperadminHeader("sa", "t-1")
	w := app.Do(t, http.MethodPost, "/api/v1/audit/integrity/rehash", "", sa)
	require.Equal(t, http.StatusBadRequest, w.Code, "未配旧 pepper 时 rehash 应 400")
}
