package repository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"meteorx/internal/modules/wiki/model"

	"gorm.io/gorm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSpace 便捷创建测试空间，返回已持久化的空间。
func newSpace(t *testing.T, repo *wikiRepository, ctx context.Context, name string) *model.WikiSpace {
	t.Helper()
	sp := &model.WikiSpace{Name: name, Visibility: model.VisibilityTenant, CreatedBy: "user-1"}
	require.NoError(t, repo.CreateSpace(ctx, sp))
	return sp
}

// TestIntegration_RestoreDocument 验证文档软删后可通过 RestoreDocument 真实还原，
// 且对不存在的文档返回 ErrDocumentNotFound。
func TestIntegration_RestoreDocument(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	space := newSpace(t, repo, ctx, "恢复文档空间")
	node := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, node))
	doc := &model.Document{TenantID: "tenant-1", NodeID: node.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))

	require.NoError(t, repo.DeleteDocument(ctx, doc.ID))
	_, err := repo.GetDocumentByNodeID(ctx, node.ID)
	assert.Error(t, err)

	require.NoError(t, repo.RestoreDocument(ctx, doc.ID))
	got, err := repo.GetDocumentByID(ctx, doc.ID)
	require.NoError(t, err)
	assert.Equal(t, doc.ID, got.ID)

	// 不存在的文档恢复返回未找到错误
	assert.ErrorIs(t, repo.RestoreDocument(ctx, "missing-doc"), ErrDocumentNotFound)
}

// TestIntegration_RestoreNodeTree 验证删除目录后，RestoreNodeTree 能级联还原整棵子树
// （含嵌套的子目录、文档类节点及其文档内容）。
func TestIntegration_RestoreNodeTree(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	space := newSpace(t, repo, ctx, "恢复节点树空间")
	folder := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeFolder, Title: "根目录", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, folder))
	subFolder := &model.WikiNode{SpaceID: space.ID, ParentID: folder.ID, Type: model.NodeTypeFolder, Title: "子目录", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, subFolder))
	docNode := &model.WikiNode{SpaceID: space.ID, ParentID: folder.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 2}
	require.NoError(t, repo.CreateNode(ctx, docNode))
	doc := &model.Document{TenantID: "tenant-1", NodeID: docNode.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))
	grandDocNode := &model.WikiNode{SpaceID: space.ID, ParentID: subFolder.ID, Type: model.NodeTypeDocument, Title: "孙文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, grandDocNode))
	grandDoc := &model.Document{TenantID: "tenant-1", NodeID: grandDocNode.ID, Content: "gc", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, grandDoc))

	// 模拟删除目录：软删全部节点与文档
	for _, n := range []*model.WikiNode{docNode, grandDocNode} {
		d, err := repo.GetDocumentByNodeID(ctx, n.ID)
		require.NoError(t, err)
		require.NoError(t, repo.DeleteDocument(ctx, d.ID))
	}
	for _, id := range []string{docNode.ID, grandDocNode.ID, subFolder.ID, folder.ID} {
		require.NoError(t, repo.DeleteNode(ctx, id))
	}
	nodes, err := repo.ListNodesBySpace(ctx, space.ID)
	require.NoError(t, err)
	assert.Empty(t, nodes)

	// 恢复根节点即级联恢复整棵子树
	require.NoError(t, repo.RestoreNodeTree(ctx, folder.ID))

	nodes, err = repo.ListNodesBySpace(ctx, space.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 4)
	_, err = repo.GetDocumentByNodeID(ctx, docNode.ID)
	assert.NoError(t, err)
	_, err = repo.GetDocumentByNodeID(ctx, grandDocNode.ID)
	assert.NoError(t, err)
}

// TestIntegration_PurgeDocument 验证 PurgeDocument 物理清除文档及其版本/附件，并清理对应回收站记录。
func TestIntegration_PurgeDocument(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	space := newSpace(t, repo, ctx, "清空文档空间")
	node := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, node))
	doc := &model.Document{TenantID: "tenant-1", NodeID: node.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))
	rev := &model.DocumentRevision{TenantID: "tenant-1", DocumentID: doc.ID, Version: 1, Content: "c", ContentHTML: "<p>", Summary: "s", EditedBy: "user-1"}
	require.NoError(t, repo.CreateRevision(ctx, rev))
	att := &model.Attachment{TenantID: "tenant-1", DocumentID: doc.ID, FileID: "file-1", FileName: "a.txt"}
	require.NoError(t, repo.CreateAttachment(ctx, att))
	trash := &model.TrashItem{TenantID: "tenant-1", ItemType: model.TrashTypeDocument, ItemID: doc.ID, SpaceID: space.ID, Title: "文档", DeletedBy: "user-1"}
	require.NoError(t, repo.CreateTrashItem(ctx, trash))

	require.NoError(t, repo.DeleteDocument(ctx, doc.ID))
	require.NoError(t, repo.PurgeDocument(ctx, doc.ID))

	var d model.Document
	assert.ErrorIs(t, repo.db.Unscoped().Where("id = ?", doc.ID).First(&d).Error, gorm.ErrRecordNotFound)
	var rv model.DocumentRevision
	assert.ErrorIs(t, repo.db.Unscoped().Where("document_id = ?", doc.ID).First(&rv).Error, gorm.ErrRecordNotFound)
	var at model.Attachment
	assert.ErrorIs(t, repo.db.Unscoped().Where("document_id = ?", doc.ID).First(&at).Error, gorm.ErrRecordNotFound)
	_, err := repo.GetTrashItem(ctx, trash.ID)
	assert.Error(t, err)
}

// TestIntegration_PurgeNodeTree 验证 PurgeNodeTree 物理清除节点子树（含文档）并清理其回收站记录。
func TestIntegration_PurgeNodeTree(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	space := newSpace(t, repo, ctx, "清空节点树空间")
	folder := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeFolder, Title: "根目录", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, folder))
	docNode := &model.WikiNode{SpaceID: space.ID, ParentID: folder.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, docNode))
	doc := &model.Document{TenantID: "tenant-1", NodeID: docNode.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))
	trashNode := &model.TrashItem{TenantID: "tenant-1", ItemType: model.TrashTypeNode, ItemID: folder.ID, SpaceID: space.ID, Title: "根目录", DeletedBy: "user-1"}
	trashDoc := &model.TrashItem{TenantID: "tenant-1", ItemType: model.TrashTypeDocument, ItemID: doc.ID, SpaceID: space.ID, Title: "文档", DeletedBy: "user-1"}
	require.NoError(t, repo.CreateTrashItem(ctx, trashNode))
	require.NoError(t, repo.CreateTrashItem(ctx, trashDoc))

	require.NoError(t, repo.PurgeNodeTree(ctx, folder.ID))

	var n model.WikiNode
	assert.ErrorIs(t, repo.db.Unscoped().Where("id = ?", folder.ID).First(&n).Error, gorm.ErrRecordNotFound)
	assert.ErrorIs(t, repo.db.Unscoped().Where("id = ?", docNode.ID).First(&n).Error, gorm.ErrRecordNotFound)
	var d model.Document
	assert.ErrorIs(t, repo.db.Unscoped().Where("id = ?", doc.ID).First(&d).Error, gorm.ErrRecordNotFound)
	_, err := repo.GetTrashItem(ctx, trashNode.ID)
	assert.Error(t, err)
	_, err = repo.GetTrashItem(ctx, trashDoc.ID)
	assert.Error(t, err)
}

// TestIntegration_PurgeSpaceTree 验证 PurgeSpaceTree 物理清除空间及其下全部节点/文档，并清理空间回收站记录。
func TestIntegration_PurgeSpaceTree(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	space := newSpace(t, repo, ctx, "清空空间空间")
	folder := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeFolder, Title: "根目录", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, folder))
	docNode := &model.WikiNode{SpaceID: space.ID, ParentID: folder.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, docNode))
	doc := &model.Document{TenantID: "tenant-1", NodeID: docNode.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))
	trash := &model.TrashItem{TenantID: "tenant-1", ItemType: model.TrashTypeSpace, ItemID: space.ID, SpaceID: space.ID, Title: "空间", DeletedBy: "user-1"}
	require.NoError(t, repo.CreateTrashItem(ctx, trash))

	// 服务层语义：先批量级联软删子树，再软删空间
	require.NoError(t, repo.SoftDeleteSpaceTree(ctx, space.ID))
	require.NoError(t, repo.DeleteSpace(ctx, space.ID))
	require.NoError(t, repo.PurgeSpaceTree(ctx, space.ID))

	var sp model.WikiSpace
	assert.ErrorIs(t, repo.db.Unscoped().Where("id = ?", space.ID).First(&sp).Error, gorm.ErrRecordNotFound)
	var nodeCount int64
	require.NoError(t, repo.db.Unscoped().Model(&model.WikiNode{}).Where("space_id = ?", space.ID).Count(&nodeCount).Error)
	assert.Equal(t, int64(0), nodeCount)
	var d model.Document
	assert.ErrorIs(t, repo.db.Unscoped().Where("id = ?", doc.ID).First(&d).Error, gorm.ErrRecordNotFound)
	_, err := repo.GetTrashItem(ctx, trash.ID)
	assert.Error(t, err)
}

// TestIntegration_ExpireTrashItems 验证过期回收站记录触发对应实体物理清除并清理记录。
func TestIntegration_ExpireTrashItems(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	space := newSpace(t, repo, ctx, "过期清理空间")
	node := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, node))
	doc := &model.Document{TenantID: "tenant-1", NodeID: node.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))

	// 软删文档并写入已过期回收站记录
	require.NoError(t, repo.DeleteDocument(ctx, doc.ID))
	item := &model.TrashItem{
		TenantID:  "tenant-1",
		ItemType:  model.TrashTypeDocument,
		ItemID:    doc.ID,
		SpaceID:   space.ID,
		Title:     "文档",
		DeletedBy: "user-1",
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	require.NoError(t, repo.CreateTrashItem(ctx, item))

	require.NoError(t, repo.ExpireTrashItems(ctx))

	var d model.Document
	assert.ErrorIs(t, repo.db.Unscoped().Where("id = ?", doc.ID).First(&d).Error, gorm.ErrRecordNotFound)
	_, err := repo.GetTrashItem(ctx, item.ID)
	assert.Error(t, err)
}

// TestIntegration_GetWikiStats 验证租户级统计正确，且删除空间（级联软删子树）后统计归零，
// 规避“空间已删但子树仍被计入”的统计偏差。
func TestIntegration_GetWikiStats(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	empty, err := repo.GetWikiStats(ctx, "tenant-empty")
	require.NoError(t, err)
	assert.Equal(t, int64(0), empty.TotalSpaces)
	assert.Equal(t, int64(0), empty.TotalNodes)
	assert.Equal(t, int64(0), empty.TotalDocuments)
	assert.Equal(t, int64(0), empty.TotalViews)

	space := newSpace(t, repo, ctx, "统计空间")
	node := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeFolder, Title: "目录", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, node))
	doc := &model.Document{TenantID: "tenant-1", NodeID: node.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))
	require.NoError(t, repo.IncrementViewCount(ctx, doc.ID))
	require.NoError(t, repo.IncrementViewCount(ctx, doc.ID))

	stats, err := repo.GetWikiStats(ctx, "tenant-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.TotalSpaces)
	assert.Equal(t, int64(1), stats.TotalNodes)
	assert.Equal(t, int64(1), stats.TotalDocuments)
	assert.Equal(t, int64(2), stats.TotalViews)

	// 删除空间（级联软删子树）后统计应归零
	require.NoError(t, repo.SoftDeleteSpaceTree(ctx, space.ID))
	require.NoError(t, repo.DeleteSpace(ctx, space.ID))
	stats, err = repo.GetWikiStats(ctx, "tenant-1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalSpaces)
	assert.Equal(t, int64(0), stats.TotalNodes)
	assert.Equal(t, int64(0), stats.TotalDocuments)
	assert.Equal(t, int64(0), stats.TotalViews)
}

// TestIntegration_Search 验证节点标题与文档内容搜索（含空间限定与租户隔离）。
func TestIntegration_Search(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	space := newSpace(t, repo, ctx, "搜索空间")
	n1 := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeDocument, Title: "季度报告", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, n1))
	n2 := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeFolder, Title: "其他目录", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, n2))
	doc := &model.Document{TenantID: "tenant-1", NodeID: n1.ID, Content: "这是季度总结内容", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))

	// 标题搜索（租户级）
	nodes, err := repo.SearchNodesByTitle(ctx, "tenant-1", "", "季度")
	require.NoError(t, err)
	assert.Len(t, nodes, 1)
	assert.Equal(t, n1.ID, nodes[0].ID)

	// 标题搜索（限定空间）
	nodes, err = repo.SearchNodesByTitle(ctx, "tenant-1", space.ID, "目录")
	require.NoError(t, err)
	assert.Len(t, nodes, 1)
	assert.Equal(t, n2.ID, nodes[0].ID)

	// 文档内容搜索（租户级 / 限定空间）
	docs, err := repo.SearchDocumentsByContent(ctx, "tenant-1", "", "季度总结")
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	docs, err = repo.SearchDocumentsByContent(ctx, "tenant-1", space.ID, "季度总结")
	require.NoError(t, err)
	assert.Len(t, docs, 1)

	// 租户隔离：另一租户查不到
	docs, err = repo.SearchDocumentsByContent(ctx, "tenant-2", "", "季度总结")
	require.NoError(t, err)
	assert.Empty(t, docs)
}

// TestIntegration_Revisions 验证文档历史版本的创建、列表（倒序）与按版本查询。
func TestIntegration_Revisions(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	space := newSpace(t, repo, ctx, "版本空间")
	node := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, node))
	doc := &model.Document{TenantID: "tenant-1", NodeID: node.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))

	r1 := &model.DocumentRevision{TenantID: "tenant-1", DocumentID: doc.ID, Version: 1, Content: "v1", ContentHTML: "<p>v1", Summary: "s1", EditedBy: "u1"}
	r2 := &model.DocumentRevision{TenantID: "tenant-1", DocumentID: doc.ID, Version: 2, Content: "v2", ContentHTML: "<p>v2", Summary: "s2", EditedBy: "u1"}
	require.NoError(t, repo.CreateRevision(ctx, r1))
	require.NoError(t, repo.CreateRevision(ctx, r2))

	revs, err := repo.ListRevisions(ctx, doc.ID)
	require.NoError(t, err)
	require.Len(t, revs, 2)
	assert.Equal(t, 2, revs[0].Version, "应按版本号倒序")
	assert.Equal(t, 1, revs[1].Version)

	got, err := repo.GetRevisionByVersion(ctx, doc.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "v1", got.Content)

	_, err = repo.GetRevisionByVersion(ctx, doc.ID, 99)
	assert.ErrorIs(t, err, ErrRevisionNotFound)
}

// TestIntegration_Attachments 验证附件的增删查与按文档批量清理。
func TestIntegration_Attachments(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	space := newSpace(t, repo, ctx, "附件空间")
	node := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, node))
	doc := &model.Document{TenantID: "tenant-1", NodeID: node.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))

	a1 := &model.Attachment{TenantID: "tenant-1", DocumentID: doc.ID, FileID: "f1", FileName: "a1.txt"}
	a2 := &model.Attachment{TenantID: "tenant-1", DocumentID: doc.ID, FileID: "f2", FileName: "a2.txt"}
	require.NoError(t, repo.CreateAttachment(ctx, a1))
	require.NoError(t, repo.CreateAttachment(ctx, a2))

	atts, err := repo.ListAttachmentsByDocument(ctx, doc.ID)
	require.NoError(t, err)
	assert.Len(t, atts, 2)

	got, err := repo.GetAttachment(ctx, a1.ID)
	require.NoError(t, err)
	assert.Equal(t, "a1.txt", got.FileName)

	require.NoError(t, repo.DeleteAttachment(ctx, a1.ID))
	atts, err = repo.ListAttachmentsByDocument(ctx, doc.ID)
	require.NoError(t, err)
	assert.Len(t, atts, 1)

	require.NoError(t, repo.DeleteAttachmentsByDocument(ctx, doc.ID))
	atts, err = repo.ListAttachmentsByDocument(ctx, doc.ID)
	require.NoError(t, err)
	assert.Empty(t, atts)
}

// TestIntegration_NodePermissions 验证节点权限的增、查、按用户查、改、按节点清空。
func TestIntegration_NodePermissions(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	space := newSpace(t, repo, ctx, "权限空间")
	node := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeFolder, Title: "目录", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, node))

	p1 := &model.WikiNodePermission{NodeID: node.ID, UserID: "u1", Permission: model.NodePermEdit}
	require.NoError(t, repo.CreateNodePermission(ctx, p1))
	p2 := &model.WikiNodePermission{NodeID: node.ID, UserID: "u2", Permission: model.NodePermView}
	require.NoError(t, repo.CreateNodePermission(ctx, p2))

	perms, err := repo.GetNodePermissions(ctx, node.ID)
	require.NoError(t, err)
	assert.Len(t, perms, 2)

	userPerms, err := repo.GetUserNodePermissions(ctx, node.ID, "u1")
	require.NoError(t, err)
	require.Len(t, userPerms, 1)
	assert.Equal(t, model.NodePermEdit, userPerms[0].Permission)

	perm, err := repo.GetNodePermission(ctx, node.ID, "u2", model.NodePermView)
	require.NoError(t, err)
	require.NotNil(t, perm)

	perm.Permission = model.NodePermEdit
	require.NoError(t, repo.UpdateNodePermission(ctx, perm))
	perm, err = repo.GetNodePermission(ctx, node.ID, "u2", model.NodePermEdit)
	require.NoError(t, err)
	require.NotNil(t, perm)

	require.NoError(t, repo.DeleteNodePermissionsByNode(ctx, node.ID))
	perms, err = repo.GetNodePermissions(ctx, node.ID)
	require.NoError(t, err)
	assert.Empty(t, perms)
}

// TestIntegration_MiscQueries 验证 ListSpaces（租户/可见性/关键词过滤）、CountChildNodes、
// GetMember（含不存在返回 nil）与 GetDocumentByID 等行为。
func TestIntegration_MiscQueries(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	s1 := newSpace(t, repo, ctx, "Alpha空间")
	s2 := &model.WikiSpace{Name: "Beta空间", Visibility: model.VisibilityPrivate, CreatedBy: "user-1"}
	require.NoError(t, repo.CreateSpace(ctx, s2))
	s3 := &model.WikiSpace{Name: "Other", Visibility: model.VisibilityTenant, CreatedBy: "user-2"}
	require.NoError(t, repo.CreateSpace(testCtx("tenant-2", "user-2"), s3))

	// 不带 userID：返回租户内全部空间
	spaces, total, err := repo.ListSpaces(ctx, "tenant-1", "", "", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, spaces, 2)

	// 带 userID 且非成员：私有空间被排除（仅返回租户可见的 s1）
	spaces, total, err = repo.ListSpaces(ctx, "tenant-1", "user-9", "", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, s1.ID, spaces[0].ID)

	// 关键词过滤
	spaces, total, err = repo.ListSpaces(ctx, "tenant-1", "", "Alpha", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 子节点计数
	node := &model.WikiNode{SpaceID: s1.ID, Type: model.NodeTypeFolder, Title: "目录", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, node))
	child := &model.WikiNode{SpaceID: s1.ID, ParentID: node.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, child))
	cnt, err := repo.CountChildNodes(ctx, node.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), cnt)
	cnt, err = repo.CountChildNodes(ctx, child.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), cnt)

	// 成员查询：存在 / 不存在返回 nil
	require.NoError(t, repo.AddMember(ctx, &model.WikiSpaceMember{SpaceID: s1.ID, UserID: "user-1", Role: model.SpaceRoleOwner}))
	m, err := repo.GetMember(ctx, s1.ID, "user-1")
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, model.SpaceRoleOwner, m.Role)
	m, err = repo.GetMember(ctx, s1.ID, "nobody")
	require.NoError(t, err)
	assert.Nil(t, m)

	// GetDocumentByID
	doc := &model.Document{TenantID: "tenant-1", NodeID: node.ID, Content: "x", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))
	got, err := repo.GetDocumentByID(ctx, doc.ID)
	require.NoError(t, err)
	assert.Equal(t, doc.ID, got.ID)
}
