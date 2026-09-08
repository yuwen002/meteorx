package handler_test

import (
	"context"
	"io"
	"mime/multipart"

	"meteorx/internal/modules/file/dto"
)

// stubFileService 桩实现 handler.FileService
type stubFileService struct {
	Err        error
	UploadResp *dto.UploadFileResp
	File       *dto.FileResp
	Files      []*dto.FileResp
	Total      int64
	BatchResp  *dto.BatchDeleteResp
	Reader     io.ReadCloser
	Filename   string

	GotID        string
	GotTenantID  string
	GotUserID    string
	GotHeader    *multipart.FileHeader
	GotListReq   *dto.FileListReq
	GotPage      int
	GotPageSize  int
	GotUpdateReq *dto.FileUpdateReq
	GotBatchReq  *dto.BatchDeleteReq
}

func (s *stubFileService) Upload(_ context.Context, fileHeader *multipart.FileHeader, tenantID, userID string) (*dto.UploadFileResp, error) {
	s.GotHeader, s.GotTenantID, s.GotUserID = fileHeader, tenantID, userID
	return s.UploadResp, s.Err
}

func (s *stubFileService) GetByID(_ context.Context, id string) (*dto.FileResp, error) {
	s.GotID = id
	return s.File, s.Err
}

func (s *stubFileService) GetByIDWithScope(_ context.Context, id, tenantID string) (*dto.FileResp, error) {
	s.GotID, s.GotTenantID = id, tenantID
	return s.File, s.Err
}

func (s *stubFileService) ListByTenant(_ context.Context, tenantID string, req *dto.FileListReq) ([]*dto.FileResp, int64, error) {
	s.GotTenantID, s.GotListReq = tenantID, req
	return s.Files, s.Total, s.Err
}

func (s *stubFileService) ListByUser(_ context.Context, tenantID, userID string, req *dto.FileListReq) ([]*dto.FileResp, int64, error) {
	s.GotTenantID, s.GotUserID, s.GotListReq = tenantID, userID, req
	return s.Files, s.Total, s.Err
}

func (s *stubFileService) Update(_ context.Context, id string, tenantID string, req *dto.FileUpdateReq) error {
	s.GotID, s.GotTenantID, s.GotUpdateReq = id, tenantID, req
	return s.Err
}

func (s *stubFileService) Delete(_ context.Context, id, tenantID string) error {
	s.GotID, s.GotTenantID = id, tenantID
	return s.Err
}

func (s *stubFileService) BatchDelete(_ context.Context, tenantID string, req *dto.BatchDeleteReq) (*dto.BatchDeleteResp, error) {
	s.GotTenantID, s.GotBatchReq = tenantID, req
	return s.BatchResp, s.Err
}

func (s *stubFileService) GetDeletedList(_ context.Context, tenantID string, page, pageSize int) ([]*dto.FileResp, int64, error) {
	s.GotTenantID, s.GotPage, s.GotPageSize = tenantID, page, pageSize
	return s.Files, s.Total, s.Err
}

func (s *stubFileService) Restore(_ context.Context, id, tenantID string) error {
	s.GotID, s.GotTenantID = id, tenantID
	return s.Err
}

func (s *stubFileService) PermanentDelete(_ context.Context, id, tenantID string) error {
	s.GotID, s.GotTenantID = id, tenantID
	return s.Err
}

func (s *stubFileService) Download(_ context.Context, id, tenantID string) (io.ReadCloser, string, error) {
	s.GotID, s.GotTenantID = id, tenantID
	return s.Reader, s.Filename, s.Err
}
