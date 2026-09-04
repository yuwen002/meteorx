package service

import (
	"context"
	"errors"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"
	apperrors "meteorx/internal/pkg/apperrors"
	dbpkg "meteorx/internal/pkg/db"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newTxManager(t *testing.T) (*dbpkg.TxManager, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	return dbpkg.NewTxManager(gormDB), mock
}

func serviceCtx(tenantID string) context.Context {
	return contextx.SetVars(context.Background(), tenantID, "user-1", []string{"admin"})
}

// TestUpdateDocument_VersionConflictIs409 验证乐观锁冲突被映射为 HTTP 409。
func TestUpdateDocument_VersionConflictIs409(t *testing.T) {
	txManager, mock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	repo.SeedDocument(&model.Document{ID: "doc-1", NodeID: "node-1", CurrentVer: 3})
	repo.UpdateDocumentErr = repository.ErrDocumentVersionConflict

	// 回调返回错误 → 事务回滚
	mock.ExpectBegin()
	mock.ExpectRollback()

	svc := NewWikiService(repo, txManager)

	_, err := svc.UpdateDocument(serviceCtx("tenant-1"), "doc-1", "user-1", &dto.UpdateDocumentReq{
		Content: "新内容",
	})

	assert.Error(t, err)
	var appErr *apperrors.AppError
	assert.True(t, errors.As(err, &appErr), "error should be AppError type")
	if appErr != nil {
		assert.Equal(t, 409, appErr.StatusCode)
	}
}

// TestUpdateDocument_NotFoundPropagated 验证文档不存在错误原样传播（非 409）。
func TestUpdateDocument_NotFoundPropagated(t *testing.T) {
	txManager, mock := newTxManager(t)
	repo := repository.NewMockWikiRepository()

	mock.ExpectBegin()
	mock.ExpectRollback()

	svc := NewWikiService(repo, txManager)

	_, err := svc.UpdateDocument(serviceCtx("tenant-1"), "doc-1", "user-1", &dto.UpdateDocumentReq{
		Content: "新内容",
	})

	assert.ErrorIs(t, err, repository.ErrDocumentNotFound)
}

// TestCreateSpace_AddMemberFailureReturnsError 验证 CreateSpace+AddMember 任一失败即返回错误，
// 且事务回滚（WithTx 未提交成功的数据）。
func TestCreateSpace_AddMemberFailureReturnsError(t *testing.T) {
	txManager, mock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	repo.AddMemberErr = errors.New("member insert failed")

	mock.ExpectBegin()
	mock.ExpectRollback()

	svc := NewWikiService(repo, txManager)

	_, err := svc.CreateSpace(serviceCtx("tenant-1"), "tenant-1", "user-1", &dto.CreateWikiSpaceReq{
		Name:        "测试空间",
		Description: "desc",
	})

	assert.Error(t, err)
	assert.EqualError(t, err, "member insert failed")
}

// TestCreateSpace_Commits 验证 CreateSpace 成功时提交（无错误返回）。
func TestCreateSpace_Commits(t *testing.T) {
	txManager, mock := newTxManager(t)
	repo := repository.NewMockWikiRepository()

	mock.ExpectBegin()
	mock.ExpectCommit()

	svc := NewWikiService(repo, txManager)

	resp, err := svc.CreateSpace(serviceCtx("tenant-1"), "tenant-1", "user-1", &dto.CreateWikiSpaceReq{
		Name: "测试空间",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "测试空间", resp.Name)
}
