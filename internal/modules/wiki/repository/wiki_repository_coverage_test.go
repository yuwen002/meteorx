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

// TestIntegration_ConcurrentIncrementViewCount 验证多协程并发自增浏览量时最终计数准确（原子 UPDATE）。
func TestIntegration_ConcurrentIncrementViewCount(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	space := newSpace(t, repo, ctx, "并发浏览量空间")
	node := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, node))
	doc := &model.Document{TenantID: "tenant-1", NodeID: node.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))

	const n = 50
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = repo.IncrementViewCount(ctx, doc.ID)
		}()
	}
	wg.Wait()

	got, err := repo.GetDocumentByID(ctx, doc.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(n), got.ViewCount, "并发自增后浏览量应精确等于请求次数")
}

// TestIntegration_ConcurrentDocumentUpdate 验证多写者并发基于同一版本号编辑文档时，
// 乐观锁保证恰好一个提交成功、其余因版本冲突失败，且最终版本与内容为胜出者。
func TestIntegration_ConcurrentDocumentUpdate(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	space := newSpace(t, repo, ctx, "并发编辑空间")
	node := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, node))
	doc := &model.Document{TenantID: "tenant-1", NodeID: node.ID, Content: "v1", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))

	writers := []struct {
		content string
		editor  string
	}{
		{"来自写者0的内容", "user-0"},
		{"来自写者1的内容", "user-1"},
	}
	const count = 2
	var wg sync.WaitGroup
	errs := make([]error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			d := &model.Document{
				ID:           doc.ID,
				TenantID:     "tenant-1",
				NodeID:       node.ID,
				Content:      writers[i].content,
				ContentHTML:  "<p>" + writers[i].content,
				Format:       "markdown",
				CurrentVer:   2,
				LastEditedBy: writers[i].editor,
			}
			errs[i] = repo.UpdateDocument(ctx, d, 1) // 均基于版本 1 提交
		}(i)
	}
	wg.Wait()

	okCount, conflictCount := 0, 0
	winnerContent := ""
	for i, e := range errs {
		if e == nil {
			okCount++
			winnerContent = writers[i].content
		}
		if errors.Is(e, ErrDocumentVersionConflict) {
			conflictCount++
		}
	}
	assert.Equal(t, 1, okCount, "恰好一个写者成功提交")
	assert.Equal(t, count-1, conflictCount, "其余写者因版本冲突失败")

	got, err := repo.GetDocumentByID(ctx, doc.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.CurrentVer)
	assert.Equal(t, winnerContent, got.Content)
}

// TestIntegration_ExpireTrashItems_OnlyExpired 验证过期清理仅清除已过期项、不触碰未过期项。
// 该用例锁定“定时任务重复/误清未到期数据”的风险。
func TestIntegration_ExpireTrashItems_OnlyExpired(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	// 过期文档：软删文档并写入已过期回收站记录
	space := newSpace(t, repo, ctx, "过期清理空间")
	node := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, node))
	doc := &model.Document{TenantID: "tenant-1", NodeID: node.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))
	require.NoError(t, repo.DeleteDocument(ctx, doc.ID))
	expiredDoc := &model.TrashItem{TenantID: "tenant-1", ItemType: model.TrashTypeDocument, ItemID: doc.ID, SpaceID: space.ID, Title: "文档", DeletedBy: "user-1", ExpiresAt: time.Now().Add(-time.Hour)}
	require.NoError(t, repo.CreateTrashItem(ctx, expiredDoc))

	// 未过期空间：空间仍在，写入未来过期回收站记录
	futureSpace := newSpace(t, repo, ctx, "未过期空间")
	futureTrash := &model.TrashItem{TenantID: "tenant-1", ItemType: model.TrashTypeSpace, ItemID: futureSpace.ID, SpaceID: futureSpace.ID, Title: "空间", DeletedBy: "user-1", ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, repo.CreateTrashItem(ctx, futureTrash))

	require.NoError(t, repo.ExpireTrashItems(ctx))

	// 过期文档被物理清除，其回收站记录被清理
	var d model.Document
	assert.ErrorIs(t, repo.db.Unscoped().Where("id = ?", doc.ID).First(&d).Error, gorm.ErrRecordNotFound)
	_, err := repo.GetTrashItem(ctx, expiredDoc.ID)
	assert.Error(t, err)

	// 未过期空间保持原样（仍可查询），其回收站记录保留
	_, err = repo.GetSpaceByID(ctx, futureSpace.ID)
	assert.NoError(t, err)
	_, err = repo.GetTrashItem(ctx, futureTrash.ID)
	assert.NoError(t, err)
}

// TestIntegration_ExpireTrashItems_Idempotent 验证过期清理可重复执行：第二次为 no-op 或容忍已清除实体，均不报错。
func TestIntegration_ExpireTrashItems_Idempotent(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	space := newSpace(t, repo, ctx, "幂等空间")
	node := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, node))
	doc := &model.Document{TenantID: "tenant-1", NodeID: node.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))
	require.NoError(t, repo.DeleteDocument(ctx, doc.ID))
	item := &model.TrashItem{TenantID: "tenant-1", ItemType: model.TrashTypeDocument, ItemID: doc.ID, SpaceID: space.ID, Title: "文档", DeletedBy: "user-1", ExpiresAt: time.Now().Add(-time.Hour)}
	require.NoError(t, repo.CreateTrashItem(ctx, item))

	// 连续两次执行过期清理，均不应报错（第二次为 no-op 或容忍已清除实体）
	require.NoError(t, repo.ExpireTrashItems(ctx))
	require.NoError(t, repo.ExpireTrashItems(ctx))

	var d model.Document
	assert.ErrorIs(t, repo.db.Unscoped().Where("id = ?", doc.ID).First(&d).Error, gorm.ErrRecordNotFound)
}

// TestIntegration_ExpireTrashItems_DanglingItem 验证回收站记录指向已缺失实体时，过期清理忽略并返回成功（不中断任务）。
func TestIntegration_ExpireTrashItems_DanglingItem(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	// 指向不存在实体的已过期回收站记录（实体已被外部提前物理清除）
	dangling := &model.TrashItem{TenantID: "tenant-1", ItemType: model.TrashTypeDocument, ItemID: "ghost-doc", SpaceID: "ghost-space", Title: "幽灵", DeletedBy: "user-1", ExpiresAt: time.Now().Add(-time.Hour)}
	require.NoError(t, repo.CreateTrashItem(ctx, dangling))

	// 即使实体缺失，过期清理也应忽略并返回成功
	require.NoError(t, repo.ExpireTrashItems(ctx))

	// 缺失实体对应的回收站记录仍应被清理
	_, err := repo.GetTrashItem(ctx, dangling.ID)
	assert.Error(t, err)
}

// TestIntegration_ExpireTrashItems_AllTypes 验证过期清理对文档/节点/空间三类回收站记录均正确级联物理清除。
func TestIntegration_ExpireTrashItems_AllTypes(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")

	// 文档类
	dSpace := newSpace(t, repo, ctx, "文档型空间")
	dNode := &model.WikiNode{SpaceID: dSpace.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, dNode))
	doc := &model.Document{TenantID: "tenant-1", NodeID: dNode.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))
	require.NoError(t, repo.DeleteDocument(ctx, doc.ID))
	docTrash := &model.TrashItem{TenantID: "tenant-1", ItemType: model.TrashTypeDocument, ItemID: doc.ID, SpaceID: dSpace.ID, Title: "文档", DeletedBy: "user-1", ExpiresAt: time.Now().Add(-time.Hour)}
	require.NoError(t, repo.CreateTrashItem(ctx, docTrash))

	// 节点类（含文档子节点）
	nSpace := newSpace(t, repo, ctx, "节点型空间")
	nFolder := &model.WikiNode{SpaceID: nSpace.ID, Type: model.NodeTypeFolder, Title: "目录", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, nFolder))
	nDocNode := &model.WikiNode{SpaceID: nSpace.ID, ParentID: nFolder.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, nDocNode))
	nDoc := &model.Document{TenantID: "tenant-1", NodeID: nDocNode.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, nDoc))
	require.NoError(t, repo.DeleteNode(ctx, nDocNode.ID))
	require.NoError(t, repo.DeleteNode(ctx, nFolder.ID))
	nodeTrash := &model.TrashItem{TenantID: "tenant-1", ItemType: model.TrashTypeNode, ItemID: nFolder.ID, SpaceID: nSpace.ID, Title: "目录", DeletedBy: "user-1", ExpiresAt: time.Now().Add(-time.Hour)}
	require.NoError(t, repo.CreateTrashItem(ctx, nodeTrash))

	// 空间类
	sSpace := newSpace(t, repo, ctx, "空间型空间")
	sFolder := &model.WikiNode{SpaceID: sSpace.ID, Type: model.NodeTypeFolder, Title: "目录", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, sFolder))
	sDocNode := &model.WikiNode{SpaceID: sSpace.ID, ParentID: sFolder.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, sDocNode))
	sDoc := &model.Document{TenantID: "tenant-1", NodeID: sDocNode.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, sDoc))
	require.NoError(t, repo.SoftDeleteSpaceTree(ctx, sSpace.ID))
	require.NoError(t, repo.DeleteSpace(ctx, sSpace.ID))
	spaceTrash := &model.TrashItem{TenantID: "tenant-1", ItemType: model.TrashTypeSpace, ItemID: sSpace.ID, SpaceID: sSpace.ID, Title: "空间", DeletedBy: "user-1", ExpiresAt: time.Now().Add(-time.Hour)}
	require.NoError(t, repo.CreateTrashItem(ctx, spaceTrash))

	require.NoError(t, repo.ExpireTrashItems(ctx))

	// 三类实体均被物理清除，回收站记录均被清理
	var d model.Document
	assert.ErrorIs(t, repo.db.Unscoped().Where("id = ?", doc.ID).First(&d).Error, gorm.ErrRecordNotFound)
	var nodeCount int64
	require.NoError(t, repo.db.Unscoped().Model(&model.WikiNode{}).Where("id IN ?", []string{nFolder.ID, nDocNode.ID}).Count(&nodeCount).Error)
	assert.Equal(t, int64(0), nodeCount)
	var sp model.WikiSpace
	assert.ErrorIs(t, repo.db.Unscoped().Where("id = ?", sSpace.ID).First(&sp).Error, gorm.ErrRecordNotFound)

	for _, id := range []string{docTrash.ID, nodeTrash.ID, spaceTrash.ID} {
		_, err := repo.GetTrashItem(ctx, id)
		assert.Error(t, err)
	}
}
