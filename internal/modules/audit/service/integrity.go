package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"time"

	"meteorx/internal/config"
	"meteorx/internal/modules/audit/model"
)

// IntegrityGuard 计算审计日志的行级完整性摘要（防篡改）。
// 用 HMAC-SHA256(pepper, canonical(log)) 覆盖固定字段集：只要任一被覆盖字段被篡改，
// 重算摘要即与落库值不符。pepper（SecretKey）不落库；未启用或未配密钥时 enabled=false，
// Compute 返回空串——既不写摘要也不阻断写入，保证审计可用性优先。
//
// 同一次进程内，router 侧与 audit 模块侧两个 AuditService 实例必须共享同一 IntegrityGuard，
// 否则两条写入路径算出的摘要不一致，校验必坏（见 bootstrap 装配与 audit.NewIntegrityGuard）。
type IntegrityGuard struct {
	key     []byte
	enabled bool
}

// NewIntegrityGuard 依配置构造完整性摘要器；未启用或缺 pepper 时返回禁用实例（不 panic、不阻断）。
func NewIntegrityGuard(cfg config.Config) *IntegrityGuard {
	ic := cfg.Audit.Integrity
	if !ic.HasIntegrityKey() {
		return &IntegrityGuard{enabled: false}
	}
	return &IntegrityGuard{
		key:     []byte(strings.TrimSpace(ic.SecretKey)),
		enabled: true,
	}
}

// Enabled 摘要器是否处于生效状态。
func (g *IntegrityGuard) Enabled() bool {
	return g != nil && g.enabled
}

// Compute 返回日志行摘要的 hex 字符串；禁用时返回空串（不写、不阻断）。
func (g *IntegrityGuard) Compute(log *model.AuditLog) string {
	if !g.Enabled() || log == nil {
		return ""
	}
	mac := hmac.New(sha256.New, g.key)
	mac.Write(canonical(log))
	return hex.EncodeToString(mac.Sum(nil))
}

// ComputeDigest 对任意字节序列做 keyed HMAC-SHA256（锚点窗口折叠与锚点哈希复用同一 pepper）。
// 禁用时返回空串。
func (g *IntegrityGuard) ComputeDigest(data []byte) string {
	if !g.Enabled() {
		return ""
	}
	mac := hmac.New(sha256.New, g.key)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// canonical 将日志的固定字段集按固定顺序做稳定序列化（长度前缀消除分隔符歧义），
// 含 CreatedAt、排除 RecordHash 自身。字段集或顺序一旦变更即与历史摘要不兼容，故须谨慎维护。
func canonical(log *model.AuditLog) []byte {
	fields := []string{
		log.ID,
		log.UserID,
		log.Username,
		log.TenantID,
		log.Module,
		log.Action,
		log.Resource,
		log.ResourceID,
		log.Method,
		log.Path,
		log.RequestBody,
		log.ResponseBody,
		log.Result,
		log.ErrorMessage,
		log.ClientIP,
		log.IPLocation,
		log.UserAgent,
		log.DeviceInfo,
		log.SessionID,
		log.RequestID,
		log.TraceID,
		log.Referer,
		log.RiskLevel,
		log.Tags,
		log.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z"),
	}
	// 定长缓冲：8 字节长度前缀 + 内容，逐字段拼接。
	var buf []byte
	var lenBuf [8]byte
	for _, f := range fields {
		binary.BigEndian.PutUint64(lenBuf[:], uint64(len(f)))
		buf = append(buf, lenBuf[:]...)
		buf = append(buf, f...)
	}
	// StatusCode / Duration 为数值字段，同样纳入摘要（拼在字符串域之后，前缀区分）。
	binary.BigEndian.PutUint64(lenBuf[:], uint64(log.StatusCode))
	buf = append(buf, lenBuf[:]...)
	binary.BigEndian.PutUint64(lenBuf[:], uint64(log.Duration))
	buf = append(buf, lenBuf[:]...)
	return buf
}

// foldTime 统一锚点时间字段的序列化格式，保证 job 与 verify 重算口径一致。
func foldTime(t time.Time) string {
	if t.IsZero() {
		return "genesis"
	}
	return t.UTC().Format("2006-01-02T15:04:05.999999999Z")
}

// FoldRecordHashes 将窗口内按 (created_at, id) 有序的行摘要逐行累加折叠为窗口累加摘要。
// 采用链式 HMAC：d_0 = HMAC(key, "")，d_i = HMAC(key, d_{i-1} || hash_i)（长度前缀消除歧义）。
// 因顺序与条数均敏感，窗口内增删改或重序都会使重算摘要与落库值不符。禁用时返回空串。
func (g *IntegrityGuard) FoldRecordHashes(hashes []string) string {
	if !g.Enabled() {
		return ""
	}
	state := ""
	var l [8]byte
	for _, h := range hashes {
		buf := make([]byte, 0, 16+len(state)+len(h))
		binary.BigEndian.PutUint64(l[:], uint64(len(state)))
		buf = append(buf, l[:]...)
		buf = append(buf, state...)
		binary.BigEndian.PutUint64(l[:], uint64(len(h)))
		buf = append(buf, l[:]...)
		buf = append(buf, h...)
		state = g.ComputeDigest(buf)
	}
	return state
}

// AnchorHashOf 计算锚点哈希，把上一条锚点、本窗口摘要、行数与窗口边界链入，
// 使删改/插删窗口或断开链都会破坏后续所有锚点的 AnchorHash。禁用时返回空串。
func (g *IntegrityGuard) AnchorHashOf(prevAnchorHash, windowDigest string, rowCount int64, start, end time.Time) string {
	if !g.Enabled() {
		return ""
	}
	var l [8]byte
	buf := make([]byte, 0, 160)
	writeStr := func(s string) {
		binary.BigEndian.PutUint64(l[:], uint64(len(s)))
		buf = append(buf, l[:]...)
		buf = append(buf, s...)
	}
	writeStr(prevAnchorHash)
	writeStr(windowDigest)
	binary.BigEndian.PutUint64(l[:], uint64(rowCount))
	buf = append(buf, l[:]...)
	writeStr(foldTime(start))
	writeStr(foldTime(end))
	return g.ComputeDigest(buf)
}
