package service

import (
	"context"
	"strconv"
	"time"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/repository"
)

// 完整性校验结论。
const (
	IntegrityVerdictVerified      = "verified"       // 全部通过
	IntegrityVerdictTampered      = "tampered"       // 检出篡改/删改/断链
	IntegrityVerdictNotConfigured = "not_configured" // 未配置 pepper，无法校验
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
		Enabled:          s.guard.Enabled(),
		TamperedSamples:  []string{},
		AnomalyAnchorIDs: []string{},
		Anomalies:        []string{},
	}
	if !s.guard.Enabled() {
		resp.Verdict = IntegrityVerdictNotConfigured
		resp.Message = "审计完整性校验未启用（未配置 pepper），无法校验"
		return resp, nil
	}

	// (1) 行级重算：全表按 (created_at, id) 有序读取整行，逐行比对存储摘要。
	logs, err := s.irepo.ListLogsInRange(ctx, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	for _, l := range logs {
		resp.CheckedRows++
		if l.RecordHash == "" {
			resp.LegacyRows++
			continue
		}
		if s.guard.Compute(l) != l.RecordHash {
			resp.TamperedRows++
			if len(resp.TamperedSamples) < verifySampleLimit {
				resp.TamperedSamples = append(resp.TamperedSamples, l.ID)
			}
		}
	}

	// (2) 锚点链重放：按窗口升序，先验链衔接，再重算窗口摘要与锚点哈希。
	anchors, err := s.irepo.ListAnchors(ctx)
	if err != nil {
		return nil, err
	}
	chainPrev := ""
	firstSeen := false
	for _, a := range anchors {
		resp.AnchorsChecked++

		// 链衔接：本条 PrevAnchorHash 必须等于上一条 AnchorHash（首条须为空）
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
		if recomputed := s.guard.FoldRecordHashes(hashes); recomputed != a.WindowDigest {
			s.addAnomaly(resp, a.ID, "窗口累加摘要不符（内容被增删改或重序）")
		}
		// 锚点行自证：用存储的 prev/摘要/行数/边界重算 AnchorHash，检测锚点行本身被改写
		if recomputed := s.guard.AnchorHashOf(a.PrevAnchorHash, a.WindowDigest, a.RowCount, a.WindowStart, a.WindowEnd); recomputed != a.AnchorHash {
			s.addAnomaly(resp, a.ID, "锚点哈希不符（锚点行被篡改）")
		}

		chainPrev = a.AnchorHash
		if !firstSeen {
			resp.CoveredFrom = a.WindowStart.UTC().Format(time.RFC3339)
			firstSeen = true
		}
		resp.CoveredTo = a.WindowEnd.UTC().Format(time.RFC3339)
	}

	if resp.TamperedRows > 0 || len(resp.AnomalyAnchorIDs) > 0 {
		resp.Verdict = IntegrityVerdictTampered
		resp.Message = "检测到完整性异常：篡改行 " + strconv.FormatInt(resp.TamperedRows, 10) + "，异常锚点 " + strconv.Itoa(len(resp.AnomalyAnchorIDs))
	} else {
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
