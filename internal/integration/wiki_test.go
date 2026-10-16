package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_Wiki_SpacesDocsShareSearch 整体验证 Wiki 模块（受保护组，仅需登录）：
//   - 空间 CRUD、节点创建、文档创建
//   - 分享链接创建 → 免登录公开访问落地页
//   - 全文搜索
//   - 负向：未登录 401、跨租户隔离（读不到他人空间）
func TestInteg_Wiki_SpacesDocsShareSearch(t *testing.T) {
	app := testkit.NewTestApp(t)
	seedTenantAdminRole(t, app.DB)

	const tenant = "t-wiki"
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/auth/register",
		`{"tenant_id":"`+tenant+`","username":"wikier123","password":"pass1234","nickname":"维基","email":"w@x.com"}`, "").Code)
	w := app.Do(t, http.MethodPost, "/api/v1/auth/login",
		`{"tenant_id":"`+tenant+`","username":"wikier123","password":"pass1234"}`, "")
	require.Equal(t, http.StatusOK, w.Code)
	auth := "Bearer " + decodeToken(t, w)

	// 未登录访问 → 401
	assert.Equal(t, http.StatusUnauthorized, app.Do(t, http.MethodGet, "/api/v1/wiki/spaces", "", "").Code)

	// 创建空间
	w = app.Do(t, http.MethodPost, "/api/v1/wiki/spaces", `{"name":"团队知识库","visibility":1}`, auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	spaceID := decodeID(t, w)

	// 列表/详情/更新
	w = app.Do(t, http.MethodGet, "/api/v1/wiki/spaces", "", auth)
	assert.Contains(t, w.Body.String(), spaceID)
	assert.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, "/api/v1/wiki/spaces/"+spaceID, "", auth).Code)
	assert.Equal(t, http.StatusOK, app.Do(t, http.MethodPut, "/api/v1/wiki/spaces/"+spaceID,
		`{"name":"团队知识库V2","visibility":1}`, auth).Code)

	// 创建文档节点（type=document 会自动创建文档记录）
	w = app.Do(t, http.MethodPost, "/api/v1/wiki/spaces/"+spaceID+"/nodes",
		`{"space_id":"`+spaceID+`","type":"document","title":"架构设计"}`, auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	nodeID := decodeID(t, w)

	// 读取文档并取得文档 ID，随后更新内容
	w = app.Do(t, http.MethodGet, "/api/v1/wiki/documents/nodes/"+nodeID, "", auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	docID := decodeID(t, w)
	assert.Equal(t, http.StatusOK, app.Do(t, http.MethodPut, "/api/v1/wiki/documents/"+docID,
		`{"content":"# 架构设计\n MeteorX 分层"}`, auth).Code)

	// 创建分享链接（URL 与 body 均带文档 ID）
	w = app.Do(t, http.MethodPost, "/api/v1/wiki/documents/"+docID+"/share",
		`{"document_id":"`+docID+`"}`, auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var share struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &share))
	require.NotEmpty(t, share.Data.Token, "分享应返回 token")

	// 免登录公开访问分享落地页
	w = app.Do(t, http.MethodGet, "/api/v1/wiki/share/"+share.Data.Token, "", "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 全文搜索
	w = app.Do(t, http.MethodGet, "/api/v1/wiki/search?q=%E6%9E%B6%E6%9E%84", "", auth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 跨租户隔离：另一租户用户读不到该空间
	otherAuth := app.AuthHeader("u-other-wiki", "t-other", []string{"member"})
	assert.NotEqual(t, http.StatusOK, app.Do(t, http.MethodGet, "/api/v1/wiki/spaces/"+spaceID, "", otherAuth).Code,
		"跨租户不应读到他人空间")
}
