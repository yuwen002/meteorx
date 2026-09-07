package repository

import (
	"context"
	"errors"

	"meteorx/internal/modules/wiki/model"
)

// MockWikiRepository 是 WikiRepository 的内存实现，用于 service 层单元测试。
// 可在字段上设置注入的错误，以模拟特定步骤的失败。
type MockWikiRepository struct {
	// CreateSpaceErr 若非 nil，CreateSpace 返回该错误
	CreateSpaceErr error
	// AddMemberErr 若非 nil，AddMember 返回该错误
	AddMemberErr error
	// UpdateDocumentErr 若非 nil，UpdateDocument 返回该错误
	UpdateDocumentErr error
	// GetMemberFn 可选注入，覆盖默认 GetMember 行为（用于模拟非成员/非 Owner 等权限场景）
	GetMemberFn func(ctx context.Context, spaceID, userID string) (*model.WikiSpaceMember, error)
	// GetNodeFn 可选注入，覆盖默认 GetNodeByID 行为（返回带 SpaceID/Type/Title 的节点）
	GetNodeFn func(ctx context.Context, id string) (*model.WikiNode, error)

	// 删除行为注入
	DeleteDocumentErr               error // 非 nil 时 DeleteDocument 返回该错误
	DeleteDocumentCalls             int   // 记录 DeleteDocument 被调用的次数
	DeleteNodeCalls                 int   // 记录 DeleteNode 被调用的次数
	DeleteAttachmentsByDocumentCalls int  // 记录 DeleteAttachmentsByDocument 被调用的次数

	// 回收站行为注入
	GetTrashItemErr      error   // 非 nil 时 GetTrashItem 优先返回该错误
	RestoreErr           error   // 非 nil 时 RestoreSpace/RestoreDocument/RestoreNodeTree 返回该错误
	PurgeErr             error   // 非 nil 时 PurgeSpaceTree/PurgeDocument/PurgeNodeTree 返回该错误
	DeleteTrashItemErr   error   // 非 nil 时 DeleteTrashItem 返回该错误
	DeleteTrashItemCalls int     // 记录 DeleteTrashItem 被调用的次数
	LastCreatedTrash     *model.TrashItem // 最近一次 CreateTrashItem 写入的项目

	spaces  map[string]*model.WikiSpace
	members map[string][]*model.WikiSpaceMember
	docs    map[string]*model.Document
	trash   map[string]*model.TrashItem
}

func NewMockWikiRepository() *MockWikiRepository {
	return &MockWikiRepository{
		spaces:  map[string]*model.WikiSpace{},
		members: map[string][]*model.WikiSpaceMember{},
		docs:    map[string]*model.Document{},
		trash:   map[string]*model.TrashItem{},
	}
}

// SeedDocument 向 mock 中预置一个文档，用于版本冲突等场景的测试。
func (m *MockWikiRepository) SeedDocument(doc *model.Document) {
	m.docs[doc.ID] = doc
}

// SeedTrashItem 向 mock 中预置一个回收站项目。
func (m *MockWikiRepository) SeedTrashItem(item *model.TrashItem) {
	m.trash[item.ID] = item
}

func (m *MockWikiRepository) CreateSpace(ctx context.Context, space *model.WikiSpace) error {
	if m.CreateSpaceErr != nil {
		return m.CreateSpaceErr
	}
	m.spaces[space.ID] = space
	return nil
}

func (m *MockWikiRepository) GetSpaceByID(ctx context.Context, id string) (*model.WikiSpace, error) {
	if s, ok := m.spaces[id]; ok {
		return s, nil
	}
	return nil, ErrWikiSpaceNotFound
}

func (m *MockWikiRepository) ListSpaces(ctx context.Context, tenantID string, userID string, keyword string, page, pageSize int) ([]*model.WikiSpace, int64, error) {
	return nil, 0, nil
}

func (m *MockWikiRepository) UpdateSpace(ctx context.Context, space *model.WikiSpace) error {
	m.spaces[space.ID] = space
	return nil
}

func (m *MockWikiRepository) DeleteSpace(ctx context.Context, id string) error {
	delete(m.spaces, id)
	return nil
}

func (m *MockWikiRepository) CreateNode(ctx context.Context, node *model.WikiNode) error {
	return nil
}

func (m *MockWikiRepository) GetNodeByID(ctx context.Context, id string) (*model.WikiNode, error) {
	if m.GetNodeFn != nil {
		return m.GetNodeFn(ctx, id)
	}
	return &model.WikiNode{ID: id, Type: model.NodeTypeDocument}, nil
}

func (m *MockWikiRepository) ListNodesBySpace(ctx context.Context, spaceID string) ([]*model.WikiNode, error) {
	return nil, nil
}

func (m *MockWikiRepository) ListChildNodes(ctx context.Context, parentID string) ([]*model.WikiNode, error) {
	return nil, nil
}

func (m *MockWikiRepository) UpdateNode(ctx context.Context, node *model.WikiNode) error {
	return nil
}

func (m *MockWikiRepository) DeleteNode(ctx context.Context, id string) error {
	m.DeleteNodeCalls++
	return nil
}

func (m *MockWikiRepository) CountChildNodes(ctx context.Context, parentID string) (int64, error) {
	return 0, nil
}

func (m *MockWikiRepository) CreateDocument(ctx context.Context, doc *model.Document) error {
	return nil
}

func (m *MockWikiRepository) GetDocumentByNodeID(ctx context.Context, nodeID string) (*model.Document, error) {
	for _, d := range m.docs {
		if d.NodeID == nodeID {
			return d, nil
		}
	}
	return nil, ErrDocumentNotFound
}

func (m *MockWikiRepository) GetDocumentByID(ctx context.Context, id string) (*model.Document, error) {
	if d, ok := m.docs[id]; ok {
		return d, nil
	}
	return nil, ErrDocumentNotFound
}

func (m *MockWikiRepository) UpdateDocument(ctx context.Context, doc *model.Document, expectedVer int) error {
	if m.UpdateDocumentErr != nil {
		return m.UpdateDocumentErr
	}
	return nil
}

func (m *MockWikiRepository) DeleteDocument(ctx context.Context, id string) error {
	m.DeleteDocumentCalls++
	return m.DeleteDocumentErr
}

func (m *MockWikiRepository) IncrementViewCount(ctx context.Context, id string) error {
	return nil
}

func (m *MockWikiRepository) CreateRevision(ctx context.Context, revision *model.DocumentRevision) error {
	return nil
}

func (m *MockWikiRepository) ListRevisions(ctx context.Context, documentID string) ([]*model.DocumentRevision, error) {
	return nil, nil
}

func (m *MockWikiRepository) GetRevisionByVersion(ctx context.Context, documentID string, version int) (*model.DocumentRevision, error) {
	return nil, ErrRevisionNotFound
}

func (m *MockWikiRepository) AddMember(ctx context.Context, member *model.WikiSpaceMember) error {
	if m.AddMemberErr != nil {
		return m.AddMemberErr
	}
	m.members[member.SpaceID] = append(m.members[member.SpaceID], member)
	return nil
}

func (m *MockWikiRepository) RemoveMember(ctx context.Context, spaceID, userID string) error {
	return nil
}

func (m *MockWikiRepository) UpdateMemberRole(ctx context.Context, member *model.WikiSpaceMember) error {
	return nil
}

func (m *MockWikiRepository) ListMembers(ctx context.Context, spaceID string) ([]*model.WikiSpaceMember, error) {
	return m.members[spaceID], nil
}

func (m *MockWikiRepository) GetMember(ctx context.Context, spaceID, userID string) (*model.WikiSpaceMember, error) {
	if m.GetMemberFn != nil {
		return m.GetMemberFn(ctx, spaceID, userID)
	}
	return &model.WikiSpaceMember{
		SpaceID: spaceID,
		UserID:  userID,
		Role:    model.SpaceRoleOwner,
	}, nil
}

func (m *MockWikiRepository) GetWikiStats(ctx context.Context, tenantID string) (*model.WikiStats, error) {
	return &model.WikiStats{}, nil
}

func (m *MockWikiRepository) CreateNodePermission(ctx context.Context, perm *model.WikiNodePermission) error {
	return nil
}

func (m *MockWikiRepository) GetNodePermissions(ctx context.Context, nodeID string) ([]*model.WikiNodePermission, error) {
	return nil, nil
}

func (m *MockWikiRepository) GetNodePermission(ctx context.Context, nodeID, userID, permission string) (*model.WikiNodePermission, error) {
	return nil, nil
}

func (m *MockWikiRepository) UpdateNodePermission(ctx context.Context, perm *model.WikiNodePermission) error {
	return nil
}

func (m *MockWikiRepository) DeleteNodePermission(ctx context.Context, id string) error {
	return nil
}

func (m *MockWikiRepository) DeleteNodePermissionsByNode(ctx context.Context, nodeID string) error {
	return nil
}

func (m *MockWikiRepository) GetUserNodePermissions(ctx context.Context, nodeID, userID string) ([]*model.WikiNodePermission, error) {
	return nil, nil
}

func (m *MockWikiRepository) CreateTrashItem(ctx context.Context, item *model.TrashItem) error {
	m.trash[item.ID] = item
	m.LastCreatedTrash = item
	return nil
}

func (m *MockWikiRepository) ListTrashItems(ctx context.Context, tenantID string, spaceID string, itemType string, page, pageSize int) ([]*model.TrashItem, int64, error) {
	var out []*model.TrashItem
	for _, it := range m.trash {
		if spaceID != "" && it.SpaceID != spaceID {
			continue
		}
		if itemType != "" && it.ItemType != itemType {
			continue
		}
		out = append(out, it)
	}
	return out, int64(len(out)), nil
}

func (m *MockWikiRepository) GetTrashItem(ctx context.Context, id string) (*model.TrashItem, error) {
	if it, ok := m.trash[id]; ok {
		return it, nil
	}
	if m.GetTrashItemErr != nil {
		return nil, m.GetTrashItemErr
	}
	return nil, errors.New("trash item not found")
}

func (m *MockWikiRepository) DeleteTrashItem(ctx context.Context, id string) error {
	m.DeleteTrashItemCalls++
	delete(m.trash, id)
	return m.DeleteTrashItemErr
}

func (m *MockWikiRepository) ExpireTrashItems(ctx context.Context) error {
	return nil
}

func (m *MockWikiRepository) RestoreDocument(ctx context.Context, id string) error {
	return m.RestoreErr
}

func (m *MockWikiRepository) RestoreSpace(ctx context.Context, id string) error {
	return m.RestoreErr
}

func (m *MockWikiRepository) RestoreNodeTree(ctx context.Context, id string) error {
	return m.RestoreErr
}

func (m *MockWikiRepository) PurgeDocument(ctx context.Context, id string) error {
	return m.PurgeErr
}

func (m *MockWikiRepository) PurgeNodeTree(ctx context.Context, id string) error {
	return m.PurgeErr
}

func (m *MockWikiRepository) PurgeSpaceTree(ctx context.Context, id string) error {
	return m.PurgeErr
}

func (m *MockWikiRepository) SearchNodesByTitle(ctx context.Context, tenantID string, spaceID string, query string) ([]*model.WikiNode, error) {
	return nil, nil
}

func (m *MockWikiRepository) SearchDocumentsByContent(ctx context.Context, tenantID string, spaceID string, query string) ([]*model.Document, error) {
	return nil, nil
}

func (m *MockWikiRepository) CreateAttachment(ctx context.Context, attachment *model.Attachment) error {
	return nil
}

func (m *MockWikiRepository) ListAttachmentsByDocument(ctx context.Context, documentID string) ([]*model.Attachment, error) {
	return nil, nil
}

func (m *MockWikiRepository) GetAttachment(ctx context.Context, id string) (*model.Attachment, error) {
	return nil, errors.New("attachment not found")
}

func (m *MockWikiRepository) DeleteAttachment(ctx context.Context, id string) error {
	return nil
}

func (m *MockWikiRepository) DeleteAttachmentsByDocument(ctx context.Context, documentID string) error {
	m.DeleteAttachmentsByDocumentCalls++
	return nil
}
