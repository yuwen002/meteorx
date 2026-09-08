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
			sqlDB.Close()
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

	// 3. 查询节点列表（ListNodesBySpace 只返回根节点）
	nodes, err := repo.ListNodesBySpace(ctx, space.ID)
	require.NoError(t, err)
	assert.Len(t, nodes, 1)
	assert.Equal(t, node.ID, nodes[0].ID)

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
	doc.CurrentVer = 2 // 手动设置新版本号
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