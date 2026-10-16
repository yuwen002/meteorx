// Package handler 实现任务模块 AI 辅助能力的 HTTP 接口。
// 端点均挂在受保护的 /tasks 分组下（需登录），访问控制以数据归属为准；
// AI 不可用/失败降级为 503/502，绝不因 AI 故障影响任务主流程。
package handler

import (
	"errors"
	"net/http"
	"time"

	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/task/dto"
	"meteorx/internal/modules/task/service"
)

// TaskAIHandler 任务 AI 辅助能力 HTTP 处理器。
type TaskAIHandler struct {
	svc *service.TaskAIService
}

// NewTaskAIHandler 创建任务 AI 处理器实例。
func NewTaskAIHandler(svc *service.TaskAIService) *TaskAIHandler {
	return &TaskAIHandler{svc: svc}
}

// Capabilities 探测 AI 能力可用性与默认模型。GET /api/v1/tasks/ai/capabilities
func (h *TaskAIHandler) Capabilities(w http.ResponseWriter, r *http.Request) {
	response.Success(w, h.svc.Capabilities())
}

// Parse 自然语言解析为任务草稿。POST /api/v1/tasks/ai/parse
func (h *TaskAIHandler) Parse(w http.ResponseWriter, r *http.Request) {
	var req dto.ParseTaskReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	draft, err := h.svc.ParseDraft(r.Context(), req.Text, time.Now())
	if err != nil {
		writeAIError(w, err, "解析任务失败")
		return
	}
	response.Success(w, draft)
}

// Suggest 优先级/标签建议。POST /api/v1/tasks/ai/suggest
func (h *TaskAIHandler) Suggest(w http.ResponseWriter, r *http.Request) {
	var req dto.SuggestReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	resp, err := h.svc.Suggest(r.Context(), req)
	if err != nil {
		writeAIError(w, err, "生成建议失败")
		return
	}
	response.Success(w, resp)
}

// Expand 描述扩写/拆解。POST /api/v1/tasks/ai/expand
func (h *TaskAIHandler) Expand(w http.ResponseWriter, r *http.Request) {
	var req dto.ExpandReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	resp, err := h.svc.Expand(r.Context(), req)
	if err != nil {
		writeAIError(w, err, "扩写描述失败")
		return
	}
	response.Success(w, resp)
}

// Summarize 任务摘要/周报。POST /api/v1/tasks/ai/summary
func (h *TaskAIHandler) Summarize(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	if tenantID == "" || userID == "" {
		response.Unauthorized(w, "未获取到用户信息")
		return
	}
	var req dto.SummaryReq
	// 请求体应为 JSON 对象（可用 {} 表示默认 all 范围）；scope 为空时服务层回落 all
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	resp, err := h.svc.Summarize(r.Context(), tenantID, userID, req.Scope)
	if err != nil {
		writeAIError(w, err, "生成摘要失败")
		return
	}
	response.Success(w, resp)
}

// writeAIError 将 AI 相关错误映射为降级响应：不可用 503、运行时失败 502，其余 500。
func writeAIError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrAIUnavailable):
		response.Fail(w, http.StatusServiceUnavailable, "AI 能力未启用")
	case errors.Is(err, service.ErrAIRuntime):
		response.Fail(w, http.StatusBadGateway, "AI 服务暂不可用，请稍后重试")
	default:
		response.Fail(w, http.StatusInternalServerError, fallback)
	}
}
