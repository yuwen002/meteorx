package service

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"

	dbpkg "meteorx/internal/pkg/db"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// notifDBSeq 用于为各测试用例分配独立的内存 SQLite 实例（避免共享 cache 导致串数据）。
var notifDBSeq uint64

func uniqueMemoryDSN() string {
	return fmt.Sprintf("file:wiki_notif_%d:?mode=memory&cache=shared", atomic.AddUint64(&notifDBSeq, 1))
}

// setupNotificationTest 构建基于真实内存 SQLite 的扩展服务与基础数据（空间/节点/文档/成员/订阅），
// 用于验证通知生成逻辑（绕开权限与 HTTP 层）。
func setupNotificationTest(t *testing.T) (*wikiServiceExtended, repository.WikiRepositoryExtended, *gorm.DB, context.Context, *model.Document, *model.WikiNode) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(uniqueMemoryDSN()), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	require.NoError(t, repository.AutoMigrate(db))

	extRepo := repository.NewWikiRepositoryExtended(db)
	tx := dbpkg.NewTxManager(db)
	svc := NewWikiServiceExtended(extRepo, tx).(*wikiServiceExtended)
	ctx := contextx.SetVars(context.Background(), "tenant-1", "owner", nil)

	space := &model.WikiSpace{ID: "space-1", TenantID: "tenant-1", Name: "空间", Visibility: model.VisibilityTenant, CreatedBy: "owner"}
	require.NoError(t, db.Create(space).Error)
	node := &model.WikiNode{ID: "node-1", TenantID: "tenant-1", SpaceID: "space-1", Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, db.Create(node).Error)
	doc := &model.Document{ID: "doc-1", TenantID: "tenant-1", NodeID: "node-1", Content: "c", Format: "markdown", CurrentVer: 1, LastEditedBy: "owner"}
	require.NoError(t, db.Create(doc).Error)

	require.NoError(t, db.Create(&model.WikiSpaceMember{ID: "m-owner", SpaceID: "space-1", TenantID: "tenant-1", UserID: "owner", Role: model.SpaceRoleOwner}).Error)
	require.NoError(t, db.Create(&model.WikiSpaceMember{ID: "m-admin", SpaceID: "space-1", TenantID: "tenant-1", UserID: "admin", Role: model.SpaceRoleAdmin}).Error)
	require.NoError(t, db.Create(&model.WikiSpaceMember{ID: "m-viewer", SpaceID: "space-1", TenantID: "tenant-1", UserID: "viewer", Role: model.SpaceRoleViewer}).Error)

	return svc, extRepo, db, ctx, doc, node
}

// TestNotifyReviewers 提交审核时应通知空间内 Owner/Admin（排除提交者本人），Viewer 无权审核不通知。
func TestNotifyReviewers(t *testing.T) {
	svc, extRepo, _, ctx, doc, node := setupNotificationTest(t)

	svc.notifyReviewers(ctx, doc, node, "owner")

	adminNotifs, _, err := extRepo.ListNotifications(ctx, "admin", 1, 10)
	require.NoError(t, err)
	require.Len(t, adminNotifs, 1)
	assert.Equal(t, model.NotificationTypeReview, adminNotifs[0].Type)

	viewerNotifs, _, _ := extRepo.ListNotifications(ctx, "viewer", 1, 10)
	assert.Empty(t, viewerNotifs, "Viewer 无审核权限，不应收到待审通知")

	ownerNotifs, _, _ := extRepo.ListNotifications(ctx, "owner", 1, 10)
	assert.Empty(t, ownerNotifs, "提交者本人应被排除")
}

// TestNotifyAuthorReviewResult 审核通过/驳回时应通知文档作者（LastEditedBy），审核人即作者时不自通知。
func TestNotifyAuthorReviewResult(t *testing.T) {
	svc, extRepo, _, ctx, doc, node := setupNotificationTest(t)

	svc.notifyAuthorReviewResult(ctx, doc, node, "admin", "通过")
	authorNotifs, _, err := extRepo.ListNotifications(ctx, "owner", 1, 10)
	require.NoError(t, err)
	require.Len(t, authorNotifs, 1)
	assert.Equal(t, model.NotificationTypeReview, authorNotifs[0].Type)

	// 审核人与作者相同：不产生新通知
	svc.notifyAuthorReviewResult(ctx, doc, node, "owner", "通过")
	authorNotifs, _, _ = extRepo.ListNotifications(ctx, "owner", 1, 10)
	assert.Len(t, authorNotifs, 1, "审核人即作者时不应产生新通知")
}

// TestNotifySubscribers 评论/编辑事件均经统一 notifySubscribers 通知订阅者，
// 验证：订阅类型过滤（all/comment/edit）、作者排除。
func TestNotifySubscribers(t *testing.T) {
	svc, extRepo, db, ctx, doc, _ := setupNotificationTest(t)

	require.NoError(t, db.Create(&model.DocumentSubscription{ID: "sub-1", DocumentID: doc.ID, TenantID: "tenant-1", NodeID: "node-1", UserID: "subAll", NotifyType: model.NotifyTypeAll}).Error)
	require.NoError(t, db.Create(&model.DocumentSubscription{ID: "sub-2", DocumentID: doc.ID, TenantID: "tenant-1", NodeID: "node-1", UserID: "commenter", NotifyType: model.NotifyTypeComment}).Error)
	require.NoError(t, db.Create(&model.DocumentSubscription{ID: "sub-3", DocumentID: doc.ID, TenantID: "tenant-1", NodeID: "node-1", UserID: "editor2", NotifyType: model.NotifyTypeEdit}).Error)

	// 评论事件：应通知 subAll 与 commenter，不通知 editor2；作者 owner 排除
	svc.notifySubscribers(ctx, doc.ID, "owner", model.NotifyTypeComment, model.NotificationTypeComment, "你订阅的文档有新评论", "评论内容", "comment-1", "comment")

	subAllNotifs, _, err := extRepo.ListNotifications(ctx, "subAll", 1, 10)
	require.NoError(t, err)
	require.Len(t, subAllNotifs, 1)
	assert.Equal(t, model.NotificationTypeComment, subAllNotifs[0].Type)
	assert.Equal(t, "comment-1", subAllNotifs[0].RelatedID)

	commenterNotifs, _, _ := extRepo.ListNotifications(ctx, "commenter", 1, 10)
	require.Len(t, commenterNotifs, 1)

	editorNotifs, _, _ := extRepo.ListNotifications(ctx, "editor2", 1, 10)
	assert.Empty(t, editorNotifs, "edit-only 订阅者不应收到评论通知")

	authorNotifs, _, _ := extRepo.ListNotifications(ctx, "owner", 1, 10)
	assert.Empty(t, authorNotifs, "作者本人应被排除")

	// 编辑事件：应通知 subAll 与 editor2，不通知 commenter
	svc.notifySubscribers(ctx, doc.ID, "owner", model.NotifyTypeEdit, model.NotificationTypeEdit, "你订阅的文档有更新", "文档标题", doc.ID, "document")

	editor2Notifs, _, _ := extRepo.ListNotifications(ctx, "editor2", 1, 10)
	require.Len(t, editor2Notifs, 1, "edit-only 订阅者应收到编辑通知")
	assert.Equal(t, model.NotificationTypeEdit, editor2Notifs[0].Type)

	commenterEditNotifs, _, _ := extRepo.ListNotifications(ctx, "commenter", 1, 10)
	assert.Len(t, commenterEditNotifs, 1, "commenter 在编辑事件中不应新增通知（仍只有之前的评论通知）")
}
