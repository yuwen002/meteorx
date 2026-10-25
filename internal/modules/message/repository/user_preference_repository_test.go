package repository

import (
	"context"
	"testing"

	metermodel "meteorx/internal/modules/message/model"

	"github.com/stretchr/testify/require"
)

func TestPreferenceRepo_Upsert_OverwritesAndIsIdempotent(t *testing.T) {
	db := newMessageTestDB(t)
	repo := NewPreferenceRepository(db)
	ctx := context.Background()

	// 重复提交同一份偏好不应堆行：唯一索引 (user_id, category) 命中后只更新 muted
	require.NoError(t, repo.Upsert(ctx, "u1", []metermodel.CategoryPreference{{Category: metermodel.CategoryTask, Muted: true}}))
	require.NoError(t, repo.Upsert(ctx, "u1", []metermodel.CategoryPreference{{Category: metermodel.CategoryTask, Muted: true}}))

	var rows []UserMessagePreferencePO
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1, "同一分类只保留一行")
	require.True(t, rows[0].Muted)

	muted, err := repo.MutedCategories(ctx, "u1")
	require.NoError(t, err)
	require.True(t, muted[metermodel.CategoryTask])

	// 改回「接收」是覆盖写而非删行：行数不变，但不再算作关闭
	require.NoError(t, repo.Upsert(ctx, "u1", []metermodel.CategoryPreference{{Category: metermodel.CategoryTask, Muted: false}}))
	muted, err = repo.MutedCategories(ctx, "u1")
	require.NoError(t, err)
	require.False(t, muted[metermodel.CategoryTask], "muted=false 的行不应被当成关闭")

	rows = nil
	require.NoError(t, db.Find(&rows).Error)
	require.Len(t, rows, 1, "覆盖写不产生新行")

	// 空入参是合法调用（无变更），不应报错
	require.NoError(t, repo.Upsert(ctx, "u1", nil))
}

// TestPreferenceRepo_IsolatedByUser 偏好按用户隔离：别人的开关不影响自己。
func TestPreferenceRepo_IsolatedByUser(t *testing.T) {
	db := newMessageTestDB(t)
	repo := NewPreferenceRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Upsert(ctx, "u1", []metermodel.CategoryPreference{
		{Category: metermodel.CategoryTask, Muted: true},
	}))
	require.NoError(t, repo.Upsert(ctx, "u2", []metermodel.CategoryPreference{
		{Category: metermodel.CategoryAnnouncement, Muted: true},
	}))

	muted, err := repo.MutedCategories(ctx, "u1")
	require.NoError(t, err)
	require.Equal(t, map[string]bool{metermodel.CategoryTask: true}, muted)

	muted, err = repo.MutedCategories(ctx, "u3")
	require.NoError(t, err)
	require.Empty(t, muted, "未设置偏好的用户应得到空集合（默认全部接收）")
}

func TestPreferenceRepo_MutedUserIDs(t *testing.T) {
	db := newMessageTestDB(t)
	repo := NewPreferenceRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Upsert(ctx, "u2", []metermodel.CategoryPreference{
		{Category: metermodel.CategoryTask, Muted: true},
	}))
	require.NoError(t, repo.Upsert(ctx, "u3", []metermodel.CategoryPreference{
		{Category: metermodel.CategorySystem, Muted: true},
	}))

	muted, err := repo.MutedUserIDs(ctx, metermodel.CategoryTask, []string{"u1", "u2", "u3", "u4"})
	require.NoError(t, err)
	require.Len(t, muted, 1, "只返回该分类下关闭接收的收件人")
	_, ok := muted["u2"]
	require.True(t, ok)

	// 分类不匹配时为空
	muted, err = repo.MutedUserIDs(ctx, metermodel.CategoryAnnouncement, []string{"u2", "u3"})
	require.NoError(t, err)
	require.Empty(t, muted)

	// 空收件人 / 空分类直接返回空集合，不打库
	muted, err = repo.MutedUserIDs(ctx, metermodel.CategoryTask, nil)
	require.NoError(t, err)
	require.Empty(t, muted)

	muted, err = repo.MutedUserIDs(ctx, "", []string{"u2"})
	require.NoError(t, err)
	require.Empty(t, muted)
}
