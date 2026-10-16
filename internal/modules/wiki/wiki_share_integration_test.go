package wiki

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// TestInteg_WikiPublicShare 真实 HTTP 校验 Wiki 免登录公开分享的安全属性。
// 通过 RegisterPublicShareRoutes 挂载真实 handler，直接驱动 GET /wiki/share/{token}。
func TestInteg_WikiPublicShare(t *testing.T) {
	dsn := "file:wiki-share-test?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.WikiSpace{}, &model.WikiNode{}, &model.Document{}, &model.DocumentRevision{},
		&model.WikiSpaceMember{}, &model.WikiNodePermission{}, &model.TrashItem{}, &model.Attachment{},
		&model.Tag{}, &model.DocumentTag{}, &model.Comment{}, &model.ShareLink{},
		&model.DocumentTemplate{}, &model.DocumentAccessLog{}, &model.DocumentSubscription{},
		&model.Notification{}, &model.EditLock{}, &model.ReviewComment{},
	))

	repo := repository.NewWikiRepositoryExtended(db)
	svc := service.NewWikiServiceExtended(repo, nil)
	h := handler.NewWikiHandlerExtended(svc)

	r := chi.NewRouter()
	RegisterPublicShareRoutes(r, h) // GET /wiki/share/{token}

	const tenantID = "t-1"
	require.NoError(t, db.Create(&model.Document{
		ID: "doc-1", TenantID: tenantID, NodeID: "node-1",
		Content: "shared body", ContentHTML: "<p>shared</p>", Format: "markdown",
	}).Error)

	getShare := func(token, password string) *httptest.ResponseRecorder {
		url := "/wiki/share/" + token
		if password != "" {
			url += "?password=" + password
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
		return w
	}

	newLink := func(token, password string, expireAt *time.Time, maxViews, viewCount int) {
		require.NoError(t, db.Create(&model.ShareLink{
			ID: "link-" + token, TenantID: tenantID, DocumentID: "doc-1", NodeID: "node-1",
			Token: token, Password: password, ExpireAt: expireAt, MaxViews: maxViews, ViewCount: viewCount, CreatedBy: "u-1",
		}).Error)
	}

	t.Run("合法无密码访问返回文档", func(t *testing.T) {
		newLink("tok-open", "", nil, 0, 0)
		w := getShare("tok-open", "")
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var resp struct {
			Data struct {
				Content string `json:"content"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "shared body", resp.Data.Content)
	})

	t.Run("需要密码但未提供 -> 非200且提示需要密码", func(t *testing.T) {
		newLink("tok-needpwd", "s3cret", nil, 0, 0)
		w := getShare("tok-needpwd", "")
		assert.NotEqual(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "需要密码")
	})

	t.Run("密码错误 -> 非200且提示密码错误", func(t *testing.T) {
		newLink("tok-wrongpwd", "s3cret", nil, 0, 0)
		w := getShare("tok-wrongpwd", "wrong")
		assert.NotEqual(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "密码错误")
	})

	t.Run("已过期 -> 非200且提示已过期", func(t *testing.T) {
		past := time.Now().Add(-time.Hour)
		newLink("tok-expired", "", &past, 0, 0)
		w := getShare("tok-expired", "")
		assert.NotEqual(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "已过期")
	})

	t.Run("令牌不存在 -> 非200且提示不存在", func(t *testing.T) {
		w := getShare("tok-nope", "")
		assert.NotEqual(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "不存在")
	})

	t.Run("达到最大访问次数 -> 非200且提示最大访问次数", func(t *testing.T) {
		newLink("tok-maxviews", "", nil, 1, 1)
		w := getShare("tok-maxviews", "")
		assert.NotEqual(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "最大访问次数")
	})

	t.Run("跨租户令牌不可访问其他租户文档", func(t *testing.T) {
		require.NoError(t, db.Create(&model.ShareLink{
			ID: "link-cross", TenantID: "t-2", DocumentID: "doc-other", NodeID: "n-2",
			Token: "tok-cross", MaxViews: 0, ViewCount: 0, CreatedBy: "u-1",
		}).Error)
		w := getShare("tok-cross", "")
		assert.NotEqual(t, http.StatusOK, w.Code)
		// 文档在该租户不存在 -> 同样以"不存在"拒绝（租户隔离）
		assert.Contains(t, w.Body.String(), "不存在")
	})
}
