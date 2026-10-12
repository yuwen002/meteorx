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

// stubSpaceQuota 固定 Wiki 空间数量上限的配额提供者。
type stubSpaceQuota struct {
	limit int
	err   error
}

func (s stubSpaceQuota) GetTenantWikiSpaceLimit(_ context.Context, _ string) (int, error) {
	return s.limit, s.err
}

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

// TestDeleteSpace_CascadesSoftDeleteToNodesAndDocuments 验证删除 Space 时通过批量级联软删整棵子树，
// 并仅生成一条空间回收站记录（不再为子节点/文档各自生成回收站条目）。
func TestDeleteSpace_CascadesSoftDeleteToNodesAndDocuments(t *testing.T) {
	txManager, mock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedSpace(t, repo, "space-1", "待删除空间", model.VisibilityTenant)
	assert.NoError(t, repo.AddMember(context.Background(), &model.WikiSpaceMember{
		SpaceID: "space-1",
		UserID:  "user-1",
		Role:    model.SpaceRoleOwner,
	}))

	// 设置事务 mock 期望
	mock.ExpectBegin()
	mock.ExpectCommit()

	svc := NewWikiService(repo, txManager)
	err := svc.DeleteSpace(serviceCtx("tenant-1"), "space-1", "tenant-1")

	assert.NoError(t, err)
	assert.Equal(t, 1, repo.SoftDeleteSpaceTreeCalls, "应调用一次批量级联软删")
	assert.Equal(t, 0, repo.DeleteNodeCalls, "不应再逐节点删除")
	assert.Equal(t, 0, repo.DeleteDocumentCalls, "不应再逐文档删除")
	assert.Equal(t, 1, repo.CreateTrashItemCalls, "仅生成一条空间回收站记录")
}

// TestDeleteSpace_RollbackOnError 验证级联删除中途失败时整体回滚。
func TestDeleteSpace_RollbackOnError(t *testing.T) {
	txManager, mock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedSpace(t, repo, "space-1", "空间", model.VisibilityTenant)
	assert.NoError(t, repo.AddMember(context.Background(), &model.WikiSpaceMember{
		SpaceID: "space-1",
		UserID:  "user-1",
		Role:    model.SpaceRoleOwner,
	}))

	repo.SoftDeleteSpaceTreeErr = errors.New("数据库错误")

	// 设置事务 mock 期望：开始事务后回滚
	mock.ExpectBegin()
	mock.ExpectRollback()

	svc := NewWikiService(repo, txManager)
	err := svc.DeleteSpace(serviceCtx("tenant-1"), "space-1", "tenant-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "数据库错误")
}

// ---------- 创建空间前配额校验 ----------

// TestCreateSpace_QuotaExceeded 已达上限时拒绝创建，不进入事务。
func TestCreateSpace_QuotaExceeded(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedSpace(t, repo, "s1", "A", model.VisibilityTenant)
	seedSpace(t, repo, "s2", "B", model.VisibilityTenant)
	// 已有 2 个，上限 2 -> 拒绝
	svc := NewWikiService(repo, txManager, WithSpaceQuota(stubSpaceQuota{limit: 2}))

	_, err := svc.CreateSpace(serviceCtx("tenant-1"), "tenant-1", "user-1", &dto.CreateWikiSpaceReq{Name: "C"})

	var appErr *apperrors.AppError
	assert.True(t, errors.As(err, &appErr), "应为 AppError")
	if appErr != nil {
		assert.Equal(t, 402, appErr.StatusCode)
	}
}

// TestCreateSpace_QuotaUnlimited 上限为 -1 时放行创建。
func TestCreateSpace_QuotaUnlimited(t *testing.T) {
	txManager, mock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	mock.ExpectBegin()
	mock.ExpectCommit()
	svc := NewWikiService(repo, txManager, WithSpaceQuota(stubSpaceQuota{limit: -1}))

	_, err := svc.CreateSpace(serviceCtx("tenant-1"), "tenant-1", "user-1", &dto.CreateWikiSpaceReq{Name: "C"})
	assert.NoError(t, err)
}

// TestCreateSpace_QuotaFailOpenOnError 配额查询失败时 fail-open，不因配额拒绝。
func TestCreateSpace_QuotaFailOpenOnError(t *testing.T) {
	txManager, mock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	mock.ExpectBegin()
	mock.ExpectCommit()
	svc := NewWikiService(repo, txManager, WithSpaceQuota(stubSpaceQuota{err: errors.New("db down")}))

	_, err := svc.CreateSpace(serviceCtx("tenant-1"), "tenant-1", "user-1", &dto.CreateWikiSpaceReq{Name: "C"})
	// 放行后正常创建（若因 stub 其他原因失败也不应是配额 402）
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		assert.NotEqual(t, 402, appErr.StatusCode)
	}
}
