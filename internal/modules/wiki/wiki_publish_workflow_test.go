package wiki

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/handler"
	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"
	"meteorx/internal/modules/wiki/service"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	publishTestTenant = "tenant-pub"
	publishTestOwner  = "u-owner"
	publishTestEditor = "u-editor"
	publishTestViewer = "u-viewer"
)

// TestInteg_WikiPublishWorkflow 真实 HTTP 校验文档发布工作流的状态机与角色权限：
// 提交审核 → 审核通过/驳回 → 直接发布/取消发布/归档，逐级验证 403 与 409 边界。
// 同时回归「review/publish 未映射到 document:review/document:publish 导致全员 403」的缺陷。
func TestInteg_WikiPublishWorkflow(t *testing.T) {
	dsn := "file:wiki-publish-test?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, repository.AutoMigrate(db))

	seedPublishFixture(t, db)

	r := chi.NewRouter()
	// 用请求头模拟认证中间件注入的租户/用户上下文
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := contextx.SetVars(req.Context(), publishTestTenant, req.Header.Get("X-Test-User"), nil)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := handler.NewWikiHandlerExtended(service.NewWikiServiceExtended(repository.NewWikiRepositoryExtended(db), nil))
	h.RegisterExtendedRoutes(r)

	do := func(method, path, user, body string) *httptest.ResponseRecorder {
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		req := httptest.NewRequest(method, path, reader)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User", user)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	status := func(w *httptest.ResponseRecorder) string {
		var resp struct {
			Data struct {
				PublishStatus string `json:"publish_status"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), w.Body.String())
		return resp.Data.PublishStatus
	}
	countNotifications := func(userID, titleLike string) int64 {
		var n int64
		require.NoError(t, db.Model(&model.Notification{}).
			Where("user_id = ? AND title LIKE ?", userID, "%"+titleLike+"%").
			Count(&n).Error)
		return n
	}

	t.Run("只读成员提交审核被拒", func(t *testing.T) {
		w := do(http.MethodPost, "/wiki/documents/doc-1/submit-review", publishTestViewer, "{}")
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	})

	t.Run("协作者提交审核进入 pending_review 并通知审核人", func(t *testing.T) {
		w := do(http.MethodPost, "/wiki/documents/doc-1/submit-review", publishTestEditor, `{"comment":"请审核版式"}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, model.PublishStatusPendingReview, status(w))
		assert.EqualValues(t, 1, countNotifications(publishTestOwner, "待你审核"))
	})

	t.Run("只读成员无权审核", func(t *testing.T) {
		w := do(http.MethodPost, "/wiki/documents/doc-1/approve", publishTestViewer, "{}")
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	})

	t.Run("协作者审核通过（document:review）", func(t *testing.T) {
		w := do(http.MethodPost, "/wiki/documents/doc-1/approve", publishTestEditor, `{"comment":"通过"}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, model.PublishStatusPublished, status(w))
		// 文档最后编辑者即审核人本人时不再自通知
		assert.EqualValues(t, 0, countNotifications(publishTestEditor, "审核"))
	})

	t.Run("空间所有者取消发布回到草稿", func(t *testing.T) {
		w := do(http.MethodPost, "/wiki/documents/doc-1/unpublish", publishTestOwner, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, model.PublishStatusDraft, status(w))
	})

	t.Run("协作者无发布权限", func(t *testing.T) {
		w := do(http.MethodPost, "/wiki/documents/doc-1/publish", publishTestEditor, "")
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	})

	t.Run("草稿态不可直接发布（状态机冲突）", func(t *testing.T) {
		w := do(http.MethodPost, "/wiki/documents/doc-1/publish", publishTestOwner, "")
		assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	})

	t.Run("待审文档可从审核队列直接发布", func(t *testing.T) {
		require.Equal(t, http.StatusOK, do(http.MethodPost, "/wiki/documents/doc-1/submit-review", publishTestEditor, "{}").Code)

		w := do(http.MethodGet, "/wiki/pending-reviews", publishTestOwner, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var list struct {
			Data struct {
				Total int64 `json:"total"`
				Items []struct {
					DocumentID string `json:"document_id"`
					SpaceName  string `json:"space_name"`
				} `json:"items"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list), w.Body.String())
		require.EqualValues(t, 1, list.Data.Total, w.Body.String())
		require.Len(t, list.Data.Items, 1)
		assert.Equal(t, "doc-1", list.Data.Items[0].DocumentID)
		assert.Equal(t, "产品空间", list.Data.Items[0].SpaceName)

		w = do(http.MethodPost, "/wiki/documents/doc-1/publish", publishTestOwner, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, model.PublishStatusPublished, status(w))

		// 发布后不再出现在待审队列
		w = do(http.MethodGet, "/wiki/pending-reviews", publishTestOwner, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		list.Data.Total, list.Data.Items = 0, nil
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list), w.Body.String())
		assert.EqualValues(t, 0, list.Data.Total)
	})

	t.Run("已发布文档重复发布返回冲突", func(t *testing.T) {
		w := do(http.MethodPost, "/wiki/documents/doc-1/publish", publishTestOwner, "")
		assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	})

	t.Run("只读成员无权归档", func(t *testing.T) {
		w := do(http.MethodPost, "/wiki/documents/doc-1/archive", publishTestViewer, "")
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	})

	t.Run("空间所有者归档已发布文档", func(t *testing.T) {
		w := do(http.MethodPost, "/wiki/documents/doc-1/archive", publishTestOwner, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, model.PublishStatusArchived, status(w))

		// 归档态不可审核通过
		w = do(http.MethodPost, "/wiki/documents/doc-1/approve", publishTestOwner, "{}")
		assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	})

	t.Run("归档态取消发布回到草稿", func(t *testing.T) {
		w := do(http.MethodPost, "/wiki/documents/doc-1/unpublish", publishTestOwner, "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, model.PublishStatusDraft, status(w))
	})

	t.Run("驳回流程记录结论并通知作者", func(t *testing.T) {
		require.Equal(t, http.StatusOK, do(http.MethodPost, "/wiki/documents/doc-1/submit-review", publishTestEditor, "{}").Code)

		w := do(http.MethodPost, "/wiki/documents/doc-1/reject", publishTestOwner, `{"comment":"缺少图例"}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, model.PublishStatusRejected, status(w))
		assert.EqualValues(t, 1, countNotifications(publishTestEditor, "驳回"))

		// 驳回后重新提交并审核通过
		require.Equal(t, http.StatusOK, do(http.MethodPost, "/wiki/documents/doc-1/submit-review", publishTestEditor, "{}").Code)
		w = do(http.MethodPost, "/wiki/documents/doc-1/approve", publishTestOwner, "{}")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, model.PublishStatusPublished, status(w))
	})
}

// seedPublishFixture 构造空间、文档节点、草稿文档与三种角色的成员关系。
func seedPublishFixture(t *testing.T, db *gorm.DB) {
	t.Helper()

	require.NoError(t, db.Create(&model.WikiSpace{
		ID: "space-1", TenantID: publishTestTenant, Name: "产品空间",
		Visibility: model.VisibilityPrivate, CreatedBy: publishTestOwner,
	}).Error)
	require.NoError(t, db.Create(&model.WikiNode{
		ID: "node-1", TenantID: publishTestTenant, SpaceID: "space-1",
		Type: model.NodeTypeDocument, Title: "发布规范", OwnerID: publishTestOwner, Status: model.NodeStatusActive,
	}).Error)
	require.NoError(t, db.Create(&model.Document{
		ID: "doc-1", TenantID: publishTestTenant, NodeID: "node-1",
		Content: "body", Format: "markdown",
		PublishStatus: model.PublishStatusDraft, LastEditedBy: publishTestEditor,
	}).Error)

	members := []*model.WikiSpaceMember{
		{ID: "m-owner", TenantID: publishTestTenant, SpaceID: "space-1", UserID: publishTestOwner, Role: model.SpaceRoleOwner},
		{ID: "m-editor", TenantID: publishTestTenant, SpaceID: "space-1", UserID: publishTestEditor, Role: model.SpaceRoleEditor},
		{ID: "m-viewer", TenantID: publishTestTenant, SpaceID: "space-1", UserID: publishTestViewer, Role: model.SpaceRoleViewer},
	}
	for _, m := range members {
		require.NoError(t, db.Create(m).Error)
	}
}
