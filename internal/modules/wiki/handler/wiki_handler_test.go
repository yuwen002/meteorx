package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/handler"
	"meteorx/internal/modules/wiki/service"
	apperrors "meteorx/internal/pkg/apperrors"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubWikiService 桩：嵌入 nil 接口自动满足 service.WikiServiceExtended；
// 只覆写被测方法，其余调用会 panic（测试路径不会触达）。
type stubWikiService struct {
	service.WikiServiceExtended

	err    error
	calls  []string
	params map[string][]string
}

func newStubWiki() *stubWikiService {
	return &stubWikiService{params: map[string][]string{}}
}

// rec 记录一次调用及其关键参数
func (s *stubWikiService) rec(method string, args ...string) {
	s.calls = append(s.calls, method)
	s.params[method] = args
}

// failWith 设置错误后返回 err（供测试复用桩做服务端错误分支）
func (s *stubWikiService) failWith(err error) *stubWikiService {
	s.err = err
	return s
}

// ---------- stub：空间 ----------

func (s *stubWikiService) CreateSpace(_ context.Context, tenantID, userID string, req *dto.CreateWikiSpaceReq) (*dto.WikiSpaceResp, error) {
	s.rec("CreateSpace", tenantID, userID, req.Name)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.WikiSpaceResp{ID: "sp-1", Name: req.Name}, nil
}

func (s *stubWikiService) GetSpace(_ context.Context, id, userID string) (*dto.WikiSpaceResp, error) {
	s.rec("GetSpace", id, userID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.WikiSpaceResp{ID: id, Name: "空间"}, nil
}

func (s *stubWikiService) ListSpaces(_ context.Context, tenantID, userID, keyword string, page, pageSize int) ([]*dto.WikiSpaceResp, int64, error) {
	s.rec("ListSpaces", tenantID, userID, keyword, strconv.Itoa(page), strconv.Itoa(pageSize))
	if s.err != nil {
		return nil, 0, s.err
	}
	return []*dto.WikiSpaceResp{{ID: "sp-1"}}, 1, nil
}

func (s *stubWikiService) UpdateSpace(_ context.Context, id, tenantID string, req *dto.UpdateWikiSpaceReq) (*dto.WikiSpaceResp, error) {
	s.rec("UpdateSpace", id, tenantID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.WikiSpaceResp{ID: id}, nil
}

func (s *stubWikiService) DeleteSpace(_ context.Context, id, tenantID string) error {
	s.rec("DeleteSpace", id, tenantID)
	return s.err
}

// ---------- stub：节点 ----------

func (s *stubWikiService) CreateNode(_ context.Context, spaceID, userID string, req *dto.CreateWikiNodeReq) (*dto.WikiNodeResp, error) {
	s.rec("CreateNode", spaceID, userID, req.Title, req.Type)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.WikiNodeResp{ID: "n-1", Title: req.Title}, nil
}

func (s *stubWikiService) GetNode(_ context.Context, id string) (*dto.WikiNodeResp, error) {
	s.rec("GetNode", id)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.WikiNodeResp{ID: id}, nil
}

func (s *stubWikiService) GetNodeTree(_ context.Context, spaceID string) ([]*dto.WikiNodeTreeResp, error) {
	s.rec("GetNodeTree", spaceID)
	if s.err != nil {
		return nil, s.err
	}
	return []*dto.WikiNodeTreeResp{{WikiNodeResp: dto.WikiNodeResp{ID: "n-1"}}}, nil
}

func (s *stubWikiService) UpdateNode(_ context.Context, id string, req *dto.UpdateWikiNodeReq) (*dto.WikiNodeResp, error) {
	s.rec("UpdateNode", id)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.WikiNodeResp{ID: id}, nil
}

func (s *stubWikiService) DeleteNode(_ context.Context, id string) error {
	s.rec("DeleteNode", id)
	return s.err
}

func (s *stubWikiService) MoveNode(_ context.Context, id, newParentID, userID string) error {
	s.rec("MoveNode", id, newParentID, userID)
	return s.err
}

func (s *stubWikiService) SortNode(_ context.Context, id string, sort int, userID string) error {
	s.rec("SortNode", id, strconv.Itoa(sort), userID)
	return s.err
}

// ---------- stub：文档 ----------

func (s *stubWikiService) CreateDocument(_ context.Context, nodeID, userID string, req *dto.CreateDocumentReq) (*dto.DocumentResp, error) {
	s.rec("CreateDocument", nodeID, userID, req.NodeID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.DocumentResp{ID: "doc-1", NodeID: nodeID}, nil
}

func (s *stubWikiService) GetDocument(_ context.Context, nodeID, userID string) (*dto.DocumentResp, error) {
	s.rec("GetDocument", nodeID, userID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.DocumentResp{ID: "doc-1", NodeID: nodeID}, nil
}

func (s *stubWikiService) UpdateDocument(_ context.Context, id, userID string, req *dto.UpdateDocumentReq) (*dto.DocumentResp, error) {
	s.rec("UpdateDocument", id, userID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.DocumentResp{ID: id}, nil
}

func (s *stubWikiService) DeleteDocument(_ context.Context, id string) error {
	s.rec("DeleteDocument", id)
	return s.err
}

// ---------- stub：版本 ----------

func (s *stubWikiService) ListRevisions(_ context.Context, documentID string) ([]*dto.DocumentRevisionResp, error) {
	s.rec("ListRevisions", documentID)
	if s.err != nil {
		return nil, s.err
	}
	return []*dto.DocumentRevisionResp{{ID: "rev-1", DocumentID: documentID}}, nil
}

func (s *stubWikiService) GetRevision(_ context.Context, documentID string, version int) (*dto.DocumentRevisionResp, error) {
	s.rec("GetRevision", documentID, strconv.Itoa(version))
	if s.err != nil {
		return nil, s.err
	}
	return &dto.DocumentRevisionResp{DocumentID: documentID, Version: version}, nil
}

func (s *stubWikiService) RestoreRevision(_ context.Context, documentID string, version int, userID string) (*dto.DocumentResp, error) {
	s.rec("RestoreRevision", documentID, strconv.Itoa(version), userID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.DocumentResp{ID: documentID}, nil
}

// ---------- stub：成员与节点权限 ----------

func (s *stubWikiService) AddMember(_ context.Context, spaceID string, req *dto.WikiSpaceMemberReq) (*dto.WikiSpaceMemberResp, error) {
	s.rec("AddMember", spaceID, req.UserID, req.Role)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.WikiSpaceMemberResp{SpaceID: spaceID, UserID: req.UserID, Role: req.Role}, nil
}

func (s *stubWikiService) RemoveMember(_ context.Context, spaceID, userID string) error {
	s.rec("RemoveMember", spaceID, userID)
	return s.err
}

func (s *stubWikiService) ListMembers(_ context.Context, spaceID string) ([]*dto.WikiSpaceMemberResp, error) {
	s.rec("ListMembers", spaceID)
	if s.err != nil {
		return nil, s.err
	}
	return []*dto.WikiSpaceMemberResp{{SpaceID: spaceID, UserID: "u-1", Role: "viewer"}}, nil
}

func (s *stubWikiService) SetNodePermission(_ context.Context, nodeID string, req *dto.SetNodePermissionReq) (*dto.NodePermissionResp, error) {
	s.rec("SetNodePermission", nodeID, req.UserID, req.Permission)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.NodePermissionResp{NodeID: nodeID, UserID: req.UserID, Permission: req.Permission}, nil
}

func (s *stubWikiService) GetNodePermissions(_ context.Context, nodeID string) ([]*dto.NodePermissionResp, error) {
	s.rec("GetNodePermissions", nodeID)
	if s.err != nil {
		return nil, s.err
	}
	return []*dto.NodePermissionResp{{NodeID: nodeID}}, nil
}

func (s *stubWikiService) RemoveNodePermission(_ context.Context, nodeID, userID, permission string) error {
	s.rec("RemoveNodePermission", nodeID, userID, permission)
	return s.err
}

// ---------- stub：回收站 / 搜索 / 附件 / 渲染 / 统计 ----------

func (s *stubWikiService) ListTrashItems(_ context.Context, tenantID, spaceID, itemType string, page, pageSize int) ([]*dto.TrashItemResp, int64, error) {
	s.rec("ListTrashItems", tenantID, spaceID, itemType, strconv.Itoa(page), strconv.Itoa(pageSize))
	if s.err != nil {
		return nil, 0, s.err
	}
	return []*dto.TrashItemResp{{ID: "trash-1"}}, 3, nil
}

func (s *stubWikiService) RestoreTrashItem(_ context.Context, id, userID string) error {
	s.rec("RestoreTrashItem", id, userID)
	return s.err
}

func (s *stubWikiService) PermanentDeleteTrashItem(_ context.Context, id, userID string) error {
	s.rec("PermanentDeleteTrashItem", id, userID)
	return s.err
}

func (s *stubWikiService) Search(_ context.Context, tenantID, userID, query, spaceID string, page, pageSize int) ([]*dto.SearchResultResp, int64, error) {
	s.rec("Search", tenantID, userID, query, spaceID, strconv.Itoa(page), strconv.Itoa(pageSize))
	if s.err != nil {
		return nil, 0, s.err
	}
	return []*dto.SearchResultResp{{ID: "doc-1", Title: query}}, 1, nil
}

func (s *stubWikiService) CreateAttachment(_ context.Context, userID string, req *dto.CreateAttachmentReq) (*dto.AttachmentResp, error) {
	s.rec("CreateAttachment", userID, req.DocumentID, req.FileName)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.AttachmentResp{ID: "att-1", FileName: req.FileName}, nil
}

func (s *stubWikiService) ListAttachments(_ context.Context, documentID, userID string) ([]*dto.AttachmentResp, error) {
	s.rec("ListAttachments", documentID, userID)
	if s.err != nil {
		return nil, s.err
	}
	return []*dto.AttachmentResp{{ID: "att-1", DocumentID: documentID}}, nil
}

func (s *stubWikiService) DeleteAttachment(_ context.Context, id, userID string) error {
	s.rec("DeleteAttachment", id, userID)
	return s.err
}

func (s *stubWikiService) RenderPreview(_ context.Context, content, format string) (string, error) {
	s.rec("RenderPreview", content, format)
	if s.err != nil {
		return "", s.err
	}
	return "<p>" + content + "</p>", nil
}

func (s *stubWikiService) GetStats(_ context.Context, tenantID string) (*dto.WikiStatsResp, error) {
	s.rec("GetStats", tenantID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.WikiStatsResp{TotalSpaces: 2, TotalDocuments: 10}, nil
}

func (s *stubWikiService) SetUploadSigner(uploadBaseURL, signKey string) {
	s.rec("SetUploadSigner", uploadBaseURL, signKey)
}

// ---------- HTTP 路由与请求辅助 ----------

const (
	testTenant = "t-1"
	testUser   = "u-1"
)

// withIdentity 注入模拟登录身份
func withIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := contextx.SetVars(r.Context(), testTenant, testUser, []string{"admin"})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// registerMainRoutes 注册与 routes.go 一致的主路由
func registerMainRoutes(r chi.Router, h *handler.WikiHandler) {
	r.Get("/stats", h.GetStats)
	r.Get("/search", h.Search)
	r.Get("/trash", h.ListTrashItems)
	r.Post("/trash/{id}/restore", h.RestoreTrashItem)
	r.Delete("/trash/{id}", h.PermanentDeleteTrashItem)

	r.Route("/spaces", func(r chi.Router) {
		r.Get("/", h.ListSpaces)
		r.Post("/", h.CreateSpace)
		r.Get("/{id}", h.GetSpace)
		r.Put("/{id}", h.UpdateSpace)
		r.Delete("/{id}", h.DeleteSpace)
		r.Route("/{spaceId}/nodes", func(r chi.Router) {
			r.Get("/tree", h.GetNodeTree)
			r.Post("/", h.CreateNode)
			r.Get("/{id}", h.GetNode)
			r.Put("/{id}", h.UpdateNode)
			r.Delete("/{id}", h.DeleteNode)
			r.Post("/{id}/move", h.MoveNode)
			r.Put("/{id}/sort", h.SortNode)
			r.Get("/{id}/permissions", h.GetNodePermissions)
			r.Post("/{id}/permissions", h.SetNodePermission)
			r.Delete("/{id}/permissions/{userId}/{permission}", h.RemoveNodePermission)
		})
		r.Route("/{spaceId}/members", func(r chi.Router) {
			r.Get("/", h.ListMembers)
			r.Post("/", h.AddMember)
			r.Delete("/{userId}", h.RemoveMember)
		})
	})

	r.Route("/documents", func(r chi.Router) {
		r.Post("/preview", h.PreviewMarkdown)
		r.Post("/nodes/{nodeId}", h.CreateDocument)
		r.Get("/nodes/{nodeId}", h.GetDocument)
		r.Put("/{id}", h.UpdateDocument)
		r.Delete("/{id}", h.DeleteDocument)
		r.Get("/{documentId}/revisions", h.ListRevisions)
		r.Get("/{documentId}/revisions/{version}", h.GetRevision)
		r.Post("/{documentId}/revisions/{version}/restore", h.RestoreRevision)
		r.Post("/attachments", h.CreateAttachment)
		r.Get("/{documentId}/attachments", h.ListAttachments)
		r.Delete("/attachments/{id}", h.DeleteAttachment)
	})
}

func newWikiRouter(stub *stubWikiService) http.Handler {
	h := handler.NewWikiHandler(stub)
	r := chi.NewRouter()
	r.Use(withIdentity)
	registerMainRoutes(r, h)
	return r
}

func doWiki(t *testing.T, router http.Handler, method, path, body string, query map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	}
	req.Header.Set("Content-Type", "application/json")
	if len(query) > 0 {
		q := req.URL.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// assertDataContains 断言 200 且响应 data 的 JSON 文本包含关键字
func assertDataContains(t *testing.T, w *httptest.ResponseRecorder, keyword string) {
	t.Helper()
	assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	var out struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Contains(t, string(out.Data), keyword)
}

func (s *stubWikiService) assertCalled(t *testing.T, method string) {
	t.Helper()
	assert.Contains(t, s.calls, method, "stub 应收到调用 %s", method)
}

// ================= 空间 =================

func TestWikiCreateSpace_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/spaces/", `{"name":"团队知识库","visibility":1}`, nil)

	assertDataContains(t, w, `"id":"sp-1"`)
	stub.assertCalled(t, "CreateSpace")
	assert.Equal(t, []string{testTenant, testUser, "团队知识库"}, stub.params["CreateSpace"])
}

func TestWikiCreateSpace_ValidationFailure_Returns400(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/spaces/", `{"name":"","visibility":9}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestWikiCreateSpace_ServiceError_Returns500(t *testing.T) {
	stub := newStubWiki().failWith(apperrors.NewInternal("space store failed"))
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/spaces/", `{"name":"团队知识库","visibility":1}`, nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "space store failed")
}

func TestWikiGetSpace_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/spaces/sp-1", "", nil)

	assertDataContains(t, w, `"name":"空间"`)
	assert.Equal(t, []string{"sp-1", testUser}, stub.params["GetSpace"])
}

func TestWikiGetSpace_NotFound_Returns404(t *testing.T) {
	stub := newStubWiki().failWith(apperrors.ErrNotFound("空间不存在"))
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/spaces/missing", "", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "空间不存在")
}

func TestWikiListSpaces_PaginationDefaultsAndContext(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/spaces/", "", map[string]string{"keyword": "知识"})

	assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), `"total":1`)
	assert.Equal(t, []string{testTenant, testUser, "知识", "1", "20"}, stub.params["ListSpaces"])
}

func TestWikiUpdateSpace_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPut, "/spaces/sp-1", `{"name":"改名后的空间"}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	// handler 将当前用户作为权限校验主体传给服务层
	assert.Equal(t, []string{"sp-1", testUser}, stub.params["UpdateSpace"])
}

func TestWikiDeleteSpace_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/spaces/sp-1", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"sp-1", testTenant}, stub.params["DeleteSpace"])
}

func TestWikiDeleteSpace_ServiceError_Returns500(t *testing.T) {
	stub := newStubWiki().failWith(apperrors.NewInternal("delete failed"))
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/spaces/sp-1", "", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ================= 节点 =================

func TestWikiCreateNode_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	// CreateWikiNodeReq.SpaceID 为 required（校验发生在 handler 使用 URL spaceId 之前），body 需携带
	w := doWiki(t, router, http.MethodPost, "/spaces/sp-1/nodes/", `{"space_id":"sp-1","type":"folder","title":"需求文档"}`, nil)

	assertDataContains(t, w, `"id":"n-1"`)
	assert.Equal(t, []string{"sp-1", testUser, "需求文档", "folder"}, stub.params["CreateNode"])
}

func TestWikiCreateNode_ValidationFailure_Returns400(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/spaces/sp-1/nodes/", `{"type":"unknown","title":""}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestWikiGetNodeTree_RequiresSpaceID(t *testing.T) {
	// tree 路由在 {spaceId} 缺失时不会命中；此处直接请求以验证注册
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/spaces/sp-1/nodes/tree", "", nil)

	assertDataContains(t, w, `"id":"n-1"`)
	assert.Equal(t, []string{"sp-1"}, stub.params["GetNodeTree"])
}

func TestWikiUpdateNode_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPut, "/spaces/sp-1/nodes/n-1", `{"title":"新标题"}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"n-1"}, stub.params["UpdateNode"])
}

func TestWikiDeleteNode_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/spaces/sp-1/nodes/n-1", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"n-1"}, stub.params["DeleteNode"])
}

func TestWikiMoveNode_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/spaces/sp-1/nodes/n-1/move", `{"new_parent_id":"n-2"}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"n-1", "n-2", testUser}, stub.params["MoveNode"])
}

func TestWikiMoveNode_BadBody_Returns400(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/spaces/sp-1/nodes/n-1/move", `{not-json`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestWikiSortNode_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPut, "/spaces/sp-1/nodes/n-1/sort", `{"sort":5}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"n-1", "5", testUser}, stub.params["SortNode"])
}

// ================= 文档 =================

func TestWikiCreateDocument_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	// CreateDocumentReq.NodeID 为 required，校验发生在 handler 覆写 nodeID 之前，body 需携带
	w := doWiki(t, router, http.MethodPost, "/documents/nodes/node-9", `{"node_id":"node-9","content":"# hi"}`, nil)

	assertDataContains(t, w, `"id":"doc-1"`)
	assert.Equal(t, []string{"node-9", testUser, "node-9"}, stub.params["CreateDocument"])
}

func TestWikiCreateDocument_ValidationFailure_Returns400(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/documents/nodes/node-9", `{}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestWikiGetDocument_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/documents/nodes/node-9", "", nil)

	assertDataContains(t, w, `"id":"doc-1"`)
	assert.Equal(t, []string{"node-9", testUser}, stub.params["GetDocument"])
}

func TestWikiUpdateDocument_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPut, "/documents/doc-1", `{"content":"v2","format":"markdown"}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"doc-1", testUser}, stub.params["UpdateDocument"])
}

func TestWikiDeleteDocument_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/documents/doc-1", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"doc-1"}, stub.params["DeleteDocument"])
}

// ================= 版本 =================

func TestWikiListRevisions_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/documents/doc-1/revisions", "", nil)

	assertDataContains(t, w, `"document_id":"doc-1"`)
	assert.Equal(t, []string{"doc-1"}, stub.params["ListRevisions"])
}

func TestWikiGetRevision_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/documents/doc-1/revisions/3", "", nil)

	assertDataContains(t, w, `"version":3`)
	assert.Equal(t, []string{"doc-1", "3"}, stub.params["GetRevision"])
}

func TestWikiRestoreRevision_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/documents/doc-1/revisions/2/restore", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"doc-1", "2", testUser}, stub.params["RestoreRevision"])
}

// ================= 成员与权限 =================

func TestWikiAddMember_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/spaces/sp-1/members/", `{"user_id":"u-9","role":"editor"}`, nil)

	assertDataContains(t, w, `"role":"editor"`)
	assert.Equal(t, []string{"sp-1", "u-9", "editor"}, stub.params["AddMember"])
}

func TestWikiAddMember_ValidationFailure_Returns400(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/spaces/sp-1/members/", `{"role":"owner"}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestWikiRemoveMember_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/spaces/sp-1/members/u-9", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"sp-1", "u-9"}, stub.params["RemoveMember"])
}

func TestWikiListMembers_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/spaces/sp-1/members/", "", nil)

	assertDataContains(t, w, `"user_id":"u-1"`)
	assert.Equal(t, []string{"sp-1"}, stub.params["ListMembers"])
}

func TestWikiSetNodePermission_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/spaces/sp-1/nodes/n-1/permissions", `{"user_id":"u-9","permission":"edit"}`, nil)

	assertDataContains(t, w, `"permission":"edit"`)
	assert.Equal(t, []string{"n-1", "u-9", "edit"}, stub.params["SetNodePermission"])
}

func TestWikiSetNodePermission_ValidationFailure_Returns400(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/spaces/sp-1/nodes/n-1/permissions", `{"user_id":"u-9","permission":"super"}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, stub.calls)
}

func TestWikiGetNodePermissions_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/spaces/sp-1/nodes/n-1/permissions", "", nil)

	assertDataContains(t, w, `"node_id":"n-1"`)
	assert.Equal(t, []string{"n-1"}, stub.params["GetNodePermissions"])
}

func TestWikiRemoveNodePermission_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/spaces/sp-1/nodes/n-1/permissions/u-9/view", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"n-1", "u-9", "view"}, stub.params["RemoveNodePermission"])
}

// ================= 回收站 / 搜索 / 附件 / 预览 / 统计 =================

func TestWikiListTrashItems_DefaultsAndPayload(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/trash", "", map[string]string{"space_id": "sp-1", "item_type": "document"})

	assert.Equal(t, http.StatusOK, w.Code)
	var out struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Contains(t, string(out.Data["items"]), `"trash-1"`)
	assert.Equal(t, "3", string(out.Data["total"]))
	assert.Contains(t, string(out.Data["page"]), "1")
	assert.Contains(t, string(out.Data["page_size"]), "20")
	assert.Equal(t, []string{testTenant, "sp-1", "document", "1", "20"}, stub.params["ListTrashItems"])
}

func TestWikiRestoreTrashItem_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/trash/trash-1/restore", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"trash-1", testUser}, stub.params["RestoreTrashItem"])
}

func TestWikiPermanentDeleteTrashItem_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/trash/trash-1", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"trash-1", testUser}, stub.params["PermanentDeleteTrashItem"])
}

func TestWikiSearch_DefaultsAndPayload(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/search", "", map[string]string{"q": "性能优化"})

	assert.Equal(t, http.StatusOK, w.Code)
	var out struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Contains(t, string(out.Data["results"]), `"doc-1"`)
	assert.Contains(t, string(out.Data["query"]), "性能优化")
	assert.Equal(t, []string{testTenant, testUser, "性能优化", "", "1", "20"}, stub.params["Search"])
}

func TestWikiCreateAttachment_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/documents/attachments", `{"document_id":"doc-1","file_name":"图.png","file_url":"/uploads/1.png"}`, nil)

	assertDataContains(t, w, `"id":"att-1"`)
	assert.Equal(t, []string{testUser, "doc-1", "图.png"}, stub.params["CreateAttachment"])
}

func TestWikiListAttachments_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/documents/doc-1/attachments", "", nil)

	assertDataContains(t, w, `"document_id":"doc-1"`)
	assert.Equal(t, []string{"doc-1", testUser}, stub.params["ListAttachments"])
}

func TestWikiDeleteAttachment_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodDelete, "/documents/attachments/att-1", "", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"att-1", testUser}, stub.params["DeleteAttachment"])
}

func TestWikiPreviewMarkdown_Success(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodPost, "/documents/preview", `{"content":"# 标题","format":"markdown"}`, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"content_html"`)
	assert.Contains(t, w.Body.String(), "# 标题")
	assert.Equal(t, []string{"# 标题", "markdown"}, stub.params["RenderPreview"])
}

func TestWikiGetStats_ContextTenant(t *testing.T) {
	stub := newStubWiki()
	router := newWikiRouter(stub)

	w := doWiki(t, router, http.MethodGet, "/stats", "", nil)

	assertDataContains(t, w, `"total_documents":10`)
	assert.Equal(t, []string{testTenant}, stub.params["GetStats"])
}
