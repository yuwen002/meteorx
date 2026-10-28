package service

import (
	"testing"

	"meteorx/internal/modules/wiki/model"

	"github.com/stretchr/testify/assert"
)

// TestAggregateReactions 覆盖纯聚合函数：按 emoji 归组计数、保留首次出现顺序、mine 命中当前用户，空输入返回 nil。
func TestAggregateReactions(t *testing.T) {
	assert.Nil(t, aggregateReactions(nil, "u1"))
	assert.Nil(t, aggregateReactions([]*model.CommentReaction{}, "u1"))

	rows := []*model.CommentReaction{
		{Emoji: "👍", UserID: "u1"},
		{Emoji: "🎉", UserID: "u2"},
		{Emoji: "👍", UserID: "u3"},
		{Emoji: "👍", UserID: "u1"},
	}
	got := aggregateReactions(rows, "u1")

	// 顺序：先出现的 👍，再 🎉
	assert.Len(t, got, 2)
	assert.Equal(t, "👍", got[0].Emoji)
	assert.Equal(t, int64(3), got[0].Count)
	assert.True(t, got[0].Mine) // u1 点过 👍
	assert.Equal(t, "🎉", got[1].Emoji)
	assert.Equal(t, int64(1), got[1].Count)
	assert.False(t, got[1].Mine) // u1 未点 🎉
}
