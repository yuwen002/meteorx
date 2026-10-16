package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"meteorx/internal/ai"
	"meteorx/internal/modules/task/dto"
	"meteorx/internal/modules/task/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAI 实现 aiProvider，按预设返回固定内容或错误，便于覆盖各能力分支。
// 复用同包 stubTaskRepo 作为摘要读取的仓储桩。
type stubAI struct {
	usable  bool
	model   string
	content string
	err     error // 非 nil 时 Chat 返回该错误（模拟上游失败）
	calls   int
}

func (s *stubAI) Chat(context.Context, ai.ChatRequest) (ai.ChatResponse, error) {
	s.calls++
	if !s.usable {
		return ai.ChatResponse{}, ai.ErrDisabled
	}
	if s.err != nil {
		return ai.ChatResponse{}, s.err
	}
	return ai.ChatResponse{Content: s.content, Model: s.model}, nil
}

func (s *stubAI) Usable() bool  { return s.usable }
func (s *stubAI) Enabled() bool { return s.usable }
func (s *stubAI) Model() string { return s.model }

func newStubAI(content string) *stubAI {
	return &stubAI{usable: true, model: "gpt-4o-mini", content: content}
}

func TestCapabilities_Usable(t *testing.T) {
	svc := NewTaskAIService(newStubAI(""), nil)
	cap := svc.Capabilities()
	assert.True(t, cap.Enabled)
	assert.Equal(t, "gpt-4o-mini", cap.Model)
}

func TestCapabilities_NotConfigured(t *testing.T) {
	svc := NewTaskAIService(&stubAI{usable: false, model: "gpt-4o-mini"}, nil)
	assert.False(t, svc.Capabilities().Enabled)
}

// ---- ParseDraft ----

func TestParseDraft_OK(t *testing.T) {
	// 固定 now，不随真实时钟翻转；due_date 为模型返回的确定串，仅做格式归一化
	now := time.Date(2026, 3, 1, 9, 0, 0, 0, time.Local)
	content := `{"title":"写周报","priority":"high","due_date":"2026-12-31 10:00:00","tags":["工作"," ","个人"],"recurrence":"weekly"}`
	svc := NewTaskAIService(newStubAI(content), nil)

	draft, err := svc.ParseDraft(context.Background(), "下周五前高优先级写周报，标记工作和个人，每周重复", now)
	require.NoError(t, err)
	assert.Equal(t, "写周报", draft.Title)
	assert.Equal(t, model.TaskPriorityHigh, draft.Priority)
	assert.Equal(t, "2026-12-31 10:00:00", draft.DueDate)
	assert.Equal(t, []string{"工作", "个人"}, draft.Tags)
	assert.Equal(t, model.TaskRecurrenceWeekly, draft.Recurrence)
}

func TestParseDraft_InvalidEnumFallsBack(t *testing.T) {
	now := time.Date(2026, 3, 1, 9, 0, 0, 0, time.Local)
	content := `{"title":"买菜","priority":"ASAP","due_date":"not-a-date","tags":[],"recurrence":"yearly"}`
	svc := NewTaskAIService(newStubAI(content), nil)

	draft, err := svc.ParseDraft(context.Background(), "买菜", now)
	require.NoError(t, err)
	assert.Equal(t, model.TaskPriorityNormal, draft.Priority) // 非法优先级回落 normal
	assert.Empty(t, draft.DueDate)                            // 无法解析的日期被丢弃
	assert.Empty(t, draft.Recurrence)                         // 非法周期回落空
	assert.NotNil(t, draft.Tags)                              // 归一化为非 nil 切片
	assert.Len(t, draft.Tags, 0)
}

func TestParseDraft_BadJSON(t *testing.T) {
	svc := NewTaskAIService(newStubAI("这不是 JSON"), nil)
	_, err := svc.ParseDraft(context.Background(), "x", time.Now())
	assert.ErrorIs(t, err, ErrAIRuntime)
}

func TestParseDraft_Unavailable(t *testing.T) {
	svc := NewTaskAIService(&stubAI{usable: false}, nil)
	_, err := svc.ParseDraft(context.Background(), "x", time.Now())
	assert.ErrorIs(t, err, ErrAIUnavailable)
}

func TestParseDraft_UpstreamError(t *testing.T) {
	svc := NewTaskAIService(&stubAI{usable: true, err: errors.New("boom")}, nil)
	_, err := svc.ParseDraft(context.Background(), "x", time.Now())
	assert.ErrorIs(t, err, ErrAIRuntime)
}

// ---- Suggest ----

func TestSuggest_OK(t *testing.T) {
	content := `{"priority":"urgent","tags":["线上","故障"],"reason":"涉及线上故障需尽快处理"}`
	svc := NewTaskAIService(newStubAI(content), nil)

	resp, err := svc.Suggest(context.Background(), dto.SuggestReq{Title: "修复登录报错", Description: "生产环境登录 500"})
	require.NoError(t, err)
	assert.Equal(t, model.TaskPriorityUrgent, resp.Priority)
	assert.Equal(t, []string{"线上", "故障"}, resp.Tags)
	assert.NotEmpty(t, resp.Reason)
}

// ---- Expand ----

func TestExpand_OK(t *testing.T) {
	content := `{"description":"完善周报数据并同步给主管","steps":["收集数据","撰写初稿","复核","发送"]}`
	svc := NewTaskAIService(newStubAI(content), nil)

	resp, err := svc.Expand(context.Background(), dto.ExpandReq{Title: "写周报"})
	require.NoError(t, err)
	assert.Equal(t, "完善周报数据并同步给主管", resp.Description)
	assert.Len(t, resp.Steps, 4)
}

// ---- Summarize ----

func TestSummarize_NilRepo(t *testing.T) {
	svc := NewTaskAIService(newStubAI("{}"), nil)
	_, err := svc.Summarize(context.Background(), "t1", "u1", "all")
	assert.ErrorIs(t, err, ErrAIUnavailable)
}

func TestSummarize_Unavailable(t *testing.T) {
	repo := newStubTaskRepo()
	svc := NewTaskAIService(&stubAI{usable: false}, repo)
	_, err := svc.Summarize(context.Background(), "t1", "u1", "all")
	assert.ErrorIs(t, err, ErrAIUnavailable)
}

func TestSummarize_EmptyResult(t *testing.T) {
	repo := newStubTaskRepo() // 无任务
	svc := NewTaskAIService(newStubAI("{}"), repo)

	resp, err := svc.Summarize(context.Background(), "t1", "u1", "week")
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Total)
	assert.NotEmpty(t, resp.Text) // 命中“暂无任务”短路，未调用模型
}

func TestSummarize_OK(t *testing.T) {
	repo := newStubTaskRepo()
	repo.tasks["x"] = &model.Task{
		ID: "x", TenantID: "t1", CreatorID: "u1", AssigneeID: "u1",
		Title: "写周报", Status: model.TaskStatusPending, Priority: model.TaskPriorityHigh,
	}
	content := `{"text":"本周有 1 条高优待办","highlights":["写周报未开始"],"suggested_focus":["写周报"]}`
	svc := NewTaskAIService(newStubAI(content), repo)

	resp, err := svc.Summarize(context.Background(), "t1", "u1", "all")
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, "本周有 1 条高优待办", resp.Text)
	assert.Equal(t, []string{"写周报未开始"}, resp.Highlights)
	assert.Equal(t, []string{"写周报"}, resp.SuggestedFocus)
}

func TestSummarize_BadJSON(t *testing.T) {
	repo := newStubTaskRepo()
	repo.tasks["x"] = &model.Task{ID: "x", TenantID: "t1", Title: "任务", Status: model.TaskStatusPending}
	svc := NewTaskAIService(newStubAI("坏 JSON"), repo)

	_, err := svc.Summarize(context.Background(), "t1", "u1", "all")
	assert.ErrorIs(t, err, ErrAIRuntime)
}
