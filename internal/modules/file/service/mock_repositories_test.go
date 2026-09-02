package service

import (
	"context"
	"errors"
	"io"
	"strings"

	"meteorx/internal/modules/file/model"
	"meteorx/internal/modules/file/repository"
	"meteorx/internal/modules/file/storage"
)

// ---------- mock: FileRepository ----------

type mockFileRepo struct {
	files    map[string]*model.File
	deleted  map[string]*model.File
	byMD5    map[string]*model.File
	deleteErr error
}

func newMockFileRepo() *mockFileRepo {
	return &mockFileRepo{
		files:   make(map[string]*model.File),
		deleted: make(map[string]*model.File),
		byMD5:   make(map[string]*model.File),
	}
}

func (m *mockFileRepo) seed(f *model.File) {
	m.files[f.ID] = f
	if f.MD5 != "" {
		m.byMD5[f.TenantID+"|"+f.MD5] = f
	}
}

func (m *mockFileRepo) Create(_ context.Context, f *model.File) error {
	m.files[f.ID] = f
	if f.MD5 != "" {
		m.byMD5[f.TenantID+"|"+f.MD5] = f
	}
	return nil
}

func (m *mockFileRepo) GetByID(_ context.Context, id string) (*model.File, error) {
	if f, ok := m.files[id]; ok {
		return f, nil
	}
	return nil, errors.New("file not found")
}

func (m *mockFileRepo) GetByIDUnscoped(_ context.Context, id string) (*model.File, error) {
	if f, ok := m.files[id]; ok {
		return f, nil
	}
	if f, ok := m.deleted[id]; ok {
		return f, nil
	}
	return nil, errors.New("file not found")
}

func (m *mockFileRepo) GetByMD5(_ context.Context, tenantID, md5 string) (*model.File, error) {
	if f, ok := m.byMD5[tenantID+"|"+md5]; ok {
		return f, nil
	}
	return nil, errors.New("file not found")
}

func (m *mockFileRepo) ListByTenant(_ context.Context, _ string, _ int, _ int) ([]*model.File, int64, error) {
	return nil, 0, nil
}

func (m *mockFileRepo) ListByUser(_ context.Context, _ string, _ string, _ int, _ int) ([]*model.File, int64, error) {
	return nil, 0, nil
}

func (m *mockFileRepo) Update(_ context.Context, f *model.File) error {
	if _, ok := m.files[f.ID]; ok {
		m.files[f.ID] = f
	}
	return nil
}

func (m *mockFileRepo) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if f, ok := m.files[id]; ok {
		delete(m.files, id)
		m.deleted[id] = f
	}
	return nil
}

func (m *mockFileRepo) PermanentDelete(_ context.Context, id string) error {
	delete(m.files, id)
	delete(m.deleted, id)
	return nil
}

func (m *mockFileRepo) GetDeletedList(_ context.Context, _ string, _ int, _ int) ([]*model.File, int64, error) {
	var out []*model.File
	for _, f := range m.deleted {
		out = append(out, f)
	}
	return out, int64(len(out)), nil
}

func (m *mockFileRepo) Restore(_ context.Context, id string) error {
	if f, ok := m.deleted[id]; ok {
		delete(m.deleted, id)
		m.files[id] = f
	}
	return nil
}

var _ repository.FileRepository = (*mockFileRepo)(nil)

// ---------- mock: Storage ----------

type mockStorage struct {
	urlPrefix string
	uploaded  map[string]bool
	deleteErr error
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		urlPrefix: "http://local/uploads",
		uploaded:  make(map[string]bool),
	}
}

func (m *mockStorage) Upload(_ context.Context, _ io.Reader, originalName string) (string, error) {
	path := "files/" + originalName
	m.uploaded[path] = true
	return path, nil
}

func (m *mockStorage) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("data")), nil
}

func (m *mockStorage) Delete(_ context.Context, path string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.uploaded, path)
	return nil
}

func (m *mockStorage) Exists(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func (m *mockStorage) GetURL(path string) string {
	return m.urlPrefix + "/" + path
}

func (m *mockStorage) PresignURL(_ context.Context, path string, _ int64) (string, error) {
	return m.urlPrefix + "/" + path, nil
}

func (m *mockStorage) Type() string {
	return "local"
}

var _ storage.Storage = (*mockStorage)(nil)
