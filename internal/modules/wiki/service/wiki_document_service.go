package service

import (
	"context"
	"errors"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"
	apperrors "meteorx/internal/pkg/apperrors"

	"gorm.io/gorm"
)

// CreateDocument 在节点上创建文档，自动渲染 Markdown
func (s *wikiService) CreateDocument(ctx context.Context, nodeID string, userID string, req *dto.CreateDocumentReq) (*dto.DocumentResp, error) {
	if err := s.CheckNodePermission(ctx, nodeID, userID, "create"); err != nil {
		return nil, err
	}

	node, err := s.repo.GetNodeByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if node.Type != model.NodeTypeDocument {
		return nil, errors.New("node is not a document")
	}

	format := req.Format
	if format == "" {
		format = "markdown"
	}

	contentHTML := ""
	if format == "markdown" && req.Content != "" {
		contentHTML = s.markdownSvc.RenderAndSanitize(req.Content)
	}

	doc := &model.Document{
		NodeID:      nodeID,
		Content:     req.Content,
		ContentHTML: contentHTML,
		Format:      format,
	}
	if err := s.repo.CreateDocument(ctx, doc); err != nil {
		return nil, err
	}

	return s.buildDocumentResp(ctx, node, doc)
}

// GetDocument 获取文档内容
func (s *wikiService) GetDocument(ctx context.Context, nodeID string, userID string) (*dto.DocumentResp, error) {
	if err := s.CheckNodePermission(ctx, nodeID, userID, "read"); err != nil {
		return nil, err
	}

	node, err := s.repo.GetNodeByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	doc, err := s.repo.GetDocumentByNodeID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	_ = s.repo.IncrementViewCount(ctx, doc.ID)

	return s.buildDocumentResp(ctx, node, doc)
}

// UpdateDocument 更新文档内容，事务中自动创建 Revision 并使用乐观锁
func (s *wikiService) UpdateDocument(ctx context.Context, id string, userID string, req *dto.UpdateDocumentReq) (*dto.DocumentResp, error) {
	if err := s.CheckDocumentPermission(ctx, id, userID, "update"); err != nil {
		return nil, err
	}

	var resp *dto.DocumentResp
	err := s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		doc, err := s.repo.GetDocumentByID(txCtx, id)
		if err != nil {
			return err
		}
		expectedVer := doc.CurrentVer

		if req.Content != "" || req.Format != "" {
			revision := &model.DocumentRevision{
				DocumentID:  doc.ID,
				Version:     doc.CurrentVer,
				Content:     doc.Content,
				ContentHTML: doc.ContentHTML,
				Summary:     req.Summary,
				EditedBy:    doc.LastEditedBy,
			}
			if err := s.repo.CreateRevision(txCtx, revision); err != nil {
				return err
			}

			if req.Content != "" {
				doc.Content = req.Content
			}
			if req.Format != "" {
				doc.Format = req.Format
			}

			if doc.Format == "markdown" && doc.Content != "" {
				doc.ContentHTML = s.markdownSvc.RenderAndSanitize(doc.Content)
			} else if doc.Format != "markdown" {
				doc.ContentHTML = ""
			}

			doc.CurrentVer++
			doc.LastEditedBy = userID
			now := time.Now()
			doc.LastEditedAt = &now
		}

		if err := s.repo.UpdateDocument(txCtx, doc, expectedVer); err != nil {
			if errors.Is(err, repository.ErrDocumentVersionConflict) {
				return apperrors.NewConflict("文档已被其他用户修改，请刷新后重试")
			}
			return err
		}

		node, err := s.repo.GetNodeByID(txCtx, doc.NodeID)
		if err != nil {
			return err
		}
		buildResp, err := s.buildDocumentResp(txCtx, node, doc)
		if err != nil {
			return err
		}
		resp = buildResp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// DeleteDocument 删除文档，先创建回收站记录再级联删除附件
func (s *wikiService) DeleteDocument(ctx context.Context, id string) error {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, id, userID, "delete"); err != nil {
		return err
	}

	doc, err := s.repo.GetDocumentByID(ctx, id)
	if err != nil {
		return err
	}

	node, err := s.repo.GetNodeByID(ctx, doc.NodeID)
	if err != nil {
		return err
	}

	if err := s.MoveToTrash(ctx, model.TrashTypeDocument, id, node.SpaceID, node.Title, userID); err != nil {
		return err
	}

	return s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		if err := s.repo.DeleteAttachmentsByDocument(txCtx, id); err != nil {
			return err
		}
		return s.repo.DeleteDocument(txCtx, id)
	})
}

// buildDocumentResp 构建文档响应结构（包含标题、内容、Revision 信息）
func (s *wikiService) buildDocumentResp(ctx context.Context, node *model.WikiNode, doc *model.Document) (*dto.DocumentResp, error) {
	title := ""
	if node != nil {
		title = node.Title
	}

	return &dto.DocumentResp{
		ID:           doc.ID,
		NodeID:       doc.NodeID,
		Title:        title,
		Content:      doc.Content,
		ContentHTML:  s.rewriteImageSrc(doc.ContentHTML),
		Format:       doc.Format,
		CurrentVer:   doc.CurrentVer,
		ViewCount:    doc.ViewCount,
		LastEditedBy: doc.LastEditedBy,
		LastEditedAt: doc.LastEditedAt,
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
	}, nil
}

// ListRevisions 列出文档的所有历史版本
func (s *wikiService) ListRevisions(ctx context.Context, documentID string) ([]*dto.DocumentRevisionResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "read"); err != nil {
		return nil, err
	}

	revisions, err := s.repo.ListRevisions(ctx, documentID)
	if err != nil {
		return nil, err
	}

	var resps []*dto.DocumentRevisionResp
	for _, r := range revisions {
		resps = append(resps, &dto.DocumentRevisionResp{
			ID:          r.ID,
			DocumentID:  r.DocumentID,
			Version:     r.Version,
			Content:     r.Content,
			ContentHTML: s.rewriteImageSrc(r.ContentHTML),
			Summary:     r.Summary,
			EditedBy:    r.EditedBy,
			CreatedAt:   r.CreatedAt,
		})
	}
	return resps, nil
}

// GetRevision 获取指定版本的历史内容
func (s *wikiService) GetRevision(ctx context.Context, documentID string, version int) (*dto.DocumentRevisionResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "read"); err != nil {
		return nil, err
	}

	revision, err := s.repo.GetRevisionByVersion(ctx, documentID, version)
	if err != nil {
		return nil, err
	}
	return &dto.DocumentRevisionResp{
		ID:          revision.ID,
		DocumentID:  revision.DocumentID,
		Version:     revision.Version,
		Content:     revision.Content,
		ContentHTML: s.rewriteImageSrc(revision.ContentHTML),
		Summary:     revision.Summary,
		EditedBy:    revision.EditedBy,
		CreatedAt:   revision.CreatedAt,
	}, nil
}

// RestoreRevision 恢复到指定版本，自动保存当前状态为新 Revision
func (s *wikiService) RestoreRevision(ctx context.Context, documentID string, version int, userID string) (*dto.DocumentResp, error) {
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "update"); err != nil {
		return nil, err
	}

	var resp *dto.DocumentResp
	err := s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		revision, err := s.repo.GetRevisionByVersion(txCtx, documentID, version)
		if err != nil {
			return err
		}

		doc, err := s.repo.GetDocumentByID(txCtx, documentID)
		if err != nil {
			return err
		}

		currentVer := doc.CurrentVer

		// 保存当前状态为新的 Revision
		currentRevision := &model.DocumentRevision{
			DocumentID:  doc.ID,
			Version:     currentVer,
			Content:     doc.Content,
			ContentHTML: doc.ContentHTML,
			Summary:     "Auto-saved before restore",
			EditedBy:    doc.LastEditedBy,
		}
		if err := s.repo.CreateRevision(txCtx, currentRevision); err != nil {
			return err
		}

		// 恢复到指定版本
		doc.Content = revision.Content
		doc.ContentHTML = revision.ContentHTML
		doc.CurrentVer = currentVer + 1
		doc.LastEditedBy = userID
		now := time.Now()
		doc.LastEditedAt = &now

		if err := s.repo.UpdateDocument(txCtx, doc, currentVer); err != nil {
			if errors.Is(err, repository.ErrDocumentVersionConflict) {
				return apperrors.NewConflict("文档已被其他用户修改，请刷新后重试")
			}
			return err
		}

		node, err := s.repo.GetNodeByID(txCtx, doc.NodeID)
		if err != nil {
			return err
		}
		buildResp, err := s.buildDocumentResp(txCtx, node, doc)
		if err != nil {
			return err
		}
		resp = buildResp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
