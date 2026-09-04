package service

import (
	"context"
	"errors"
	"testing"

	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"
	apperrors "meteorx/internal/pkg/apperrors"

	"github.com/stretchr/testify/assert"
)

func seedSpace(t *testing.T, repo *repository.MockWikiRepository, id string, name string, visibility int) {
	t.Helper()
	space := &model.WikiSpace{
		ID:         id,
		Name:       name,
		TenantID:   "tenant-1",
		Visibility: visibility,
		CreatedBy:  "user-1",
	}
	assert.NoError(t, repo.CreateSpace(context.Background(), space))
}

// TestGetSpace_ReturnsDetailWithRole 验证成员获取 Space 详情时返回成员数与自身角色。
func TestGetSpace_ReturnsDetailWithRole(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedSpace(t, repo, "space-1", "产品空间", model.VisibilityTenant)
	assert.NoError(t, repo.AddMember(context.Background(), &model.WikiSpaceMember{
		SpaceID: "space-1",
		UserID:  "user-1",
		Role:    model.SpaceRoleOwner,
	}))

	svc := NewWikiService(repo, txManager)
	resp, err := svc.GetSpace(serviceCtx("tenant-1"), "space-1", "user-1")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "space-1", resp.ID)
	assert.Equal(t, "产品空间", resp.Name)
	assert.Equal(t, int64(1), resp.MemberCount)
	assert.Equal(t, model.SpaceRoleOwner, resp.MyRole)
}

// TestGetSpace_NotFound 验证不存在的 Space 错误原样传播。
func TestGetSpace_NotFound(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	svc := NewWikiService(repo, txManager)

	_, err := svc.GetSpace(serviceCtx("tenant-1"), "missing", "user-1")

	assert.ErrorIs(t, err, repository.ErrWikiSpaceNotFound)
}

// TestGetSpace_PrivateSpaceNonMemberForbidden 验证非成员访问私有 Space 返回 403。
func TestGetSpace_PrivateSpaceNonMemberForbidden(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedSpace(t, repo, "space-1", "私有空间", model.VisibilityPrivate)
	repo.GetMemberFn = func(ctx context.Context, spaceID, userID string) (*model.WikiSpaceMember, error) {
		return nil, nil
	}
	svc := NewWikiService(repo, txManager)

	resp, err := svc.GetSpace(serviceCtx("tenant-1"), "space-1", "other-1")

	assert.Nil(t, resp)
	var appErr *apperrors.AppError
	assert.True(t, errors.As(err, &appErr), "error should be AppError type")
	if appErr != nil {
		assert.Equal(t, 403, appErr.StatusCode)
	}
}

// TestUpdateSpace_UpdatesFields 验证更新 Space 名称/描述成功。
func TestUpdateSpace_UpdatesFields(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedSpace(t, repo, "space-1", "旧名称", model.VisibilityTenant)
	svc := NewWikiService(repo, txManager)

	resp, err := svc.UpdateSpace(serviceCtx("tenant-1"), "space-1", "tenant-1", &dto.UpdateWikiSpaceReq{
		Name:        "新名称",
		Description: "新描述",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "新名称", resp.Name)
	assert.Equal(t, "新描述", resp.Description)
}

// TestListMembers_ReturnsMembers 验证成员列表返回。
func TestListMembers_ReturnsMembers(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedSpace(t, repo, "space-1", "空间", model.VisibilityTenant)
	for _, u := range []string{"user-1", "user-2"} {
		assert.NoError(t, repo.AddMember(context.Background(), &model.WikiSpaceMember{
			SpaceID: "space-1",
			UserID:  u,
			Role:    model.SpaceRoleEditor,
		}))
	}
	svc := NewWikiService(repo, txManager)

	members, err := svc.ListMembers(serviceCtx("tenant-1"), "space-1")

	assert.NoError(t, err)
	assert.Len(t, members, 2)
}
