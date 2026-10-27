package handler_test

import (
	"context"

	"meteorx/internal/modules/export/dto"
	"meteorx/internal/modules/export/model"
)

// stubExportService 桩实现 handler.ExportServiceAPI，记录入参并按预置字段返回。
type stubExportService struct {
	createJob *model.ExportJob
	createErr error
	getJob    *model.ExportJob
	getErr    error
	listItems []*model.ExportJob
	listTotal int64
	listErr   error
	delJob    *model.ExportJob
	delErr    error

	gotTenant string
	gotUser   string
	gotMaster bool
	gotID     string
	gotList   dto.ListExportReq
	gotCreate dto.CreateExportReq
}

func (s *stubExportService) CreateExport(_ context.Context, tenantID, userID string, req dto.CreateExportReq) (*model.ExportJob, error) {
	s.gotTenant, s.gotUser, s.gotCreate = tenantID, userID, req
	return s.createJob, s.createErr
}

func (s *stubExportService) GetByID(_ context.Context, id, tenantID, userID string, isMaster bool) (*model.ExportJob, error) {
	s.gotID, s.gotTenant, s.gotUser, s.gotMaster = id, tenantID, userID, isMaster
	return s.getJob, s.getErr
}

func (s *stubExportService) List(_ context.Context, tenantID, userID string, req dto.ListExportReq, isMaster bool) ([]*model.ExportJob, int64, error) {
	s.gotTenant, s.gotUser, s.gotList, s.gotMaster = tenantID, userID, req, isMaster
	return s.listItems, s.listTotal, s.listErr
}

func (s *stubExportService) Delete(_ context.Context, id, tenantID, userID string, isMaster bool) (*model.ExportJob, error) {
	s.gotID, s.gotTenant, s.gotUser, s.gotMaster = id, tenantID, userID, isMaster
	return s.delJob, s.delErr
}
