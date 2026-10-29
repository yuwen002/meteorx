package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"meteorx/internal/cache"
	"meteorx/internal/modules/settings/dto"
	"meteorx/internal/modules/settings/model"
	"meteorx/internal/modules/settings/repository"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// newTestDB 创建内存 SQLite 并迁移系统配置表
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	require.NoError(t, repository.AutoMigrate(db))
	return db
}

// newTestRedis 基于 miniredis 构造可用的 cache.Redis
func newTestRedis(t *testing.T) (*cache.Redis, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb, err := cache.NewRedis(mr.Addr(), "", 0)
	require.NoError(t, err)
	return rdb, mr
}

func newSvc(db *gorm.DB, rdb *cache.Redis) *SettingService {
	return NewSettingService(repository.NewSettingRepository(db), rdb)
}

// TestSettingSvc_GetFallbackToRegistryDefault DB 无任何行时，读取回退注册表内置默认值
func TestSettingSvc_GetFallbackToRegistryDefault(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db, nil)
	ctx := context.Background()

	v, err := svc.GetBool(ctx, model.KeyRegistrationEnabled)
	require.NoError(t, err)
	require.True(t, v, "注册开关默认应为开启")

	v, err = svc.GetBool(ctx, model.KeyMaintenanceMode)
	require.NoError(t, err)
	require.False(t, v, "维护模式默认应为关闭")

	msg, err := svc.GetString(ctx, model.KeyMaintenanceMessage)
	require.NoError(t, err)
	require.Equal(t, "系统维护中，请稍后再试", msg)

	// 未登记的键：明确报 not found，而非空值
	_, err = svc.GetString(ctx, "no_such_key")
	require.ErrorIs(t, err, ErrSettingNotFound)
}

// TestSettingSvc_GetIntAndDuration 类型化读取 int / duration（duration 走库内行，验证行侧类型解析）
func TestSettingSvc_GetIntAndDuration(t *testing.T) {
	db := newTestDB(t)
	repo := repository.NewSettingRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, &model.Setting{
		Key: "session_ttl", Value: "10m", ValueType: model.ValueTypeDuration,
	}))
	require.NoError(t, repo.Create(ctx, &model.Setting{
		Key: "max_upload_mb", Value: "64", ValueType: model.ValueTypeInt,
	}))
	svc := newSvc(db, nil)

	d, err := svc.GetDuration(ctx, "session_ttl")
	require.NoError(t, err)
	require.Equal(t, 10*time.Minute, d)

	n, err := svc.GetInt(ctx, "max_upload_mb")
	require.NoError(t, err)
	require.Equal(t, int64(64), n)

	// 类型不匹配时报错而不是静默零值
	require.NoError(t, repo.Create(ctx, &model.Setting{Key: "bad_int", Value: "abc", ValueType: model.ValueTypeInt}))
	_, err = svc.GetInt(ctx, "bad_int")
	require.ErrorIs(t, err, ErrInvalidSettingValue)
}

// TestSettingSvc_SeedDefaults_Idempotent 种子幂等：仅补缺失键，二次执行不覆盖已修改值
func TestSettingSvc_SeedDefaults_Idempotent(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db, nil)
	ctx := context.Background()

	created, err := svc.SeedDefaults(ctx)
	require.NoError(t, err)
	require.Equal(t, len(model.Registry()), created)

	// 用户把注册开关改为关闭
	_, err = svc.UpdateBulk(ctx, []dto.SettingUpdateItem{{Key: model.KeyRegistrationEnabled, Value: "false"}})
	require.NoError(t, err)

	// 再次种子：不得有任何新建，也不得把 false 覆盖回 true
	created, err = svc.SeedDefaults(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, created)

	v, err := svc.GetBool(ctx, model.KeyRegistrationEnabled)
	require.NoError(t, err)
	require.False(t, v, "种子不得覆盖用户已修改的值")
}

// TestSettingSvc_UpdateBulk_TypeValidation 白名单 + 类型校验：非法键/值整批拒绝，不产生半应用
func TestSettingSvc_UpdateBulk_TypeValidation(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db, nil)
	ctx := context.Background()

	// 合法值 + 未登记键混在一批：整批失败，合法项也不得落库
	_, err := svc.UpdateBulk(ctx, []dto.SettingUpdateItem{
		{Key: model.KeyRegistrationEnabled, Value: "false"},
		{Key: "hacker_key", Value: "1"},
	})
	require.ErrorIs(t, err, ErrUnknownSettingKey)
	v, err := svc.GetBool(ctx, model.KeyRegistrationEnabled)
	require.NoError(t, err)
	require.True(t, v, "整批校验失败时合法项不得被写入")

	// bool 键传非法值
	_, err = svc.UpdateBulk(ctx, []dto.SettingUpdateItem{{Key: model.KeyMaintenanceMode, Value: "yes-please"}})
	require.ErrorIs(t, err, ErrInvalidSettingValue)

	// 合法批量：bool 支持 "1" 形态并归一化为 "true" 入库
	views, err := svc.UpdateBulk(ctx, []dto.SettingUpdateItem{
		{Key: model.KeyMaintenanceMode, Value: "1"},
		{Key: model.KeyMaintenanceMessage, Value: "今晚 22:00 升级"},
	})
	require.NoError(t, err)
	v, err = svc.GetBool(ctx, model.KeyMaintenanceMode)
	require.NoError(t, err)
	require.True(t, v)

	byKey := map[string]*dto.SettingView{}
	for _, vw := range views {
		byKey[vw.Key] = vw
	}
	require.Equal(t, sourceDB, byKey[model.KeyMaintenanceMode].Source)
	require.Equal(t, "true", byKey[model.KeyMaintenanceMode].Value, "归一化后入库形态应为规范 bool 字符串")
	require.Equal(t, "今晚 22:00 升级", byKey[model.KeyMaintenanceMessage].Value)
}

// TestSettingSvc_RedisCacheHitAndInvalidate 整表缓存：读后落 Redis；写后失效；缓存陈旧时不穿透
func TestSettingSvc_RedisCacheHitAndInvalidate(t *testing.T) {
	db := newTestDB(t)
	rdb, mr := newTestRedis(t)
	svc := newSvc(db, rdb)
	ctx := context.Background()

	// 首次读触发缓存重建
	v, err := svc.GetBool(ctx, model.KeyRegistrationEnabled)
	require.NoError(t, err)
	require.True(t, v)
	require.True(t, mr.Exists(CacheKeyAll), "读取后整表缓存键应已建立")

	// 绕过服务直接改库：应仍读到缓存值（证明读路径命中缓存、未穿透 DB）
	require.NoError(t, repository.NewSettingRepository(db).UpdateValue(ctx, model.KeyRegistrationEnabled, "false"))
	v, err = svc.GetBool(ctx, model.KeyRegistrationEnabled)
	require.NoError(t, err)
	require.True(t, v, "缓存有效期内应读到旧值，而非穿透查库")

	// 通过服务更新：写后主动失效，再读必须看到新值
	_, err = svc.UpdateBulk(ctx, []dto.SettingUpdateItem{{Key: model.KeyRegistrationEnabled, Value: "false"}})
	require.NoError(t, err)
	v, err = svc.GetBool(ctx, model.KeyRegistrationEnabled)
	require.NoError(t, err)
	require.False(t, v)
}

// TestSettingSvc_RedisUnavailableFailOpen Redis 不可用（nil）时读写均降级直查 DB，不影响正确性
func TestSettingSvc_RedisUnavailableFailOpen(t *testing.T) {
	db := newTestDB(t)
	deadRDB := &cache.Redis{} // Client 为 nil：IsAvailable false
	svc := newSvc(db, deadRDB)
	ctx := context.Background()

	_, err := svc.UpdateBulk(ctx, []dto.SettingUpdateItem{{Key: model.KeyRegistrationEnabled, Value: "0"}})
	require.NoError(t, err, "Redis 不可用时写入不得失败")
	v, err := svc.GetBool(ctx, model.KeyRegistrationEnabled)
	require.NoError(t, err)
	require.False(t, v)
}

// TestSettingSvc_ConsumerFallbacks 消费者便利方法：任何故障回退安全默认，配置中心挂掉不得拦人
func TestSettingSvc_ConsumerFallbacks(t *testing.T) {
	ctx := context.Background()

	// DB 正常：读到真实值
	db := newTestDB(t)
	svc := newSvc(db, nil)
	repo := repository.NewSettingRepository(db)
	require.NoError(t, repo.Create(ctx, &model.Setting{
		Key: model.KeyRegistrationEnabled, Value: "false", ValueType: model.ValueTypeBool,
	}))
	require.False(t, svc.RegistrationEnabled(ctx))

	enabled, msg := svc.MaintenanceState(ctx)
	require.False(t, enabled)
	require.Empty(t, msg, "未开启维护时不返回文案")

	// DB 故障：RegistrationEnabled fail-open 回退默认 true；MaintenanceState 按未开启处理
	failing := NewSettingService(failRepo{}, nil)
	require.True(t, failing.RegistrationEnabled(ctx), "读取失败必须回退注册表默认值放行")
	enabled, _ = failing.MaintenanceState(ctx)
	require.False(t, enabled, "读取失败必须按未开启处理，不得误锁全站")
}

// TestSettingSvc_GetPublicMap 公开视角：仅 IsPublic 键、值按类型还原
func TestSettingSvc_GetPublicMap(t *testing.T) {
	db := newTestDB(t)
	svc := newSvc(db, nil)
	ctx := context.Background()

	out, err := svc.GetPublicMap(ctx)
	require.NoError(t, err)
	require.Equal(t, true, out[model.KeyRegistrationEnabled], "bool 应以 JSON 布尔暴露")
	require.Equal(t, false, out[model.KeyMaintenanceMode])
	require.Equal(t, "系统维护中，请稍后再试", out[model.KeyMaintenanceMessage])
	require.Len(t, out, len(model.Registry()), "当前登记键全部为公开键")

	// 私有化一键后不再暴露
	repo := repository.NewSettingRepository(db)
	require.NoError(t, repo.Create(ctx, &model.Setting{
		Key: "private_key", Value: "s", ValueType: model.ValueTypeString, IsPublic: false,
	}))
	out, err = svc.GetPublicMap(ctx)
	require.NoError(t, err)
	_, exists := out["private_key"]
	require.False(t, exists, "IsPublic=false 的键不得出现在公开配置中")
}

// failRepo 恒错的仓储桩，验证各读取路径的失败回退语义
type failRepo struct{}

var errFailRepo = errors.New("db down")

func (failRepo) GetAll(context.Context) ([]*model.Setting, error)         { return nil, errFailRepo }
func (failRepo) GetByKey(context.Context, string) (*model.Setting, error) { return nil, errFailRepo }
func (failRepo) Create(context.Context, *model.Setting) error             { return errFailRepo }
func (failRepo) UpdateValue(context.Context, string, string) error        { return errFailRepo }
func (failRepo) Upsert(context.Context, *model.Setting) error             { return errFailRepo }

var _ repository.SettingRepository = failRepo{}
