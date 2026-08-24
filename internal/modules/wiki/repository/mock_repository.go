package repository

import (
	"context"

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
	return nil, nil
}

func (m *MockWikiRepository) GetWikiStats(ctx context.Context, tenantID string) (*model.WikiStats, error) {
	return &model.WikiStats{}, nil
}
