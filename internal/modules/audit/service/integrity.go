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

// anchorGraceDefault 锚点封口的默认安全边界：窗口右端取 now-此值，规避刚写入、created_at 落在
// 窗口边缘的迟到行被本次封口漏掉后又落进已封窗口（否则校验会因窗口内容漂移而误报）。
const anchorGraceDefault = time.Minute

// IntegrityGuard 计算审计日志的行级完整性摘要（防篡改）。
// 用 HMAC-SHA256(pepper, canonical(log)) 覆盖固定字段集：只要任一被覆盖字段被篡改，
// 重算摘要即与落库值不符。pepper（SecretKey）不落库；未启用或未配密钥时 enabled=false，
// Compute 返回空串——既不写摘要也不阻断写入，保证审计可用性优先。
//
// 同一次进程内，router 侧与 audit 模块侧两个 AuditService 实例必须共享同一 IntegrityGuard，
// 否则两条写入路径算出的摘要不一致，校验必坏（见 bootstrap 装配与 audit.NewIntegrityGuard）。
type IntegrityGuard struct {
	key      []byte
	prevKey  []byte // 轮换过渡期的旧 pepper；nil/空表示未处于轮换期
	interval time.Duration
	grace    time.Duration
	enabled  bool
}

// NewIntegrityGuard 依配置构造完整性摘要器；未启用或缺 pepper 时返回禁用实例（不 panic、不阻断）。
// 处于密钥轮换过渡期（额外配置 previous_secret_key）时一并载入旧 key，供校验复验与 rehash 识别使用。
func NewIntegrityGuard(cfg config.Config) *IntegrityGuard {
	ic := cfg.Audit.Integrity
	if !ic.HasIntegrityKey() {
		return &IntegrityGuard{enabled: false}
	}
	g := &IntegrityGuard{
		key:      []byte(strings.TrimSpace(ic.SecretKey)),
		interval: ic.GetAnchorInterval(),
		grace:    anchorGraceDefault,
		enabled:  true,
	}
	if ic.HasPreviousKey() {
		g.prevKey = []byte(strings.TrimSpace(ic.PreviousSecretKey))
	}
	return g
}

// AnchorInterval 锚点链时间桶宽度，供 rehash 全量重建时按同一口径切窗。
func (g *IntegrityGuard) AnchorInterval() time.Duration {
	if g == nil || g.interval <= 0 {
		return time.Hour
	}
	return g.interval
}

// AnchorGrace 封口安全边界：窗口右端取 now-grace，给异步批量 flush 的迟到行留出落库时间。
// 周期锚点任务与 rehash 全量重建共用同一口径，避免两处对「可封窗口右端」的理解漂移。
func (g *IntegrityGuard) AnchorGrace() time.Duration {
	if g == nil || g.grace < 0 {
		return anchorGraceDefault
	}
	return g.grace
}

// Enabled 摘要器是否处于生效状态。
func (g *IntegrityGuard) Enabled() bool {
	return g != nil && g.enabled
}

// HasPrevious 是否处于密钥轮换过渡期：完整性生效且载入了非空旧 pepper。
func (g *IntegrityGuard) HasPrevious() bool {
	return g != nil && g.enabled && len(g.prevKey) > 0
}

// Compute 返回日志行摘要的 hex 字符串（当前 key）；禁用时返回空串（不写、不阻断）。
func (g *IntegrityGuard) Compute(log *model.AuditLog) string {
	if !g.Enabled() || log == nil {
		return ""
	}
	return computeWith(g.key, log)
}

// ComputeWithPrevious 用旧 pepper 重算行摘要；无旧 key（非轮换期）或禁用时返回空串。
func (g *IntegrityGuard) ComputeWithPrevious(log *model.AuditLog) string {
	if !g.HasPrevious() || log == nil {
		return ""
	}
	return computeWith(g.prevKey, log)
}

// RowKeyStatus 行摘要与新旧两把 key 的匹配结论，供校验分类与 rehash 识别。
type RowKeyStatus int

const (
	RowLegacy   RowKeyStatus = iota // RecordHash 为空：legacy/未启用，不参与校验
	RowCurrent                      // 匹配当前 key：已验证
	RowPrevious                     // 仅匹配旧 key：待 rehash（合法历史行，非篡改）
	RowNone                         // 两把 key 都不匹配：真篡改
)

// ClassifyRow 判定一行摘要归属于哪把 key：空=Legacy；命中当前=Current；仅命中旧=Previous；否则 None。
// 禁用时一律按 Legacy 处理（无摘要体系）。
func (g *IntegrityGuard) ClassifyRow(log *model.AuditLog) RowKeyStatus {
	if !g.Enabled() || log == nil || log.RecordHash == "" {
		return RowLegacy
	}
	if computeWith(g.key, log) == log.RecordHash {
		return RowCurrent
	}
	if g.HasPrevious() && computeWith(g.prevKey, log) == log.RecordHash {
		return RowPrevious
	}
	return RowNone
}

// ComputeDigest 对任意字节序列做 keyed HMAC-SHA256（锚点窗口折叠与锚点哈希复用同一 pepper）。
// 禁用时返回空串。
func (g *IntegrityGuard) ComputeDigest(data []byte) string {
	if !g.Enabled() {
		return ""
	}
	return digestWith(g.key, data)
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

// FoldRecordHashes 将窗口内按 (created_at, id) 有序的行摘要逐行累加折叠为窗口累加摘要（当前 key）。
// 采用链式 HMAC：d_0 = HMAC(key, "")，d_i = HMAC(key, d_{i-1} || hash_i)（长度前缀消除歧义）。
// 因顺序与条数均敏感，窗口内增删改或重序都会使重算摘要与落库值不符。禁用时返回空串。
func (g *IntegrityGuard) FoldRecordHashes(hashes []string) string {
	if !g.Enabled() {
		return ""
	}
	return foldWith(g.key, hashes)
}

// FoldRecordHashesWithPrevious 用旧 pepper 折叠窗口行摘要；无旧 key（非轮换期）或禁用时返回空串。
func (g *IntegrityGuard) FoldRecordHashesWithPrevious(hashes []string) string {
	if !g.HasPrevious() {
		return ""
	}
	return foldWith(g.prevKey, hashes)
}

// foldWith 以给定 key 执行链式折叠（current/previous 共用同一算法）。
func foldWith(key []byte, hashes []string) string {
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
		state = digestWith(key, buf)
	}
	return state
}

// AnchorHashOf 计算锚点哈希（当前 key），把上一条锚点、本窗口摘要、行数与窗口边界链入，
// 使删改/插删窗口或断开链都会破坏后续所有锚点的 AnchorHash。禁用时返回空串。
func (g *IntegrityGuard) AnchorHashOf(prevAnchorHash, windowDigest string, rowCount int64, start, end time.Time) string {
	if !g.Enabled() {
		return ""
	}
	return anchorWith(g.key, prevAnchorHash, windowDigest, rowCount, start, end)
}

// AnchorHashOfWithPrevious 用旧 pepper 计算锚点哈希；无旧 key（非轮换期）或禁用时返回空串。
func (g *IntegrityGuard) AnchorHashOfWithPrevious(prevAnchorHash, windowDigest string, rowCount int64, start, end time.Time) string {
	if !g.HasPrevious() {
		return ""
	}
	return anchorWith(g.prevKey, prevAnchorHash, windowDigest, rowCount, start, end)
}

// anchorWith 以给定 key 计算锚点哈希（current/previous 共用同一序列化口径）。
func anchorWith(key []byte, prevAnchorHash, windowDigest string, rowCount int64, start, end time.Time) string {
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
	return digestWith(key, buf)
}

// computeWith 以给定 key 对日志行计算 HMAC-SHA256 hex（canonical 口径与历史一致）。
func computeWith(key []byte, log *model.AuditLog) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(canonical(log))
	return hex.EncodeToString(mac.Sum(nil))
}

// digestWith 以给定 key 对任意字节序列做 keyed HMAC-SHA256 hex。
func digestWith(key, data []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}
