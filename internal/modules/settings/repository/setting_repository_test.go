package repository

import (
	"context"
	"testing"

	"meteorx/internal/modules/settings/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// newSettingsTestDB 创建内存 SQLite 并迁移系统配置表
func newSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	require.NoError(t, AutoMigrate(db))
	return db
}

// TestSettingRepo_CreateGetRoundTrip 创建后按键读回，字段完整映射（含 config_key 列名映射）
func TestSettingRepo_CreateGetRoundTrip(t *testing.T) {
	db := newSettingsTestDB(t)
	repo := NewSettingRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &model.Setting{
		Key: "registration_enabled", Value: "true", ValueType: model.ValueTypeBool, IsPublic: true,
	}))

	got, err := repo.GetByKey(ctx, "registration_enabled")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "true", got.Value)
	require.Equal(t, model.ValueTypeBool, got.ValueType)
	require.True(t, got.IsPublic)
	require.False(t, got.CreatedAt.IsZero())

	all, err := repo.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)
}

// TestSettingRepo_GetByKey_NotFound 键不存在时返回 (nil, nil)，便于上层判"不存在"
func TestSettingRepo_GetByKey_NotFound(t *testing.T) {
	db := newSettingsTestDB(t)
	repo := NewSettingRepository(db)

	got, err := repo.GetByKey(context.Background(), "no_such_key")
	require.NoError(t, err)
	require.Nil(t, got)
}

// TestSettingRepo_UpdateValue_OnlyValueAndTime 更新仅触及值与更新时间，元数据列不被误改
func TestSettingRepo_UpdateValue_OnlyValueAndTime(t *testing.T) {
	db := newSettingsTestDB(t)
	repo := NewSettingRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &model.Setting{
		Key: "maintenance_mode", Value: "false", ValueType: model.ValueTypeBool, IsPublic: true,
	}))
	createdAt := mustGet(t, repo, "maintenance_mode").CreatedAt

	require.NoError(t, repo.UpdateValue(ctx, "maintenance_mode", "true"))

	got := mustGet(t, repo, "maintenance_mode")
	require.Equal(t, "true", got.Value)
	require.Equal(t, model.ValueTypeBool, got.ValueType, "值类型元数据不得被更新覆盖")
	require.True(t, got.IsPublic, "公开性元数据不得被更新覆盖")
	require.Equal(t, createdAt.UTC().Format("2006-01-02"), got.CreatedAt.UTC().Format("2006-01-02"))
	require.False(t, got.UpdatedAt.Before(got.CreatedAt))
}

// TestSettingRepo_Upsert_BothDirections Upsert：缺行创建、有行仅改值
func TestSettingRepo_Upsert_BothDirections(t *testing.T) {
	db := newSettingsTestDB(t)
	repo := NewSettingRepository(db)
	ctx := context.Background()

	// 缺行 → 创建
	require.NoError(t, repo.Upsert(ctx, &model.Setting{
		Key: "site_name", Value: "MeteorX", ValueType: model.ValueTypeString, IsPublic: false,
	}))
	got := mustGet(t, repo, "site_name")
	require.Equal(t, "MeteorX", got.Value)

	// 有行 → 仅更新值
	require.NoError(t, repo.Upsert(ctx, &model.Setting{
		Key: "site_name", Value: "MeteorX Pro", ValueType: model.ValueTypeString, IsPublic: false,
	}))
	got = mustGet(t, repo, "site_name")
	require.Equal(t, "MeteorX Pro", got.Value)

	all, err := repo.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1, "Upsert 不得产生重复行")
}

// mustGet 读取指定键，不存在则直接失败
func mustGet(t *testing.T, repo SettingRepository, key string) *model.Setting {
	t.Helper()
	got, err := repo.GetByKey(context.Background(), key)
	require.NoError(t, err)
	require.NotNil(t, got, "键 %s 应已存在", key)
	return got
}
