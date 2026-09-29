package service

import (
	"context"
	"fmt"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	apperrors "meteorx/internal/pkg/apperrors"
)

var validTransitions = map[string]map[string]bool{
	model.PublishStatusDraft: {
		model.PublishStatusPendingReview: true,
	},
	model.PublishStatusPendingReview: {
		model.PublishStatusPublished: true,
		model.PublishStatusRejected: true,
		model.PublishStatusDraft:    true,
	},
	model.PublishStatusPublished: {
		model.PublishStatusArchived: true,
		model.PublishStatusDraft:    true,
	},
	model.PublishStatusRejected: {
		model.PublishStatusPendingReview: true,
		model.PublishStatusDraft:         true,
	},
	model.PublishStatusArchived: {
		model.PublishStatusDraft: true,
	},
}

func canTransition(from, to string) bool {
	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}
	return allowed[to]
}

func (s *wikiServiceExtended) SubmitForReview(ctx context.Context, documentID string, req *dto.SubmitForReviewReq) (*dto.DocumentPublishStatusResp, error) {
	userID := contextx.GetUserID(ctx)

	if err := s.CheckDocumentPermission(ctx, documentID, userID, "update"); err != nil {
		return nil, err
	}

	doc, node, err := s.getExtendedRepo().GetDocumentByIDWithNode(ctx, documentID)
	if err != nil {
		return nil, err
	}

	if !canTransition(doc.PublishStatus, model.PublishStatusPendingReview) {
		return nil, apperrors.NewConflict(fmt.Sprintf("文档当前状态为 %s，无法提交审核", doc.PublishStatus))
	}

	if err := s.getExtendedRepo().UpdateDocumentPublishStatus(ctx, documentID, model.PublishStatusPendingReview, "", nil, nil); err != nil {
		return nil, err
	}

	if req.Comment != "" {
		_ = s.getExtendedRepo().CreateReviewComment(ctx, &model.ReviewComment{
			DocumentID: documentID,
			NodeID:     doc.NodeID,
			Action:     model.ReviewActionSubmit,
			Content:    req.Comment,
			ReviewerID: userID,
		})
	}

	return &dto.DocumentPublishStatusResp{
		DocumentID:    documentID,
		NodeID:        doc.NodeID,
		Title:         node.Title,
		PublishStatus: model.PublishStatusPendingReview,
	}, nil
}

func (s *wikiServiceExtended) ApproveDocument(ctx context.Context, documentID string, req *dto.ReviewActionReq) (*dto.DocumentPublishStatusResp, error) {
	userID := contextx.GetUserID(ctx)

	if err := s.CheckDocumentPermission(ctx, documentID, userID, "review"); err != nil {
		return nil, err
	}

	doc, node, err := s.getExtendedRepo().GetDocumentByIDWithNode(ctx, documentID)
	if err != nil {
		return nil, err
	}

	if !canTransition(doc.PublishStatus, model.PublishStatusPublished) {
		return nil, apperrors.NewConflict(fmt.Sprintf("文档当前状态为 %s，无法审核通过", doc.PublishStatus))
	}

	now := time.Now()
	if err := s.getExtendedRepo().UpdateDocumentPublishStatus(ctx, documentID, model.PublishStatusPublished, userID, &now, &now); err != nil {
		return nil, err
	}

	_ = s.getExtendedRepo().CreateReviewComment(ctx, &model.ReviewComment{
		DocumentID: documentID,
		NodeID:     doc.NodeID,
		Action:     model.ReviewActionApprove,
		Content:    req.Comment,
		ReviewerID: userID,
	})

	return &dto.DocumentPublishStatusResp{
		DocumentID:    documentID,
		NodeID:        doc.NodeID,
		Title:         node.Title,
		PublishStatus: model.PublishStatusPublished,
		PublishedAt:   &now,
		ReviewedBy:    userID,
		ReviewedAt:    &now,
	}, nil
}

func (s *wikiServiceExtended) RejectDocument(ctx context.Context, documentID string, req *dto.ReviewActionReq) (*dto.DocumentPublishStatusResp, error) {
	userID := contextx.GetUserID(ctx)

	if err := s.CheckDocumentPermission(ctx, documentID, userID, "review"); err != nil {
		return nil, err
	}

	doc, node, err := s.getExtendedRepo().GetDocumentByIDWithNode(ctx, documentID)
	if err != nil {
		return nil, err
	}

	if !canTransition(doc.PublishStatus, model.PublishStatusRejected) {
		return nil, apperrors.NewConflict(fmt.Sprintf("文档当前状态为 %s，无法驳回", doc.PublishStatus))
	}

	now := time.Now()
	if err := s.getExtendedRepo().UpdateDocumentPublishStatus(ctx, documentID, model.PublishStatusRejected, userID, &now, nil); err != nil {
		return nil, err
	}

	_ = s.getExtendedRepo().CreateReviewComment(ctx, &model.ReviewComment{
		DocumentID: documentID,
		NodeID:     doc.NodeID,
		Action:     model.ReviewActionReject,
		Content:    req.Comment,
		ReviewerID: userID,
	})

	return &dto.DocumentPublishStatusResp{
		DocumentID:    documentID,
		NodeID:        doc.NodeID,
		Title:         node.Title,
		PublishStatus: model.PublishStatusRejected,
		ReviewedBy:    userID,
		ReviewedAt:    &now,
	}, nil
}

func (s *wikiServiceExtended) PublishDocument(ctx context.Context, documentID string) (*dto.DocumentPublishStatusResp, error) {
	userID := contextx.GetUserID(ctx)

	if err := s.CheckDocumentPermission(ctx, documentID, userID, "publish"); err != nil {
		return nil, err
	}

	doc, node, err := s.getExtendedRepo().GetDocumentByIDWithNode(ctx, documentID)
	if err != nil {
		return nil, err
	}

	if !canTransition(doc.PublishStatus, model.PublishStatusPublished) {
		return nil, apperrors.NewConflict(fmt.Sprintf("文档当前状态为 %s，无法直接发布", doc.PublishStatus))
	}

	now := time.Now()
	if err := s.getExtendedRepo().UpdateDocumentPublishStatus(ctx, documentID, model.PublishStatusPublished, userID, &now, &now); err != nil {
		return nil, err
	}

	_ = s.getExtendedRepo().CreateReviewComment(ctx, &model.ReviewComment{
		DocumentID: documentID,
		NodeID:     doc.NodeID,
		Action:     model.ReviewActionPublish,
		ReviewerID: userID,
	})

	return &dto.DocumentPublishStatusResp{
		DocumentID:    documentID,
		NodeID:        doc.NodeID,
		Title:         node.Title,
		PublishStatus: model.PublishStatusPublished,
		PublishedAt:   &now,
		ReviewedBy:    userID,
		ReviewedAt:    &now,
	}, nil
}

func (s *wikiServiceExtended) UnpublishDocument(ctx context.Context, documentID string) (*dto.DocumentPublishStatusResp, error) {
	userID := contextx.GetUserID(ctx)

	if err := s.CheckDocumentPermission(ctx, documentID, userID, "publish"); err != nil {
		return nil, err
	}

	doc, node, err := s.getExtendedRepo().GetDocumentByIDWithNode(ctx, documentID)
	if err != nil {
		return nil, err
	}

	if !canTransition(doc.PublishStatus, model.PublishStatusDraft) {
		return nil, apperrors.NewConflict(fmt.Sprintf("文档当前状态为 %s，无法取消发布", doc.PublishStatus))
	}

	if err := s.getExtendedRepo().UpdateDocumentPublishStatus(ctx, documentID, model.PublishStatusDraft, "", nil, nil); err != nil {
		return nil, err
	}

	return &dto.DocumentPublishStatusResp{
		DocumentID:    documentID,
		NodeID:        doc.NodeID,
		Title:         node.Title,
		PublishStatus: model.PublishStatusDraft,
	}, nil
}

func (s *wikiServiceExtended) ArchiveDocument(ctx context.Context, documentID string) (*dto.DocumentPublishStatusResp, error) {
	userID := contextx.GetUserID(ctx)

	if err := s.CheckDocumentPermission(ctx, documentID, userID, "publish"); err != nil {
		return nil, err
	}

	doc, node, err := s.getExtendedRepo().GetDocumentByIDWithNode(ctx, documentID)
	if err != nil {
		return nil, err
	}

	if !canTransition(doc.PublishStatus, model.PublishStatusArchived) {
		return nil, apperrors.NewConflict(fmt.Sprintf("文档当前状态为 %s，无法归档", doc.PublishStatus))
	}

	if err := s.getExtendedRepo().UpdateDocumentPublishStatus(ctx, documentID, model.PublishStatusArchived, "", nil, nil); err != nil {
		return nil, err
	}

	_ = s.getExtendedRepo().CreateReviewComment(ctx, &model.ReviewComment{
		DocumentID: documentID,
		NodeID:     doc.NodeID,
		Action:     model.ReviewActionArchive,
		ReviewerID: userID,
	})

	return &dto.DocumentPublishStatusResp{
		DocumentID:    documentID,
		NodeID:        doc.NodeID,
		Title:         node.Title,
		PublishStatus: model.PublishStatusArchived,
	}, nil
}

func (s *wikiServiceExtended) ListReviewComments(ctx context.Context, documentID string) ([]*dto.ReviewCommentResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "read"); err != nil {
		return nil, err
	}

	comments, err := s.getExtendedRepo().ListReviewComments(ctx, documentID)
	if err != nil {
		return nil, err
	}

	var resps []*dto.ReviewCommentResp
	for _, c := range comments {
		resp := &dto.ReviewCommentResp{
			ID:         c.ID,
			DocumentID: c.DocumentID,
			NodeID:     c.NodeID,
			Action:     c.Action,
			Content:    c.Content,
			ReviewerID: c.ReviewerID,
			CreatedAt:  c.CreatedAt,
		}

		var user struct {
			Username string `gorm:"column:username"`
		}
		if err := s.tx.DB().Table("users").Select("username").Where("id = ?", c.ReviewerID).First(&user).Error; err == nil {
			resp.ReviewerName = user.Username
		}

		resps = append(resps, resp)
	}
	return resps, nil
}

func (s *wikiServiceExtended) ListPendingReviews(ctx context.Context, page, pageSize int) ([]*dto.ListPendingReviewsResp, int64, error) {
	tenantID := contextx.GetTenantID(ctx)

	docs, total, err := s.getExtendedRepo().ListPendingReviewDocuments(ctx, tenantID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	var resps []*dto.ListPendingReviewsResp
	for _, doc := range docs {
		resp := &dto.ListPendingReviewsResp{
			DocumentID:    doc.ID,
			NodeID:        doc.NodeID,
			PublishStatus: doc.PublishStatus,
		}
		if !doc.UpdatedAt.IsZero() {
			t := doc.UpdatedAt
			resp.SubmittedAt = &t
		}

		if node, err := s.getExtendedRepo().GetNodeByID(ctx, doc.NodeID); err == nil {
			resp.Title = node.Title
			resp.SpaceID = node.SpaceID
			if space, err := s.getExtendedRepo().GetSpaceByID(ctx, node.SpaceID); err == nil {
				resp.SpaceName = space.Name
			}
		}

		resps = append(resps, resp)
	}
	return resps, total, nil
}