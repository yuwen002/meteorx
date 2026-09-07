package handler_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/handler"
	apperrors "meteorx/internal/pkg/apperrors"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- stub：标签 / 评论 / 分享 / 模板 ----------

func (s *stubWikiService) CreateTag(_ context.Context, req *dto.CreateTagReq) (*dto.TagResp, error) {
	s.rec("CreateTag", req.Name, req.Color)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.TagResp{ID: "tag-1", Name: req.Name}, nil
}

func (s *stubWikiService) ListTags(_ context.Context) ([]*dto.TagResp, error) {
	s.rec("ListTags")
	if s.err != nil {
		return nil, s.err
	}
	return []*dto.TagResp{{ID: "tag-1", Name: "重要"}}, nil
}

func (s *stubWikiService) DeleteTag(_ context.Context, id string) error {
	s.rec("DeleteTag", id)
	return s.err
}

func (s *stubWikiService) AddDocumentTag(_ context.Context, documentID, tagID string) error {
	s.rec("AddDocumentTag", documentID, tagID)
	return s.err
}

func (s *stubWikiService) RemoveDocumentTag(_ context.Context, documentID, tagID string) error {
	s.rec("RemoveDocumentTag", documentID, tagID)
	return s.err
}

func (s *stubWikiService) ListDocumentTags(_ context.Context, documentID string) ([]*dto.DocumentTagResp, error) {
	s.rec("ListDocumentTags", documentID)
	if s.err != nil {
		return nil, s.err
	}
	return []*dto.DocumentTagResp{{DocumentID: documentID}}, nil
}

func (s *stubWikiService) CreateComment(_ context.Context, req *dto.CreateCommentReq) (*dto.CommentResp, error) {
	s.rec("CreateComment", req.DocumentID, req.Content)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.CommentResp{DocumentID: req.DocumentID, Content: req.Content}, nil
}

func (s *stubWikiService) ListComments(_ context.Context, documentID string) ([]*dto.CommentResp, error) {
	s.rec("ListComments", documentID)
	if s.err != nil {
		return nil, s.err
	}
	return []*dto.CommentResp{{DocumentID: documentID}}, nil
}

func (s *stubWikiService) UpdateComment(_ context.Context, id string, req *dto.UpdateCommentReq) error {
	s.rec("UpdateComment", id, req.Content)
	return s.err
}

func (s *stubWikiService) DeleteComment(_ context.Context, id string) error {
	s.rec("DeleteComment", id)
	return s.err
}

func (s *stubWikiService) CreateShareLink(_ context.Context, req *dto.ShareLinkReq) (*dto.ShareLinkResp, error) {
	s.rec("CreateShareLink", req.DocumentID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.ShareLinkResp{ID: "link-1", DocumentID: req.DocumentID, Token: "tok-1"}, nil
}

func (s *stubWikiService) GetShareLink(_ context.Context, token, password string) (*dto.ShareLinkResp, error) {
	s.rec("GetShareLink", token, password)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.ShareLinkResp{Token: token}, nil
}

func (s *stubWikiService) AccessSharedDocument(_ context.Context, token, password string) (*dto.SharedDocumentResp, error) {
	s.rec("AccessSharedDocument", token, password)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.SharedDocumentResp{DocumentID: "doc-1", Title: "分享标题", NeedPassword: password == ""}, nil
}

func (s *stubWikiService) ListShareLinks(_ context.Context, documentID string) ([]*dto.ShareLinkResp, error) {
	s.rec("ListShareLinks", documentID)
	if s.err != nil {
		return nil, s.err
	}
	return []*dto.ShareLinkResp{{ID: "link-1", DocumentID: documentID}}, nil
}

func (s *stubWikiService) DeleteShareLink(_ context.Context, id string) error {
	s.rec("DeleteShareLink", id)
	return s.err
}

func (s *stubWikiService) CreateTemplate(_ context.Context, req *dto.CreateTemplateReq) (*dto.DocumentTemplateResp, error) {
	s.rec("CreateTemplate", req.Name)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.DocumentTemplateResp{ID: "tpl-1", Name: req.Name}, nil
}

func (s *stubWikiService) ListTemplates(_ context.Context, category string) ([]*dto.DocumentTemplateResp, error) {
	s.rec("ListTemplates", category)
	if s.err != nil {
		return nil, s.err
	}
	return []*dto.DocumentTemplateResp{{ID: "tpl-1"}}, nil
}

func (s *stubWikiService) GetTemplate(_ context.Context, id string) (*dto.DocumentTemplateResp, error) {
	s.rec("GetTemplate", id)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.DocumentTemplateResp{ID: id}, nil
}

func (s *stubWikiService) UpdateTemplate(_ context.Context, id string, req *dto.UpdateTemplateReq) (*dto.DocumentTemplateResp, error) {
	s.rec("UpdateTemplate", id)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.DocumentTemplateResp{ID: id}, nil
}

func (s *stubWikiService) DeleteTemplate(_ context.Context, id string) error {
	s.rec("DeleteTemplate", id)
	return s.err
}

// ---------- stub：统计 / 日志 / 订阅 / 通知 / 锁 / 批量 / 对比 / 导入导出 ----------

func (s *stubWikiService) GetDocumentStats(_ context.Context, documentID string) (*dto.DocumentStatsResp, error) {
	s.rec("GetDocumentStats", documentID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.DocumentStatsResp{TotalViews: 42}, nil
}

func (s *stubWikiService) ListAccessLogs(_ context.Context, documentID string, page, pageSize int) ([]*dto.DocumentAccessLogResp, int64, error) {
	s.rec("ListAccessLogs", documentID, strconv.Itoa(page), strconv.Itoa(pageSize))
	if s.err != nil {
		return nil, 0, s.err
	}
	return []*dto.DocumentAccessLogResp{{DocumentID: documentID, Action: "view"}}, 1, nil
}

func (s *stubWikiService) SubscribeDocument(_ context.Context, documentID, notifyType string) (*dto.SubscriptionResp, error) {
	s.rec("SubscribeDocument", documentID, notifyType)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.SubscriptionResp{DocumentID: documentID, NotifyType: notifyType}, nil
}

func (s *stubWikiService) UnsubscribeDocument(_ context.Context, documentID string) error {
	s.rec("UnsubscribeDocument", documentID)
	return s.err
}

func (s *stubWikiService) ListUserSubscriptions(_ context.Context) ([]*dto.SubscriptionResp, error) {
	s.rec("ListUserSubscriptions")
	if s.err != nil {
		return nil, s.err
	}
	return []*dto.SubscriptionResp{{ID: "sub-1"}}, nil
}

func (s *stubWikiService) ListNotifications(_ context.Context, page, pageSize int) ([]*dto.NotificationResp, int64, error) {
	s.rec("ListNotifications", strconv.Itoa(page), strconv.Itoa(pageSize))
	if s.err != nil {
		return nil, 0, s.err
	}
	return []*dto.NotificationResp{{ID: "ntf-1", Title: "有人@你"}}, 1, nil
}

func (s *stubWikiService) MarkNotificationAsRead(_ context.Context, id string) error {
	s.rec("MarkNotificationAsRead", id)
	return s.err
}

func (s *stubWikiService) MarkAllNotificationsAsRead(_ context.Context) error {
	s.rec("MarkAllNotificationsAsRead")
	return s.err
}

func (s *stubWikiService) GetUnreadNotificationCount(_ context.Context) (int64, error) {
	s.rec("GetUnreadNotificationCount")
	if s.err != nil {
		return 0, s.err
	}
	return 5, nil
}

func (s *stubWikiService) AcquireEditLock(_ context.Context, documentID string) (*dto.EditLockResp, error) {
	s.rec("AcquireEditLock", documentID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.EditLockResp{DocumentID: documentID, UserID: testUser, CanEdit: true}, nil
}

func (s *stubWikiService) ReleaseEditLock(_ context.Context, documentID string) error {
	s.rec("ReleaseEditLock", documentID)
	return s.err
}

func (s *stubWikiService) RefreshEditLock(_ context.Context, documentID string) error {
	s.rec("RefreshEditLock", documentID)
	return s.err
}

func (s *stubWikiService) GetEditLock(_ context.Context, documentID string) (*dto.EditLockResp, error) {
	s.rec("GetEditLock", documentID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.EditLockResp{DocumentID: documentID, CanEdit: true}, nil
}

func (s *stubWikiService) BatchDeleteNodes(_ context.Context, nodeIDs []string) error {
	s.rec("BatchDeleteNodes", nodeIDs...)
	return s.err
}

func (s *stubWikiService) BatchMoveNodes(_ context.Context, nodeIDs []string, newParentID string) error {
	s.rec("BatchMoveNodes", append(append([]string{}, nodeIDs...), newParentID)...)
	return s.err
}

func (s *stubWikiService) CompareRevisions(_ context.Context, documentID string, version1, version2 int) (*dto.DiffResult, error) {
	s.rec("CompareRevisions", documentID, strconv.Itoa(version1), strconv.Itoa(version2))
	if s.err != nil {
		return nil, s.err
	}
	return &dto.DiffResult{OldVersion: version1, NewVersion: version2}, nil
}

func (s *stubWikiService) ExportDocument(_ context.Context, documentID, format string) ([]byte, string, error) {
	s.rec("ExportDocument", documentID, format)
	if s.err != nil {
		return nil, "", s.err
	}
	return []byte("# 导出内容"), "doc.md", nil
}

func (s *stubWikiService) ImportDocument(_ context.Context, documentID string, content []byte, format string) (*dto.DocumentResp, error) {
	s.rec("ImportDocument", documentID, format, string(content))
	if s.err != nil {
		return nil, s.err
	}
	return &dto.DocumentResp{ID: documentID}, nil
}

// ---------- 扩展路由与辅助 ----------

func newWikiExtendedRouter(stub *stubWikiService) http.Handler {
	h := handler.NewWikiHandlerExtended(stub)
	r := chi.NewRouter()
	r.Use(withIdentity)
	h.RegisterExtendedRoutes(r)
	return r
}

// public 分享路由：免登录
func newWikiShareRouter(stub *stubWikiService) http.Handler {
	h := handler.NewWikiHandlerExtended(stub)
	r := chi.NewRouter()
	r.Get("/wiki/share/{token}", h.AccessSharedDocument)
	return r
}

func (s *stubWikiService) assertNotCalled(t *testing.T, method string) {
	t.Helper()
	assert.NotContains(t, s.calls, method)
}

// ================= 标签 =================

func TestWikiCreateTag_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/spaces/tags", `{"name":"重要","color":"#f00"}`, nil)

	assertDataContains(t, w, `"id":"tag-1"`)
	assert.Equal(t, []string{"重要", "#f00"}, stub.params["CreateTag"])
}

func TestWikiListTags_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/spaces/tags", "", nil)

	assertDataContains(t, w, `"name":"重要"`)
	stub.assertCalled(t, "ListTags")
}

func TestWikiDeleteTag_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/wiki/spaces/tags/tag-1", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"tag-1"}, stub.params["DeleteTag"])
}

func TestWikiAddDocumentTag_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/documents/doc-1/tags/tag-1", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"doc-1", "tag-1"}, stub.params["AddDocumentTag"])
}

func TestWikiRemoveDocumentTag_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/wiki/documents/doc-1/tags/tag-1", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"doc-1", "tag-1"}, stub.params["RemoveDocumentTag"])
}

func TestWikiListDocumentTags_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/documents/doc-1/tags", "", nil)

	assertDataContains(t, w, `"document_id":"doc-1"`)
	assert.Equal(t, []string{"doc-1"}, stub.params["ListDocumentTags"])
}

// ================= 评论 =================

func TestWikiCreateComment_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/documents/doc-1/comments", `{"document_id":"doc-1","content":"说得对"}`, nil)

	assertDataContains(t, w, `"content":"说得对"`)
	assert.Equal(t, []string{"doc-1", "说得对"}, stub.params["CreateComment"])
}

func TestWikiCreateComment_ValidationFailure_Returns400(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/documents/doc-1/comments", `{"content":""}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestWikiListComments_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/documents/doc-1/comments", "", nil)

	assertDataContains(t, w, `"document_id":"doc-1"`)
	assert.Equal(t, []string{"doc-1"}, stub.params["ListComments"])
}

func TestWikiUpdateComment_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPut, "/wiki/documents/comments/c-1", `{"content":"新评论"}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"c-1", "新评论"}, stub.params["UpdateComment"])
}

func TestWikiDeleteComment_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/wiki/documents/comments/c-1", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"c-1"}, stub.params["DeleteComment"])
}

// ================= 分享 =================

func TestWikiCreateShareLink_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/documents/doc-1/share", `{"document_id":"doc-1","password":"123456"}`, nil)

	assertDataContains(t, w, `"token":"tok-1"`)
	assert.Equal(t, []string{"doc-1"}, stub.params["CreateShareLink"])
}

func TestWikiListShareLinks_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/documents/doc-1/shares", "", nil)

	assertDataContains(t, w, `"document_id":"doc-1"`)
	assert.Equal(t, []string{"doc-1"}, stub.params["ListShareLinks"])
}

func TestWikiDeleteShareLink_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/wiki/documents/shares/link-1", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"link-1"}, stub.params["DeleteShareLink"])
}

func TestWikiAccessSharedDocument_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiShareRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/share/tok-1", "", map[string]string{"password": ""})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"title":"分享标题"`)
	assert.Equal(t, []string{"tok-1", ""}, stub.params["AccessSharedDocument"])
}

func TestWikiAccessSharedDocument_Expired_Returns403(t *testing.T) {
	stub := newStubWiki().failWith(apperrors.ErrForbidden("分享链接已过期"))
	router := newWikiShareRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/share/tok-1", "", nil)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "分享链接已过期")
}

// ================= 模板 =================

func TestWikiCreateTemplate_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/spaces/templates", `{"name":"会议纪要","content":"# 会议"}`, nil)

	assertDataContains(t, w, `"id":"tpl-1"`)
	assert.Equal(t, []string{"会议纪要"}, stub.params["CreateTemplate"])
}

func TestWikiCreateTemplate_ValidationFailure_Returns400(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/spaces/templates", `{"name":"","content":""}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestWikiListTemplates_FiltersByCategory(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/spaces/templates", "", map[string]string{"category": "meeting"})

	assertDataContains(t, w, `"id":"tpl-1"`)
	assert.Equal(t, []string{"meeting"}, stub.params["ListTemplates"])
}

func TestWikiGetTemplate_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/spaces/templates/tpl-1", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"tpl-1"}, stub.params["GetTemplate"])
}

func TestWikiUpdateTemplate_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPut, "/wiki/spaces/templates/tpl-1", `{"name":"改后模板"}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"tpl-1"}, stub.params["UpdateTemplate"])
}

func TestWikiDeleteTemplate_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/wiki/spaces/templates/tpl-1", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"tpl-1"}, stub.params["DeleteTemplate"])
}

// ================= 文档统计 / 访问日志 / 订阅 =================

func TestWikiGetDocumentStats_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/documents/doc-1/stats", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"doc-1"}, stub.params["GetDocumentStats"])
}

func TestWikiListAccessLogs_PaginationDefaults(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/documents/doc-1/access-logs", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"doc-1", "1", "20"}, stub.params["ListAccessLogs"])
}

func TestWikiSubscribeDocument_DefaultNotifyTypeAll(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/documents/doc-1/subscribe", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"doc-1", "all"}, stub.params["SubscribeDocument"])
}

func TestWikiSubscribeDocument_WithNotifyType(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/documents/doc-1/subscribe", "", map[string]string{"notify_type": "comment"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"doc-1", "comment"}, stub.params["SubscribeDocument"])
}

func TestWikiUnsubscribeDocument_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/wiki/documents/doc-1/subscribe", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"doc-1"}, stub.params["UnsubscribeDocument"])
}

func TestWikiListUserSubscriptions_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/spaces/subscriptions", "", nil)

	assertDataContains(t, w, `"id":"sub-1"`)
	stub.assertCalled(t, "ListUserSubscriptions")
}

// ================= 通知 =================

func TestWikiListNotifications_PaginationDefaults(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/spaces/notifications", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"1", "20"}, stub.params["ListNotifications"])
}

func TestWikiMarkNotificationAsRead_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPut, "/wiki/spaces/notifications/ntf-1/read", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"ntf-1"}, stub.params["MarkNotificationAsRead"])
}

func TestWikiMarkAllNotificationsAsRead_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPut, "/wiki/spaces/notifications/read-all", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	stub.assertCalled(t, "MarkAllNotificationsAsRead")
}

func TestWikiGetUnreadNotificationCount_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/spaces/notifications/unread-count", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"count":5`)
}

// ================= 编辑锁 =================

func TestWikiAcquireEditLock_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/documents/doc-1/edit-lock", "", nil)

	assertDataContains(t, w, `"can_edit":true`)
	assert.Equal(t, []string{"doc-1"}, stub.params["AcquireEditLock"])
}

func TestWikiReleaseEditLock_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/wiki/documents/doc-1/edit-lock", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"doc-1"}, stub.params["ReleaseEditLock"])
}

func TestWikiRefreshEditLock_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPut, "/wiki/documents/doc-1/edit-lock", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"doc-1"}, stub.params["RefreshEditLock"])
}

func TestWikiGetEditLock_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/documents/doc-1/edit-lock", "", nil)

	assertDataContains(t, w, `"can_edit":true`)
	assert.Equal(t, []string{"doc-1"}, stub.params["GetEditLock"])
}

// ================= 批量 / 对比 / 导出 / 导入 =================

func TestWikiBatchDelete_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/spaces/nodes/batch", `{"action":"delete","node_ids":["n-1","n-2"]}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"n-1", "n-2"}, stub.params["BatchDeleteNodes"])
}

func TestWikiBatchMove_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/spaces/nodes/batch", `{"action":"move","node_ids":["n-1"],"target":"n-9"}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"n-1", "n-9"}, stub.params["BatchMoveNodes"])
}

func TestWikiBatch_InvalidAction_Returns400(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	// action 被 oneof=move delete 拦截，default 分支不可达
	w := doWiki(t, router, http.MethodPost, "/wiki/spaces/nodes/batch", `{"action":"explode","node_ids":["n-1"]}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestWikiBatch_MissingNodeIDs_Returns400(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/spaces/nodes/batch", `{"action":"delete"}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestWikiCompareRevisions_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/documents/doc-1/revisions/compare", "", map[string]string{"version1": "1", "version2": "3"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"doc-1", "1", "3"}, stub.params["CompareRevisions"])
}

func TestWikiCompareRevisions_MissingVersion_Returns400(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/wiki/documents/doc-1/revisions/compare", "", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "required")
	assert.Empty(t, stub.calls)
}

func TestWikiExportDocument_StreamsFile(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/documents/doc-1/export", `{"format":"markdown"}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "attachment; filename=doc.md", w.Header().Get("Content-Disposition"))
	assert.Equal(t, "# 导出内容", w.Body.String())
	assert.Equal(t, []string{"doc-1", "markdown"}, stub.params["ExportDocument"])
}

func TestWikiExportDocument_ValidationFailure_Returns400(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/documents/doc-1/export", `{"format":"exe"}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestWikiImportDocument_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "note.md")
	require.NoError(t, err)
	_, err = part.Write([]byte("# 导入的文档"))
	require.NoError(t, err)
	require.NoError(t, writer.WriteField("format", "markdown"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/wiki/documents/doc-1/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"document_id":"doc-1"`)
	assert.Equal(t, []string{"doc-1", "markdown", "# 导入的文档"}, stub.params["ImportDocument"])
}

func TestWikiImportDocument_MissingFile_Returns400(t *testing.T) {
	stub := newStubWiki()
	router := newWikiExtendedRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/wiki/documents/doc-1/import", "", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestWikiImportDocument_ServiceError_Returns500(t *testing.T) {
	stub := newStubWiki().failWith(apperrors.NewInternal("import failed"))
	router := newWikiExtendedRouter(stub)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "note.md")
	require.NoError(t, err)
	_, err = part.Write([]byte("# 内容"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/wiki/documents/doc-1/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "import failed")
}
