package service

import (
	"errors"
	"strings"
	"testing"

	"meteorx/internal/modules/wiki/repository"
	apperrors "meteorx/internal/pkg/apperrors"

	"github.com/stretchr/testify/assert"
)

// TestSearch_EmptyQuery 验证空关键字直接返回空结果（不触发查询）。
func TestSearch_EmptyQuery(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	svc := NewWikiService(repo, txManager)

	results, total, err := svc.Search(serviceCtx("tenant-1"), "tenant-1", "user-1", "", "", 1, 10)

	assert.NoError(t, err)
	assert.Empty(t, results)
	assert.Equal(t, int64(0), total)
}

// TestSearch_NoMatch 验证无匹配内容时返回空且无错误。
func TestSearch_NoMatch(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	svc := NewWikiService(repo, txManager)

	results, total, err := svc.Search(serviceCtx("tenant-1"), "tenant-1", "user-1", "不存在的关键字", "", 1, 10)

	assert.NoError(t, err)
	assert.Empty(t, results)
	assert.Equal(t, int64(0), total)
}

// TestGenerateSnippet_MiddleMatch 验证片段围绕命中位置截取并保留关键字。
func TestGenerateSnippet_MiddleMatch(t *testing.T) {
	txManager, _ := newTxManager(t)
	svc := NewWikiService(repository.NewMockWikiRepository(), txManager)
	content := strings.Repeat("前", 200) + "meteorx关键字" + strings.Repeat("后", 200)

	snippet := svc.(*wikiService).generateSnippet(content, "meteorx关键字")

	assert.True(t, strings.Contains(snippet, "meteorx关键字"))
	assert.True(t, strings.HasPrefix(snippet, "..."))
	assert.True(t, strings.HasSuffix(snippet, "..."))
	assert.LessOrEqual(t, len(snippet), 206)
}

// TestGenerateSnippet_ShortContent 验证短内容原样返回。
func TestGenerateSnippet_ShortContent(t *testing.T) {
	txManager, _ := newTxManager(t)
	svc := NewWikiService(repository.NewMockWikiRepository(), txManager)

	snippet := svc.(*wikiService).generateSnippet("hello world", "world")

	assert.Equal(t, "hello world", snippet)
}

// TestRestoreTrashItem_NotFound 验证不存在的回收站项目返回 404。
func TestRestoreTrashItem_NotFound(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	svc := NewWikiService(repo, txManager)

	err := svc.RestoreTrashItem(serviceCtx("tenant-1"), "missing-item", "user-1")

	var appErr *apperrors.AppError
	assert.True(t, errors.As(err, &appErr), "error should be AppError type")
	if appErr != nil {
		assert.Equal(t, 404, appErr.StatusCode)
	}
}

// TestPermanentDeleteTrashItem_NotFound 验证永久删除不存在的项目返回 404。
func TestPermanentDeleteTrashItem_NotFound(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	svc := NewWikiService(repo, txManager)

	err := svc.PermanentDeleteTrashItem(serviceCtx("tenant-1"), "missing-item", "user-1")

	var appErr *apperrors.AppError
	assert.True(t, errors.As(err, &appErr), "error should be AppError type")
	if appErr != nil {
		assert.Equal(t, 404, appErr.StatusCode)
	}
}

// TestDeleteAttachment_NotFound 验证删除不存在的附件返回 404。
func TestDeleteAttachment_NotFound(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	svc := NewWikiService(repo, txManager)

	err := svc.DeleteAttachment(serviceCtx("tenant-1"), "missing-att", "user-1")

	var appErr *apperrors.AppError
	assert.True(t, errors.As(err, &appErr), "error should be AppError type")
	if appErr != nil {
		assert.Equal(t, 404, appErr.StatusCode)
	}
}
