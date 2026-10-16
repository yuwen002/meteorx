package integration

import (
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_Notification_AnnouncementCRUD 整体验证公告模块：
// 超管创建/列表/详情/改状态/删除闭环；非超管创建被 403；租户侧公告列表可访问。
func TestInteg_Notification_AnnouncementCRUD(t *testing.T) {
	app := testkit.NewTestApp(t)
	sa := app.SuperadminHeader("sa", "t-1")
	member := app.AuthHeader("m", "t-1", []string{"member"})

	body := `{"title":"系统升级","content":"将于今晚升级","scope":"all","status":1}`
	w := app.Do(t, http.MethodPost, "/api/v1/admin/announcements", body, sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	id := decodeID(t, w)

	w = app.Do(t, http.MethodGet, "/api/v1/admin/announcements", "", sa)
	assert.Contains(t, w.Body.String(), id)
	w = app.Do(t, http.MethodGet, "/api/v1/admin/announcements/"+id, "", sa)
	assert.Equal(t, http.StatusOK, w.Code)
	w = app.Do(t, http.MethodPut, "/api/v1/admin/announcements/"+id+"/status", `{"status":2}`, sa)
	assert.Equal(t, http.StatusOK, w.Code)
	w = app.Do(t, http.MethodDelete, "/api/v1/admin/announcements/"+id, "", sa)
	assert.Equal(t, http.StatusOK, w.Code)

	// 非超管创建公告 → 403
	w = app.Do(t, http.MethodPost, "/api/v1/admin/announcements", body, member)
	assert.Equal(t, http.StatusForbidden, w.Code, "非超管创建公告应 403")

	// 租户侧公告列表（公开侧，需登录）→ 200
	w = app.Do(t, http.MethodGet, "/api/v1/announcements", "", member)
	assert.Equal(t, http.StatusOK, w.Code)
}
