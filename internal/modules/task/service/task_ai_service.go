// Package service 实现任务模块的 AI 辅助能力。
// 作为可选叠加层：依赖通用 ai.Provider 完成自然语言解析、字段建议、描述扩写与任务摘要。
// 未启用/未配置凭据或上游失败时返回明确错误，由 handler 降级为 503/502，绝不写库、不影响主流程。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"meteorx/internal/ai"
	"meteorx/internal/modules/task/dto"
	"meteorx/internal/modules/task/model"
	"meteorx/internal/modules/task/repository"
)

// 业务错误定义
var (
	// ErrAIUnavailable AI 能力未启用或未配置凭据，调用方应降级而非报系统故障
	ErrAIUnavailable = errors.New("ai capability unavailable")
	// ErrAIRuntime 上游调用失败或返回不可解析内容
	ErrAIRuntime = errors.New("ai runtime error")
)

// aiProvider 服务层依赖的 AI 能力抽象：既能发起补全，又能探测可用性与默认模型。
// 由 *ai.Client 实现；测试可注入 stub。
type aiProvider interface {
	Chat(ctx context.Context, req ai.ChatRequest) (ai.ChatResponse, error)
	Usable() bool
	Enabled() bool
	Model() string
}

// TaskAIService 任务 AI 辅助服务。
type TaskAIService struct {
	provider aiProvider
	repo     repository.TaskRepository // 摘要能力需读取当前用户任务，可为 nil（此时摘要返回不可用）
}

// NewTaskAIService 创建任务 AI 辅助服务实例。
func NewTaskAIService(provider aiProvider, repo repository.TaskRepository) *TaskAIService {
	return &TaskAIService{provider: provider, repo: repo}
}

// Capabilities 返回 AI 能力探测结果，供前端门控按钮显隐。
// Enabled 表示"可用"（开关开启且已配置凭据）；摘要口径额外要求仓储可用。
func (s *TaskAIService) Capabilities() dto.AICapabilityResp {
	return dto.AICapabilityResp{
		Enabled: s.provider.Usable(),
		Model:   s.provider.Model(),
	}
}

// ParseDraft 将自然语言解析为任务草稿。now 由调用方注入（便于测试与业务时区一致），
// 用于让模型正确推算"下周五/三天后"等相对日期。
func (s *TaskAIService) ParseDraft(ctx context.Context, text string, now time.Time) (*dto.TaskDraftResp, error) {
	sys := "你是任务解析助手。将用户的自然语言拆解为结构化任务字段，仅返回 JSON。" +
		fmt.Sprintf("当前时间：%s（%s）。", now.Format("2006-01-02 15:04:05"), "星期"+weekdayCN(now.Weekday())) +
		"字段：title(字符串)、priority(low|normal|high|urgent 之一)、due_date(相对时间换算为 \"2006-01-02 15:04:05\"，无法确定则留空)、" +
		"tags(字符串数组，至多 5 个)、recurrence(daily|weekly|monthly 之一，未提及留空)。"
	content, err := s.completeJSON(ctx, sys, text)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Title      string   `json:"title"`
		Priority   string   `json:"priority"`
		DueDate    string   `json:"due_date"`
		Tags       []string `json:"tags"`
		Recurrence string   `json:"recurrence"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIRuntime, err)
	}
	draft := &dto.TaskDraftResp{
		Title:      strings.TrimSpace(raw.Title),
		Priority:   normalizePriority(raw.Priority),
		Tags:       cleanStrings(raw.Tags),
		Recurrence: normalizeRecurrence(raw.Recurrence),
	}
	// 归一化截止时间：可解析则回标准格式，否则丢弃（前端可手动补）
	if d := strings.TrimSpace(raw.DueDate); d != "" {
		if t, perr := ParseTaskDate(d); perr == nil && t != nil {
			draft.DueDate = t.Format("2006-01-02 15:04:05")
		}
	}
	return draft, nil
}

// Suggest 依据标题/描述给出优先级与标签建议。
func (s *TaskAIService) Suggest(ctx context.Context, req dto.SuggestReq) (*dto.SuggestResp, error) {
	sys := "你是任务管理助手。根据任务标题与描述建议优先级和标签，仅返回 JSON。" +
		"字段：priority(low|normal|high|urgent 之一)、tags(字符串数组，至多 5 个)、reason(不超过 40 字的简短中文理由)。"
	content, err := s.completeJSON(ctx, sys, fmt.Sprintf("标题：%s\n描述：%s", req.Title, req.Description))
	if err != nil {
		return nil, err
	}
	var raw struct {
		Priority string   `json:"priority"`
		Tags     []string `json:"tags"`
		Reason   string   `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIRuntime, err)
	}
	return &dto.SuggestResp{
		Priority: normalizePriority(raw.Priority),
		Tags:     cleanStrings(raw.Tags),
		Reason:   strings.TrimSpace(raw.Reason),
	}, nil
}

// Expand 依据标题/描述扩写更完整的描述并拆解为子步骤。
func (s *TaskAIService) Expand(ctx context.Context, req dto.ExpandReq) (*dto.ExpandResp, error) {
	sys := "你是任务详情助手。根据标题与简述扩写出条理清晰的完整描述，并拆解为可执行的子步骤，仅返回 JSON。" +
		"字段：description(扩写后的中文描述)、steps(字符串数组，3~7 条，无法拆解则空数组)。"
	content, err := s.completeJSON(ctx, sys, fmt.Sprintf("标题：%s\n简述：%s", req.Title, req.Description))
	if err != nil {
		return nil, err
	}
	var raw struct {
		Description string   `json:"description"`
		Steps       []string `json:"steps"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIRuntime, err)
	}
	steps := make([]string, 0, len(raw.Steps))
	for _, st := range raw.Steps {
		if st = strings.TrimSpace(st); st != "" {
			steps = append(steps, st)
		}
	}
	return &dto.ExpandResp{Description: strings.TrimSpace(raw.Description), Steps: steps}, nil
}

// Summarize 读取当前用户在指定范围内的任务，产出中文摘要与优先处理建议。
func (s *TaskAIService) Summarize(ctx context.Context, tenantID, userID, scope string) (*dto.SummaryResp, error) {
	if s.repo == nil {
		return nil, ErrAIUnavailable
	}
	if !s.provider.Usable() {
		return nil, ErrAIUnavailable
	}
	if scope == "" {
		scope = "all"
	}
	tasks, total, err := s.collectTasks(ctx, tenantID, userID, scope)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIRuntime, err)
	}
	if total == 0 {
		return &dto.SummaryResp{Text: "当前范围内暂无任务。", Highlights: []string{}, SuggestedFocus: []string{}, Total: 0}, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "共 %d 条任务，请据此生成中文摘要与优先处理建议：\n", total)
	for i, t := range tasks {
		due := ""
		if t.DueDate != nil {
			due = t.DueDate.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(&b, "%d. [%s|%s] %s；状态=%s；负责人=%s；截止=%s；标签=%s\n",
			i+1, t.Priority, t.Status, t.Title, t.Status, t.AssigneeName, orDash(due), strings.Join(t.Tags, ","))
	}

	sys := "你是任务分析助手。根据用户任务清单生成简明中文摘要，仅返回 JSON。" +
		"字段：text(2-4 句整体概览)、highlights(字符串数组，关键要点至多 5 条)、suggested_focus(字符串数组，建议优先处理的任务标题，至多 5 条)。"
	content, err := s.completeJSON(ctx, sys, b.String())
	if err != nil {
		return nil, err
	}
	var raw struct {
		Text           string   `json:"text"`
		Highlights     []string `json:"highlights"`
		SuggestedFocus []string `json:"suggested_focus"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIRuntime, err)
	}
	return &dto.SummaryResp{
		Text:           strings.TrimSpace(raw.Text),
		Highlights:     cleanStrings(raw.Highlights),
		SuggestedFocus: cleanStrings(raw.SuggestedFocus),
		Total:          int(total),
	}, nil
}

// collectTasks 按摘要范围拉取当前用户可见任务（上限 100 条喂给模型）。
func (s *TaskAIService) collectTasks(ctx context.Context, tenantID, userID, scope string) ([]*model.Task, int64, error) {
	now := time.Now()
	f := repository.ListFilter{TenantID: tenantID, UserID: userID, Page: 1, PageSize: 100}
	switch scope {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		f.DueFrom, f.DueTo = &start, &end
	case "week":
		end := now.AddDate(0, 0, 7)
		f.DueFrom, f.DueTo = &now, &end
	case "overdue":
		f.Overdue = true
	default: // all：不限日期，排除已完成，聚焦待办
		f.Status = ""
	}
	return s.repo.List(ctx, f)
}

// completeJSON 通用一次 JSON 补全：不可用时降级为 ErrAIUnavailable，其余失败包装为 ErrAIRuntime。
func (s *TaskAIService) completeJSON(ctx context.Context, sys, user string) (string, error) {
	if !s.provider.Usable() {
		return "", ErrAIUnavailable
	}
	resp, err := s.provider.Chat(ctx, ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: sys},
			{Role: "user", Content: user},
		},
		JSONMode: true,
	})
	if err != nil {
		if errors.Is(err, ai.ErrDisabled) {
			return "", ErrAIUnavailable
		}
		return "", fmt.Errorf("%w: %v", ErrAIRuntime, err)
	}
	return resp.Content, nil
}

// normalizePriority 归一化优先级，非法值回落 normal。
func normalizePriority(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	if model.IsValidPriority(p) {
		return p
	}
	return model.TaskPriorityNormal
}

// normalizeRecurrence 归一化重复周期，非法值回落空（不重复）。
func normalizeRecurrence(r string) string {
	r = strings.ToLower(strings.TrimSpace(r))
	if model.IsValidRecurrence(r) {
		return r
	}
	return ""
}

// cleanStrings 去除空白项，保证返回非 nil 切片（便于前端 JSON 序列化为 []）。
func cleanStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// orDash 空串占位为 "-"，便于拼装清单文本。
func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// weekdayCN 将星期转为中文。
func weekdayCN(w time.Weekday) string {
	names := [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return names[int(w)]
}
