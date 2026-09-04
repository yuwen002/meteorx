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

	spaces  map[string]*model.WikiSpace
	members map[string][]*model.WikiSpaceMember
	docs    map[string]*model.Document
}

func NewMockWikiRepository() *MockWikiRepository {
	return &MockWikiRepository{
		spaces:  map[string]*model.WikiSpace{},
		members: map[string][]*model.WikiSpaceMember{},
		docs:    map[string]*model.Document{},
	}
}

// SeedDocument 向 mock 中预置一个文档，用于版本冲突等场景的测试。
func (m *MockWikiRepository) SeedDocument(doc *model.Document) {
	m.docs[doc.ID] = doc
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

func (m *MockWikiRepository) ListSpaces(ctx context.Context, tenantID string, userID string, page, pageSize int) ([]*model.WikiSpace, int64, error) {
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
	return nil
}

func (m *MockWikiRepository) CountChildNodes(ctx context.Context, parentID string) (int64, error) {
	return 0, nil
}

func (m *MockWikiRepository) CreateDocument(ctx context.Context, doc *model.Document) error {
	return nil
}

func (m *MockWikiRepository) GetDocumentByNodeID(ctx context.Context, nodeID string) (*model.Document, error) {
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
	return nil
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
	return nil
}

func (m *MockWikiRepository) ListTrashItems(ctx context.Context, tenantID string, spaceID string, itemType string, page, pageSize int) ([]*model.TrashItem, int64, error) {
	return nil, 0, nil
}

func (m *MockWikiRepository) GetTrashItem(ctx context.Context, id string) (*model.TrashItem, error) {
	return nil, errors.New("trash item not found")
}

func (m *MockWikiRepository) DeleteTrashItem(ctx context.Context, id string) error {
	return nil
}

func (m *MockWikiRepository) ExpireTrashItems(ctx context.Context) error {
	return nil
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
	return nil
}
