package handler_test

import (
	"context"

	"meteorx/internal/modules/plan/dto"
)

// stubPlanService 桩实现 handler.PlanService
type stubPlanService struct {
	Err     error
	Resp    *dto.PlanResp
	Current *dto.CurrentPlanResp
	List    []*dto.PlanResp
	Total   int64

	GotID      string
	GotTenant  string
	GotPage    int
	GotPageSz  int
	GotKeyword string
	GotStatus  *int
	GotCreate  *dto.CreatePlanReq
	GotUpdate  *dto.UpdatePlanReq
	GotAssign  *dto.AssignPlanReq
}

func (s *stubPlanService) CreatePlan(_ context.Context, req dto.CreatePlanReq) (*dto.PlanResp, error) {
	s.GotCreate = &req
	return s.Resp, s.Err
}

func (s *stubPlanService) UpdatePlan(_ context.Context, id string, req dto.UpdatePlanReq) (*dto.PlanResp, error) {
	s.GotID, s.GotUpdate = id, &req
	return s.Resp, s.Err
}

func (s *stubPlanService) ListPlans(_ context.Context, page, pageSize int, keyword string, status *int) ([]*dto.PlanResp, int64, error) {
	s.GotPage, s.GotPageSz, s.GotKeyword, s.GotStatus = page, pageSize, keyword, status
	return s.List, s.Total, s.Err
}

func (s *stubPlanService) ListEnabledPlans(_ context.Context) ([]*dto.PlanResp, error) {
	return s.List, s.Err
}

func (s *stubPlanService) DeletePlan(_ context.Context, id string) error {
	s.GotID = id
	return s.Err
}

func (s *stubPlanService) AssignPlan(_ context.Context, tenantID string, req dto.AssignPlanReq) error {
	s.GotTenant, s.GotAssign = tenantID, &req
	return s.Err
}

func (s *stubPlanService) GetCurrentPlan(_ context.Context, tenantID string) (*dto.CurrentPlanResp, error) {
	s.GotTenant = tenantID
	return s.Current, s.Err
}
