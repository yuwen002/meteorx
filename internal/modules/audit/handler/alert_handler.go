package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"meteorx/internal/common/response"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
)

// AlertService 告警服务接口（handler 依赖的最小业务面；*service.AlertService 完整实现）
type AlertService interface {
	CreateRule(ctx context.Context, req dto.CreateAlertRuleReq) (*model.AlertRule, error)
	UpdateRule(ctx context.Context, id string, req dto.UpdateAlertRuleReq) (*model.AlertRule, error)
	DeleteRule(ctx context.Context, id string) error
	ListRules(ctx context.Context) ([]*model.AlertRule, error)
	ListAlerts(ctx context.Context, page, pageSize int, ruleID, userID, riskLevel string) ([]*model.AuditAlert, int64, error)
	GetAlertStats(ctx context.Context, days int) (*model.AlertStats, error)
}

type AlertHandler struct {
	alertSvc AlertService
}

func NewAlertHandler(alertSvc AlertService) *AlertHandler {
	return &AlertHandler{alertSvc: alertSvc}
}

// CreateRule 创建告警规则
// POST /api/v1/audit/alert-rules
func (h *AlertHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAlertRuleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "请求参数错误")
		return
	}

	rule, err := h.alertSvc.CreateRule(r.Context(), req)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "创建告警规则失败")
		return
	}

	response.Success(w, dto.ToAlertRuleResp(rule))
}

// UpdateRule 更新告警规则
// PUT /api/v1/audit/alert-rules/:id
func (h *AlertHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "规则ID不能为空")
		return
	}

	var req dto.UpdateAlertRuleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "请求参数错误")
		return
	}

	rule, err := h.alertSvc.UpdateRule(r.Context(), id, req)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "更新告警规则失败")
		return
	}

	response.Success(w, dto.ToAlertRuleResp(rule))
}

// DeleteRule 删除告警规则
// DELETE /api/v1/audit/alert-rules/:id
func (h *AlertHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "规则ID不能为空")
		return
	}

	if err := h.alertSvc.DeleteRule(r.Context(), id); err != nil {
		response.Fail(w, http.StatusInternalServerError, "删除告警规则失败")
		return
	}

	response.Success(w, nil)
}

// ListRules 获取所有告警规则
// GET /api/v1/audit/alert-rules
func (h *AlertHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.alertSvc.ListRules(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取告警规则列表失败")
		return
	}

	resp := make([]*dto.AlertRuleResp, len(rules))
	for i, rule := range rules {
		resp[i] = dto.ToAlertRuleResp(rule)
	}

	response.Success(w, resp)
}

// ListAlerts 分页查询告警记录
// GET /api/v1/audit/alerts
func (h *AlertHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	alerts, total, err := h.alertSvc.ListAlerts(
		r.Context(),
		page,
		pageSize,
		r.URL.Query().Get("rule_id"),
		r.URL.Query().Get("user_id"),
		r.URL.Query().Get("risk_level"),
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取告警记录列表失败")
		return
	}

	resp := make([]*dto.AlertLogResp, len(alerts))
	for i, alert := range alerts {
		resp[i] = dto.ToAlertLogResp(alert)
	}

	result := map[string]interface{}{
		"items": resp,
		"total": total,
	}
	response.Success(w, result)
}

// GetAlertStats 获取告警统计数据
// GET /api/v1/audit/alerts/stats
func (h *AlertHandler) GetAlertStats(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days, _ := strconv.Atoi(daysStr)

	if days <= 0 {
		days = 7
	}

	stats, err := h.alertSvc.GetAlertStats(r.Context(), days)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取告警统计失败")
		return
	}

	response.Success(w, stats)
}
