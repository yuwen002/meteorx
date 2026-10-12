package repository

import (
	"context"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupIntegrationTest 创建内存 SQLite 数据库并返回 repository
func setupIntegrationTest(t *testing.T) (*wikiRepository, func()) {
	t.Helper()

	db, err := setupSQLiteMemory()
	require.NoError(t, err)

	repo := &wikiRepository{db: db}
	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}

	return repo, cleanup
}

// testCtx 创建包含租户和用户信息的测试 context
func testCtx(tenantID, userID string) context.Context {
	ctx := context.Background()
	return contextx.SetVars(ctx, tenantID, userID, []string{"admin"})
}

// TestIntegration_SpaceCRUD 验证 Space 的完整 CRUD 流程
func TestIntegration_SpaceCRUD(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := testCtx("tenant-1", "user-1")

	// 1. 创建 Space
	space := &model.WikiSpace{
		Name:        "测试空间",
		Description: "用于集成测试",
		Icon:        "📁",
		Visibility:  model.VisibilityTenant,
		CreatedBy:   "user-1",
	}
	require.NoError(t, repo.CreateSpace(ctx, space))
	assert.NotEmpty(t, space.ID)
	assert.Equal(t, "tenant-1", space.TenantID)
	assert.Equal(t, "测试空间", space.Name)

	// 2. 查询 Space
	got, err := repo.GetSpaceByID(ctx, space.ID)
	require.NoError(t, err)
	assert.Equal(t, space.ID, got.ID)
	assert.Equal(t, "测试空间", got.Name)

	// 3. 更新 Space
	space.Name = "更新后的空间"
	space.Description = "已更新"
	require.NoError(t, repo.UpdateSpace(ctx, space))
	got, err = repo.GetSpaceByID(ctx, space.ID)
	require.NoError(t, err)
	assert.Equal(t, "更新后的空间", got.Name)
	assert.Equal(t, "已更新", got.Description)

	// 4. 删除 Space（软删）
	require.NoError(t, repo.DeleteSpace(ctx, space.ID))
	_, err = repo.GetSpaceByID(ctx, space.ID)
	assert.Error(t, err)
}

// TestIntegration_NodeCRUD 验证 Node 的完整 CRUD 流程
func TestIntegration_NodeCRUD(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := testCtx("tenant-1", "user-1")

	// 先创建 Space
	space := &model.WikiSpace{
		Name:       "节点测试空间",
		Visibility: model.VisibilityTenant,
		CreatedBy:  "user-1",
	}
	require.NoError(t, repo.CreateSpace(ctx, space))

	// 1. 创建根节点
	node := &model.WikiNode{
		SpaceID: space.ID,
		Type:    model.NodeTypeFolder,
		Title:   "根目录",
		Sort:    1,
	}
	require.NoError(t, repo.CreateNode(ctx, node))
	assert.NotEmpty(t, node.ID)

	// 2. 创建子节点
	child := &model.WikiNode{
		SpaceID:  space.ID,
		ParentID: node.ID,
		Type:     model.NodeTypeDocument,
		Title:    "子文档",
		Sort:     1,
	}
	require.NoError(t, repo.CreateNode(ctx, child))

	// 3. 查询节点列表（ListNodesBySpace 返回空间内全部节点，按 sort 排序）
	nodes, err := repo.ListNodesBySpace(ctx, space.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
	assert.Equal(t, node.ID, nodes[0].ID)
	assert.Equal(t, child.ID, nodes[1].ID)

	// 查询子节点
	children, err := repo.ListChildNodes(ctx, node.ID)
	require.NoError(t, err)
	assert.Len(t, children, 1)
	assert.Equal(t, child.ID, children[0].ID)

	// 4. 更新节点
	child.Title = "更新后的文档"
	require.NoError(t, repo.UpdateNode(ctx, child))
	got, err := repo.GetNodeByID(ctx, child.ID)
	require.NoError(t, err)
	assert.Equal(t, "更新后的文档", got.Title)

	// 5. 删除节点（软删）
	require.NoError(t, repo.DeleteNode(ctx, child.ID))
	_, err = repo.GetNodeByID(ctx, child.ID)
	assert.Error(t, err)
}

// TestIntegration_DocumentCRUD 验证 Document 的完整 CRUD 流程
func TestIntegration_DocumentCRUD(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := testCtx("tenant-1", "user-1")

	// 创建 Space 和 Node
	space := &model.WikiSpace{
		Name:       "文档测试空间",
		Visibility: model.VisibilityTenant,
		CreatedBy:  "user-1",
	}
	require.NoError(t, repo.CreateSpace(ctx, space))

	node := &model.WikiNode{
		SpaceID: space.ID,
		Type:    model.NodeTypeDocument,
		Title:   "测试文档",
		Sort:    1,
	}
	require.NoError(t, repo.CreateNode(ctx, node))

	// 1. 创建文档
	doc := &model.Document{
		TenantID:    "tenant-1",
		NodeID:      node.ID,
		Content:     "# 内容",
		ContentHTML: "<h1>内容</h1>",
		Format:      "markdown",
		CurrentVer:  1,
	}
	require.NoError(t, repo.CreateDocument(ctx, doc))
	assert.NotEmpty(t, doc.ID)
	assert.Equal(t, 1, doc.CurrentVer)

	// 2. 查询文档
	got, err := repo.GetDocumentByNodeID(ctx, node.ID)
	require.NoError(t, err)
	assert.Equal(t, doc.ID, got.ID)
	assert.Equal(t, "# 内容", got.Content)

	// 3. 更新文档（创建新版本）
	doc.Content = "更新内容"
	doc.ContentHTML = "<h1>更新内容</h1>"
	doc.CurrentVer = 2                                   // 手动设置新版本号
	require.NoError(t, repo.UpdateDocument(ctx, doc, 1)) // expectedVer=1
	got, err = repo.GetDocumentByNodeID(ctx, node.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.CurrentVer)
	assert.Equal(t, "更新内容", got.Content)

	// 4. 删除文档（软删）
	require.NoError(t, repo.DeleteDocument(ctx, doc.ID))
	_, err = repo.GetDocumentByNodeID(ctx, node.ID)
	assert.Error(t, err)
}

// TestIntegration_SpaceMembers 验证 Space 成员管理
func TestIntegration_SpaceMembers(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := testCtx("tenant-1", "user-1")

	// 创建 Space
	space := &model.WikiSpace{
		Name:       "成员测试空间",
		Visibility: model.VisibilityPrivate,
		CreatedBy:  "user-1",
	}
	require.NoError(t, repo.CreateSpace(ctx, space))

	// 1. 添加成员
	member := &model.WikiSpaceMember{
		SpaceID: space.ID,
		UserID:  "user-2",
		Role:    model.SpaceRoleEditor,
	}
	require.NoError(t, repo.AddMember(ctx, member))

	// 2. 查询成员列表
	members, err := repo.ListMembers(ctx, space.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, "user-2", members[0].UserID)
	assert.Equal(t, model.SpaceRoleEditor, members[0].Role)

	// 3. 更新成员角色
	member.Role = model.SpaceRoleAdmin
	require.NoError(t, repo.UpdateMemberRole(ctx, member))
	members, err = repo.ListMembers(ctx, space.ID)
	require.NoError(t, err)
	assert.Equal(t, model.SpaceRoleAdmin, members[0].Role)

	// 4. 移除成员
	require.NoError(t, repo.RemoveMember(ctx, space.ID, "user-2"))
	members, err = repo.ListMembers(ctx, space.ID)
	require.NoError(t, err)
	assert.Empty(t, members)
}

// TestIntegration_TrashBin 验证回收站功能
func TestIntegration_TrashBin(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := testCtx("tenant-1", "user-1")

	// 创建 Space
	space := &model.WikiSpace{
		Name:       "回收站测试空间",
		Visibility: model.VisibilityTenant,
		CreatedBy:  "user-1",
	}
	require.NoError(t, repo.CreateSpace(ctx, space))

	// 1. 创建回收站记录
	trashItem := &model.TrashItem{
		TenantID:  "tenant-1",
		ItemType:  model.TrashTypeSpace,
		ItemID:    space.ID,
		SpaceID:   space.ID,
		Title:     space.Name,
		DeletedBy: "user-1",
	}
	require.NoError(t, repo.CreateTrashItem(ctx, trashItem))
	assert.NotEmpty(t, trashItem.ID)

	// 2. 查询回收站列表
	items, total, err := repo.ListTrashItems(ctx, "tenant-1", space.ID, "", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, items, 1)
	assert.Equal(t, space.ID, items[0].ItemID)
}

// TestIntegration_SoftDeleteKeepsRow 验证空间/节点/文档删除为「软删」而非硬删：
// 删除后正常查询不可见，但 Unscoped 仍能取到且 deleted_at 已填充，且清空后可恢复。
// 该用例锁定此前模型 DeletedAt 为 *time.Time 时 GORM 不识别软删、实际执行硬删的缺陷。
func TestIntegration_SoftDeleteKeepsRow(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := testCtx("tenant-1", "user-1")

	space := &model.WikiSpace{Name: "软删空间", Visibility: model.VisibilityTenant, CreatedBy: "user-1"}
	require.NoError(t, repo.CreateSpace(ctx, space))
	node := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeFolder, Title: "软删节点"}
	require.NoError(t, repo.CreateNode(ctx, node))
	doc := &model.Document{TenantID: "tenant-1", NodeID: node.ID, Content: "x", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))

	// 删除空间（级联删除节点与文档由 service 层负责；此处仅验证仓储层各自软删）
	require.NoError(t, repo.DeleteSpace(ctx, space.ID))
	require.NoError(t, repo.DeleteNode(ctx, node.ID))
	require.NoError(t, repo.DeleteDocument(ctx, doc.ID))

	// 正常查询均不可见
	_, err := repo.GetSpaceByID(ctx, space.ID)
	assert.Error(t, err)
	_, err = repo.GetNodeByID(ctx, node.ID)
	assert.Error(t, err)
	_, err = repo.GetDocumentByNodeID(ctx, node.ID)
	assert.Error(t, err)

	// Unscoped 仍可取到，且 deleted_at 已填充
	var sp model.WikiSpace
	require.NoError(t, repo.db.Unscoped().First(&sp, "id = ?", space.ID).Error)
	assert.False(t, sp.DeletedAt.Time.IsZero())

	var nd model.WikiNode
	require.NoError(t, repo.db.Unscoped().First(&nd, "id = ?", node.ID).Error)
	assert.False(t, nd.DeletedAt.Time.IsZero())

	var dm model.Document
	require.NoError(t, repo.db.Unscoped().First(&dm, "id = ?", doc.ID).Error)
	assert.False(t, dm.DeletedAt.Time.IsZero())

	// 恢复（清空 deleted_at）后正常查询可见
	require.NoError(t, repo.db.Unscoped().Model(&model.WikiNode{}).Where("id = ?", node.ID).Update("deleted_at", nil).Error)
	_, err = repo.GetNodeByID(ctx, node.ID)
	assert.NoError(t, err)
}

// TestIntegration_SpaceDeleteCascadesAndRestores 验证删除空间会级联软删整棵子树（节点+文档），
// 且恢复空间能整体还原子树内容，避免“删空间后内容仍可达”或“恢复空间后子树丢失”。
func TestIntegration_SpaceDeleteCascadesAndRestores(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := testCtx("tenant-1", "user-1")

	space := &model.WikiSpace{Name: "级联空间", Visibility: model.VisibilityTenant, CreatedBy: "user-1"}
	require.NoError(t, repo.CreateSpace(ctx, space))
	folder := &model.WikiNode{SpaceID: space.ID, Type: model.NodeTypeFolder, Title: "目录", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, folder))
	docNode := &model.WikiNode{SpaceID: space.ID, ParentID: folder.ID, Type: model.NodeTypeDocument, Title: "文档", Sort: 1}
	require.NoError(t, repo.CreateNode(ctx, docNode))
	doc := &model.Document{TenantID: "tenant-1", NodeID: docNode.ID, Content: "c", Format: "markdown", CurrentVer: 1}
	require.NoError(t, repo.CreateDocument(ctx, doc))

	// 服务层语义：先批量级联软删子树，再软删空间
	require.NoError(t, repo.SoftDeleteSpaceTree(ctx, space.ID))
	require.NoError(t, repo.DeleteSpace(ctx, space.ID))

	// 正常查询：空间/节点/文档均不可见
	_, err := repo.GetSpaceByID(ctx, space.ID)
	assert.Error(t, err)
	nodes, err := repo.ListNodesBySpace(ctx, space.ID)
	require.NoError(t, err)
	assert.Empty(t, nodes)
	_, err = repo.GetDocumentByNodeID(ctx, docNode.ID)
	assert.Error(t, err)

	// Unscoped：子树与空间行仍存在（仅被软删）
	var allNodes []model.WikiNode
	require.NoError(t, repo.db.Unscoped().Where("space_id = ?", space.ID).Find(&allNodes).Error)
	assert.Len(t, allNodes, 2)
	for _, n := range allNodes {
		assert.False(t, n.DeletedAt.Time.IsZero())
	}
	var sp model.WikiSpace
	require.NoError(t, repo.db.Unscoped().First(&sp, "id = ?", space.ID).Error)
	assert.False(t, sp.DeletedAt.Time.IsZero())

	// 恢复空间（级联恢复整棵子树）
	require.NoError(t, repo.RestoreSpace(ctx, space.ID))

	_, err = repo.GetSpaceByID(ctx, space.ID)
	assert.NoError(t, err)
	nodes, err = repo.ListNodesBySpace(ctx, space.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
	gotDoc, err := repo.GetDocumentByNodeID(ctx, docNode.ID)
	require.NoError(t, err)
	assert.Equal(t, doc.ID, gotDoc.ID)
}
