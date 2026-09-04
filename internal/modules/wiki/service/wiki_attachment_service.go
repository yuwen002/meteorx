package service

import (
	"context"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	apperrors "meteorx/internal/pkg/apperrors"
)

// CreateAttachment 创建文档附件（需文档 update 权限）
func (s *wikiService) CreateAttachment(ctx context.Context, userID string, req *dto.CreateAttachmentReq) (*dto.AttachmentResp, error) {
	if _, err := s.repo.GetDocumentByID(ctx, req.DocumentID); err != nil {
		return nil, err
	}

	if err := s.CheckDocumentPermission(ctx, req.DocumentID, userID, "update"); err != nil {
		return nil, err
	}

	attachment := &model.Attachment{
		TenantID:   contextx.GetTenantID(ctx),
		DocumentID: req.DocumentID,
		FileName:   req.FileName,
		FileSize:   req.FileSize,
		MimeType:   req.MimeType,
		FileURL:    req.FileURL,
		UploadedBy: userID,
	}

	if err := s.repo.CreateAttachment(ctx, attachment); err != nil {
		return nil, err
	}

	return &dto.AttachmentResp{
		ID:         attachment.ID,
		DocumentID: attachment.DocumentID,
		FileName:   attachment.FileName,
		FileSize:   attachment.FileSize,
		MimeType:   attachment.MimeType,
		FileURL:    attachment.FileURL,
		UploadedBy: attachment.UploadedBy,
		CreatedAt:  attachment.CreatedAt,
	}, nil
}

// ListAttachments 列出文档的所有附件（需文档 read 权限）
func (s *wikiService) ListAttachments(ctx context.Context, documentID string, userID string) ([]*dto.AttachmentResp, error) {
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "read"); err != nil {
		return nil, err
	}

	attachments, err := s.repo.ListAttachmentsByDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}

	var resps []*dto.AttachmentResp
	for _, a := range attachments {
		resps = append(resps, &dto.AttachmentResp{
			ID:         a.ID,
			DocumentID: a.DocumentID,
			FileName:   a.FileName,
			FileSize:   a.FileSize,
			MimeType:   a.MimeType,
			FileURL:    a.FileURL,
			UploadedBy: a.UploadedBy,
			CreatedAt:  a.CreatedAt,
		})
	}

	return resps, nil
}

// DeleteAttachment 删除附件（需文档 update 权限）
func (s *wikiService) DeleteAttachment(ctx context.Context, id string, userID string) error {
	attachment, err := s.repo.GetAttachment(ctx, id)
	if err != nil {
		return apperrors.ErrNotFound("附件不存在")
	}

	if err := s.CheckDocumentPermission(ctx, attachment.DocumentID, userID, "update"); err != nil {
		return err
	}

	return s.repo.DeleteAttachment(ctx, id)
}
