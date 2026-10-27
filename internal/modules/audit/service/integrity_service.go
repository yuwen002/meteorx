package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/repository"
	"meteorx/pkg/idgen"
)

// 完整性校验结论。
const (
	IntegrityVerdictVerified      = "verified"       // 全部通过
	IntegrityVerdictTampered      = "tampered"       // 检出篡改/删改/断链（两把 key 都对不上）
	IntegrityVerdictNotConfigured = "not_configured" // 未配置 pepper，无法校验
	IntegrityVerdictPendingRehash = "pending_rehash" // 存在仅匹配旧 pepper 的行/锚点，待密钥重算（非篡改）
)

// verifySampleLimit 结果中各类样本的最大返回条数，避免整表异常时响应过大。
const verifySampleLimit = 20

// IntegrityService 审计完整性校验服务：只读重放行摘要与锚点链，不做任何写入。
// 与写入侧共用同一 IntegrityGuard（同一 pepper），据以重算并比对。
type IntegrityService struct {
	guard *IntegrityGuard
	irepo repository.IntegrityRepository
}

// NewIntegrityService 创建完整性校验服务。
func NewIntegrityService(guard *IntegrityGuard, irepo repository.IntegrityRepository) *IntegrityService {
	return &IntegrityService{guard: guard, irepo: irepo}
}

// Verify 执行完整性校验：
// (1) 逐行重算 record_hash 与存储比对（RecordHash 为空记为 legacy，不计失败）；
// (2) 依锚点重建各窗口 digest 与 AnchorHash 链并比对，检出篡改/删行/插行/重序/断链。
func (s *IntegrityService) Verify(ctx context.Context) (*dto.IntegrityVerifyResp, error) {
	resp := &dto.IntegrityVerifyResp{
		Enabled:              s.guard.Enabled(),
		TamperedSamples:      []string{},
		PendingRehashSamples: []string{},
		AnomalyAnchorIDs:     []string{},
		Anomalies:            []string{},
	}
	if !s.guard.Enabled() {
		resp.Verdict = IntegrityVerdictNotConfigured
		resp.Message = "审计完整性校验未启用（未配置 pepper），无法校验"
		return resp, nil
	}

	// (1) 行级重算：全表按 (created_at, id) 有序读取整行，逐行分类比对存储摘要。
	// 处于密钥轮换过渡期时，仅匹配旧 pepper 的行判为“待重算”而非篡改；两把 key 都不匹配才计篡改。
	logs, err := s.irepo.ListLogsInRange(ctx, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	for _, l := range logs {
		resp.CheckedRows++
		switch s.guard.ClassifyRow(l) {
		case RowLegacy:
			resp.LegacyRows++
		case RowCurrent:
			// 匹配当前 key，通过
		case RowPrevious:
			resp.PendingRehashRows++
			if len(resp.PendingRehashSamples) < verifySampleLimit {
				resp.PendingRehashSamples = append(resp.PendingRehashSamples, l.ID)
			}
		case RowNone:
			resp.TamperedRows++
			if len(resp.TamperedSamples) < verifySampleLimit {
				resp.TamperedSamples = append(resp.TamperedSamples, l.ID)
			}
		}
	}

	// (2) 锚点链重放：按窗口升序，先验链衔接，再重算窗口摘要与锚点哈希。
	// 每条锚点先用当前 key 验，不匹配再用旧 key 复验（轮换前锚点）；旧 key 命中记为待重算，两把都不命中才判异常。
	anchors, err := s.irepo.ListAnchors(ctx)
	if err != nil {
		return nil, err
	}
	chainPrev := ""
	firstSeen := false
	hadStaleAnchor := false
	for _, a := range anchors {
		resp.AnchorsChecked++

		// 链衔接：本条 PrevAnchorHash 必须等于上一条 AnchorHash（首条须为空）。链衔接与生成 key 无关，
		// 比较的是落库字符串，故轮换过渡期整链仍为旧代、内部自洽。
		if a.PrevAnchorHash != chainPrev {
			s.addAnomaly(resp, a.ID, "锚点链断裂：PrevAnchorHash 与上一条 AnchorHash 不衔接")
		}

		// 窗口重放：按 (created_at,id) 读取窗口内存储的行摘要，重折叠比对 WindowDigest 与行数
		wrows, err := s.irepo.ListLogsInRange(ctx, a.WindowStart, a.WindowEnd)
		if err != nil {
			return nil, err
		}
		hashes := make([]string, len(wrows))
		for i, w := range wrows {
			hashes[i] = w.RecordHash
		}
		if int64(len(wrows)) != a.RowCount {
			s.addAnomaly(resp, a.ID, "窗口行数与锚点记录不符（存在删行/插行）")
		}
		switch {
		case s.guard.FoldRecordHashes(hashes) == a.WindowDigest:
			// 当前 key 命中：进一步验锚点行自证哈希
			if recomputed := s.guard.AnchorHashOf(a.PrevAnchorHash, a.WindowDigest, a.RowCount, a.WindowStart, a.WindowEnd); recomputed != a.AnchorHash {
				s.addAnomaly(resp, a.ID, "锚点哈希不符（锚点行被篡改）")
			}
		case s.guard.HasPrevious() && s.guard.FoldRecordHashesWithPrevious(hashes) == a.WindowDigest:
			// 旧 key 命中：轮换前锚点，待 rehash 全量重建，不计篡改
			hadStaleAnchor = true
		default:
			s.addAnomaly(resp, a.ID, "窗口累加摘要不符（内容被增删改或重序）")
		}

		chainPrev = a.AnchorHash
		if !firstSeen {
			resp.CoveredFrom = a.WindowStart.UTC().Format(time.RFC3339)
			firstSeen = true
		}
		resp.CoveredTo = a.WindowEnd.UTC().Format(time.RFC3339)
	}

	switch {
	case resp.TamperedRows > 0 || len(resp.AnomalyAnchorIDs) > 0:
		resp.Verdict = IntegrityVerdictTampered
		resp.Message = "检测到完整性异常：篡改行 " + strconv.FormatInt(resp.TamperedRows, 10) + "，异常锚点 " + strconv.Itoa(len(resp.AnomalyAnchorIDs))
	case resp.PendingRehashRows > 0 || hadStaleAnchor:
		resp.Verdict = IntegrityVerdictPendingRehash
		resp.Message = "检测到 " + strconv.FormatInt(resp.PendingRehashRows, 10) + " 行待密钥重算（旧 pepper），可执行 rehash 迁移到新密钥"
	default:
		resp.Verdict = IntegrityVerdictVerified
		resp.Message = "完整性校验通过：" + strconv.FormatInt(resp.CheckedRows, 10) + " 行、" + strconv.FormatInt(resp.AnchorsChecked, 10) + " 个锚点均一致"
	}
	return resp, nil
}

// addAnomaly 记录一条异常（锚点ID 去重收集 + 描述样本限量）。
func (s *IntegrityService) addAnomaly(resp *dto.IntegrityVerifyResp, anchorID, msg string) {
	exists := false
	for _, id := range resp.AnomalyAnchorIDs {
		if id == anchorID {
			exists = true
			break
		}
	}
	if !exists {
		resp.AnomalyAnchorIDs = append(resp.AnomalyAnchorIDs, anchorID)
	}
	if len(resp.Anomalies) < verifySampleLimit {
		resp.Anomalies = append(resp.Anomalies, anchorID+": "+msg)
	}
}

// rehash 前置错误（handler 映射 400）。
var (
	ErrIntegrityNotEnabled    = errors.New("审计完整性校验未启用")
	ErrIntegrityNoPreviousKey = errors.New("未配置旧 pepper，无待迁移密钥")
)

// Rehash 密钥轮换迁移：把仅匹配旧 pepper 的历史行重算到当前 pepper，随后全量重建锚点链。
// 只重写 ClassifyRow==RowPrevious（旧 key 可验证）的行——两把 key 都对不上的真篡改行不会被洗白。
// 因行摘要值改变会使旧锚点必然失效，故先清空旧锚点再按 anchor_interval 时间桶从创世重建。
// 本方法不做并发锁与自审（由上层 coordinator 编排）。
func (s *IntegrityService) Rehash(ctx context.Context) (*dto.IntegrityRehashResp, error) {
	if !s.guard.Enabled() {
		return nil, ErrIntegrityNotEnabled
	}
	if !s.guard.HasPrevious() {
		return nil, ErrIntegrityNoPreviousKey
	}

	// 1) 读全量行，收集仅匹配旧 key 的行，用当前 key 重算后批量回写
	logs, err := s.irepo.ListLogsInRange(ctx, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	updates := make([]repository.RecordHashUpdate, 0)
	for _, l := range logs {
		if s.guard.ClassifyRow(l) == RowPrevious {
			updates = append(updates, repository.RecordHashUpdate{ID: l.ID, RecordHash: s.guard.Compute(l)})
		}
	}
	if err := s.irepo.BatchUpdateRecordHash(ctx, updates); err != nil {
		return nil, err
	}

	// 2) 重建锚点链：按当前 key 从创世重新折叠全量历史行，并在单事务内原子替换旧链（删旧+建新不可分割）。
	rebuilt, err := s.rebuildAnchors(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.IntegrityRehashResp{
		RehashedRows:   int64(len(updates)),
		AnchorsRebuilt: rebuilt,
		Message: "已重算 " + strconv.FormatInt(int64(len(updates)), 10) +
			" 行到新密钥，重建 " + strconv.FormatInt(rebuilt, 10) + " 个锚点",
	}, nil
}

// rebuildAnchors 按 (created_at, id) 有序读取全量行（已为新 key 摘要），以 anchor_interval 为时间桶
// 从创世链式生成整条新锚点链，最后通过 ReplaceAnchors 单事务原子替换旧链（删旧+建新不可分割）。
// 窗口右端一律不越过安全界 sealLimit=now-grace（与周期任务同口径）：一方面避免重建出以未来时刻
// 收尾的窗口（否则其后新写入的行会落进已封窗口内，令校验重放时窗口内容漂移而误报），另一方面使
// sealLimit 之后的新行留给周期任务下次封口（它们仍以当前 key 写入，行级校验照常通过）。
// 空表时清空锚点。返回落库锚点条数。
func (s *IntegrityService) rebuildAnchors(ctx context.Context) (int64, error) {
	logs, err := s.irepo.ListLogsInRange(ctx, time.Time{}, time.Time{})
	if err != nil {
		return 0, err
	}
	sealLimit := time.Now().Add(-s.guard.AnchorGrace())
	interval := s.guard.AnchorInterval()

	var (
		anchors  []*repository.IntegrityAnchor
		prevHash string
	)
	for i := 0; i < len(logs); {
		winStart := logs[i].CreatedAt
		if !winStart.Before(sealLimit) {
			break // 该窗起始已在安全界之后，尾部交给周期任务封口
		}
		winEnd := winStart.Add(interval)
		if winEnd.After(sealLimit) {
			winEnd = sealLimit // 末窗 clamp：右端不越过安全界进入未来
		}
		var hashes []string
		j := i
		for j < len(logs) && logs[j].CreatedAt.Before(winEnd) {
			hashes = append(hashes, logs[j].RecordHash)
			j++
		}
		windowDigest := s.guard.FoldRecordHashes(hashes)
		anchor := &repository.IntegrityAnchor{
			ID:             idgen.New(),
			WindowStart:    winStart,
			WindowEnd:      winEnd,
			RowCount:       int64(len(hashes)),
			WindowDigest:   windowDigest,
			PrevAnchorHash: prevHash,
		}
		anchor.AnchorHash = s.guard.AnchorHashOf(prevHash, windowDigest, anchor.RowCount, winStart, winEnd)
		anchors = append(anchors, anchor)
		prevHash = anchor.AnchorHash
		i = j
	}

	if err := s.irepo.ReplaceAnchors(ctx, anchors); err != nil {
		return 0, err
	}
	return int64(len(anchors)), nil
}
