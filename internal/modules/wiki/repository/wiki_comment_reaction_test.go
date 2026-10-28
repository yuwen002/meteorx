package repository

import (
	"testing"

	"meteorx/internal/modules/wiki/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_CommentReactions 覆盖评论表情回应的幂等切换、批量拉取、级联删除，
// 以及 SetCommentStatus 后列表对「已解决」与「已删除」的可见性区分。
func TestIntegration_CommentReactions(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	ctx := testCtx("tenant-1", "user-1")
	ext := &wikiRepositoryExtended{wikiRepository: *repo}

	c := &model.Comment{DocumentID: "doc-1", NodeID: "node-1", Content: "hi", CreatedBy: "user-1", Status: model.CommentStatusActive}
	require.NoError(t, ext.CreateComment(ctx, c))

	// 首次表态 → 新增
	added, err := ext.ToggleCommentReaction(ctx, &model.CommentReaction{CommentID: c.ID, UserID: "user-1", Emoji: "👍"})
	require.NoError(t, err)
	assert.True(t, added)
	rows, err := ext.ListReactionsByCommentIDs(ctx, []string{c.ID})
	require.NoError(t, err)
	assert.Len(t, rows, 1)

	// 再次同表情 → 取消（幂等切换）
	added, err = ext.ToggleCommentReaction(ctx, &model.CommentReaction{CommentID: c.ID, UserID: "user-1", Emoji: "👍"})
	require.NoError(t, err)
	assert.False(t, added)
	rows, err = ext.ListReactionsByCommentIDs(ctx, []string{c.ID})
	require.NoError(t, err)
	assert.Empty(t, rows)

	// 两个用户同表情 → 各自一行，由上层聚合计数
	_, err = ext.ToggleCommentReaction(ctx, &model.CommentReaction{CommentID: c.ID, UserID: "user-1", Emoji: "🎉"})
	require.NoError(t, err)
	_, err = ext.ToggleCommentReaction(ctx, &model.CommentReaction{CommentID: c.ID, UserID: "user-2", Emoji: "🎉"})
	require.NoError(t, err)
	rows, err = ext.ListReactionsByCommentIDs(ctx, []string{c.ID})
	require.NoError(t, err)
	assert.Len(t, rows, 2)

	// 空 ID 集合直接短路，不触库
	rows, err = ext.ListReactionsByCommentIDs(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, rows)

	// 级联物理删除
	require.NoError(t, ext.DeleteReactionsByComment(ctx, c.ID))
	rows, err = ext.ListReactionsByCommentIDs(ctx, []string{c.ID})
	require.NoError(t, err)
	assert.Empty(t, rows)

	// 状态切换：已解决仍可见，已删除被过滤
	require.NoError(t, ext.SetCommentStatus(ctx, c.ID, model.CommentStatusResolved))
	list, err := ext.ListCommentsByDocument(ctx, "doc-1")
	require.NoError(t, err)
	assert.Len(t, list, 1)

	require.NoError(t, ext.SetCommentStatus(ctx, c.ID, model.CommentStatusDeleted))
	list, err = ext.ListCommentsByDocument(ctx, "doc-1")
	require.NoError(t, err)
	assert.Empty(t, list)
}
