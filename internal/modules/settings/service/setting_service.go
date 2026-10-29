// Package service 平台运行时配置服务：DB 行 → 注册表内置默认值的三层回退解析
// （环境兜底由 bootstrap 启动时的 SeedSettings 落库完成），类型化读取与批量更新，
// 整表 JSON 缓存于 Redis 单键、写后失效，Redis 不可用时直查 DB（fail-open）。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"meteorx/internal/cache"
	"meteorx/internal/modules/settings/dto"
	"meteorx/internal/modules/settings/model"
	"meteorx/internal/modules/settings/repository"
	"meteorx/pkg/logger"
)

// 业务错误哨兵
var (
	ErrSettingNotFound     = errors.New("setting not found")
	ErrUnknownSettingKey   = errors.New("unknown setting key")
	ErrInvalidSettingValue = errors.New("invalid setting value")
)

const (
	// CacheKeyAll 整表缓存的 Redis 单键
	CacheKeyAll = "settings:all"
	// CacheTTL 整表缓存过期时长：写后主动失效之外的最终一致兜底
	CacheTTL = 5 * time.Minute

	sourceDB      = "db"
	sourceDefault = "default"
)

// SettingService 平台运行时配置服务
type SettingService struct {
	repo repository.SettingRepository
	rdb  *cache.Redis
}

// NewSettingService 创建配置服务实例。rdb 可为 nil 或不可用状态，此时跳过缓存直查 DB。
func NewSettingService(repo repository.SettingRepository, rdb *cache.Redis) *SettingService {
	return &SettingService{repo: repo, rdb: rdb}
}

// loadRows 读取全表配置行，优先命中 Redis 整表缓存。
// DB 是事实源：缓存损坏/不可用时回源 DB 并重建缓存；DB 错误原样返回（不降级）。
func (s *SettingService) loadRows(ctx context.Context) (map[string]*model.Setting, error) {
	if s.rdb.IsAvailable() {
		if val, err := s.rdb.Get(ctx, CacheKeyAll); err == nil {
			var rows []*model.Setting
			if json.Unmarshal([]byte(val), &rows) == nil {
				return settingsToMap(rows), nil
			}
			// 缓存内容损坏：忽略，回源 DB 重建
		}
	}

	rows, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	if s.rdb.IsAvailable() {
		if data, mErr := json.Marshal(rows); mErr == nil {
			if sErr := s.rdb.Set(ctx, CacheKeyAll, data, CacheTTL); sErr != nil {
				logger.Warnf("[Settings] 整表缓存写入失败(不影响正确性，直查DB): %v", sErr)
			}
		}
	}
	return settingsToMap(rows), nil
}

// invalidateCache 使整表缓存失效；失败仅告警（脏读窗口不超过 CacheTTL）
func (s *SettingService) invalidateCache(ctx context.Context) {
	if !s.rdb.IsAvailable() {
		return
	}
	if err := s.rdb.Delete(ctx, CacheKeyAll); err != nil && !errors.Is(err, cache.ErrRedisUnavailable) {
		logger.Warnf("[Settings] 整表缓存失效失败(TTL 内可能读到旧值): %v", err)
	}
}

func settingsToMap(rows []*model.Setting) map[string]*model.Setting {
	m := make(map[string]*model.Setting, len(rows))
	for _, r := range rows {
		m[r.Key] = r
	}
	return m
}

// resolve 三层回退解析单键有效值：DB 行 → 注册表内置默认。
// 返回值、生效类型、来源(db/default)；键既无行也未登记时返回 ErrSettingNotFound。
func (s *SettingService) resolve(ctx context.Context, key string) (value string, valueType string, source string, err error) {
	rows, err := s.loadRows(ctx)
	if err != nil {
		return "", "", "", err
	}
	def, defOK := model.Lookup(key)
	if row, ok := rows[key]; ok {
		vt := row.ValueType
		if vt == "" && defOK {
			vt = def.ValueType
		}
		if vt == "" {
			vt = model.ValueTypeString
		}
		return row.Value, vt, sourceDB, nil
	}
	if defOK {
		return def.DefaultValue, def.ValueType, sourceDefault, nil
	}
	return "", "", "", fmt.Errorf("%w: %s", ErrSettingNotFound, key)
}

// GetString 读取字符串配置值
func (s *SettingService) GetString(ctx context.Context, key string) (string, error) {
	v, _, _, err := s.resolve(ctx, key)
	return v, err
}

// GetBool 读取布尔配置值（接受 true/false/1/0 等 ParseBool 形态）
func (s *SettingService) GetBool(ctx context.Context, key string) (bool, error) {
	v, _, _, err := s.resolve(ctx, key)
	if err != nil {
		return false, err
	}
	b, pErr := strconv.ParseBool(strings.TrimSpace(v))
	if pErr != nil {
		return false, fmt.Errorf("%w: %s=%q 无法解析为 bool", ErrInvalidSettingValue, key, v)
	}
	return b, nil
}

// GetInt 读取整数配置值
func (s *SettingService) GetInt(ctx context.Context, key string) (int64, error) {
	v, _, _, err := s.resolve(ctx, key)
	if err != nil {
		return 0, err
	}
	n, pErr := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if pErr != nil {
		return 0, fmt.Errorf("%w: %s=%q 无法解析为 int", ErrInvalidSettingValue, key, v)
	}
	return n, nil
}

// GetDuration 读取时长配置值（Go duration 字符串形态，如 "10m"）
func (s *SettingService) GetDuration(ctx context.Context, key string) (time.Duration, error) {
	v, _, _, err := s.resolve(ctx, key)
	if err != nil {
		return 0, err
	}
	d, pErr := time.ParseDuration(strings.TrimSpace(v))
	if pErr != nil {
		return 0, fmt.Errorf("%w: %s=%q 无法解析为 duration", ErrInvalidSettingValue, key, v)
	}
	return d, nil
}

// RegistrationEnabled 是否开放自助注册（注册接口的强制开关）。
// 任意读取/解析失败均回退注册表默认值：配置中心故障不得阻断原本放行的注册。
func (s *SettingService) RegistrationEnabled(ctx context.Context) bool {
	v, err := s.GetBool(ctx, model.KeyRegistrationEnabled)
	if err != nil {
		def, _ := model.Lookup(model.KeyRegistrationEnabled)
		fb, _ := strconv.ParseBool(def.DefaultValue)
		logger.Warnf("[Settings] 读取 %s 失败，回退默认值 %v: %v", model.KeyRegistrationEnabled, fb, err)
		return fb
	}
	return v
}

// MaintenanceState 维护模式当前状态（开关 + 提示文案）。
// 任意失败均按「未开启」处理：配置中心故障不得把全站误锁死。
func (s *SettingService) MaintenanceState(ctx context.Context) (enabled bool, message string) {
	v, err := s.GetBool(ctx, model.KeyMaintenanceMode)
	if err != nil {
		logger.Warnf("[Settings] 读取 %s 失败，按未开启处理: %v", model.KeyMaintenanceMode, err)
		return false, ""
	}
	if !v {
		return false, ""
	}
	msg, mErr := s.GetString(ctx, model.KeyMaintenanceMessage)
	if mErr != nil {
		def, _ := model.Lookup(model.KeyMaintenanceMessage)
		msg = def.DefaultValue
	}
	return true, msg
}

// ListAll 管理视角：登记键（含未落库、显示默认值与来源）+ DB 中的额外历史行
func (s *SettingService) ListAll(ctx context.Context) ([]*dto.SettingView, error) {
	rows, err := s.loadRows(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]*dto.SettingView, 0, len(model.Registry())+len(rows))
	seen := make(map[string]bool, len(rows))
	for _, def := range model.Registry() {
		seen[def.Key] = true
		if row, ok := rows[def.Key]; ok {
			views = append(views, rowToView(row, def.DefaultValue, def.Description))
			continue
		}
		views = append(views, &dto.SettingView{
			Key:          def.Key,
			Value:        def.DefaultValue,
			ValueType:    def.ValueType,
			DefaultValue: def.DefaultValue,
			IsPublic:     def.IsPublic,
			Source:       sourceDefault,
			Description:  def.Description,
		})
	}

	// 注册表白名单之外仍可能残留历史行（键曾被登记后又移除）：按键排序稳定展示
	extras := make([]string, 0, len(rows))
	for key := range rows {
		if !seen[key] {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	for _, key := range extras {
		views = append(views, rowToView(rows[key], "", ""))
	}
	return views, nil
}

func rowToView(row *model.Setting, defaultValue, description string) *dto.SettingView {
	return &dto.SettingView{
		Key:          row.Key,
		Value:        row.Value,
		ValueType:    row.ValueType,
		DefaultValue: defaultValue,
		IsPublic:     row.IsPublic,
		Source:       sourceDB,
		Description:  description,
		UpdatedAt:    row.UpdatedAt.Format(time.RFC3339),
	}
}

// GetPublicMap 公开视角：仅返回 IsPublic 键，值按类型解析为 JSON 友好形态
func (s *SettingService) GetPublicMap(ctx context.Context) (map[string]interface{}, error) {
	rows, err := s.loadRows(ctx)
	if err != nil {
		return nil, err
	}

	out := make(map[string]interface{})
	// 登记键（DB 行优先，缺行回退默认值）
	for _, def := range model.Registry() {
		if !def.IsPublic {
			continue
		}
		val, vt := def.DefaultValue, def.ValueType
		if row, ok := rows[def.Key]; ok {
			val, vt = row.Value, row.ValueType
		}
		out[def.Key] = parseTyped(vt, val)
	}
	// 历史行中的公开键（未登记也暴露，保持与库内一致）
	for key, row := range rows {
		if row.IsPublic {
			if _, ok := out[key]; !ok {
				out[key] = parseTyped(row.ValueType, row.Value)
			}
		}
	}
	return out, nil
}

// parseTyped 按类型把字符串值还原为 JSON 值；解析失败时原样返回字符串（best-effort）
func parseTyped(valueType, val string) interface{} {
	switch valueType {
	case model.ValueTypeBool:
		if b, err := strconv.ParseBool(strings.TrimSpace(val)); err == nil {
			return b
		}
	case model.ValueTypeInt:
		if n, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil {
			return n
		}
	}
	return val
}

// UpdateBulk 批量更新配置：整批校验（键必须已登记、值必须匹配类型）后一次写入，
// 校验失败不产生半应用状态；写入成功后主动失效整表缓存。
func (s *SettingService) UpdateBulk(ctx context.Context, items []dto.SettingUpdateItem) ([]*dto.SettingView, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("更新项不能为空")
	}

	// 第一遍：整批校验并归一化
	pending := make(map[string]string, len(items))
	for _, it := range items {
		def, ok := model.Lookup(it.Key)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownSettingKey, it.Key)
		}
		canon, vErr := normalizeValue(def.ValueType, it.Value)
		if vErr != nil {
			return nil, fmt.Errorf("%w: %s=%q", ErrInvalidSettingValue, it.Key, it.Value)
		}
		pending[it.Key] = canon
	}

	// 第二遍：按键排序写入，保证多行更新顺序确定
	keys := make([]string, 0, len(pending))
	for k := range pending {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		def, _ := model.Lookup(key)
		setting := &model.Setting{Key: key, Value: pending[key], ValueType: def.ValueType, IsPublic: def.IsPublic}
		if err := s.repo.Upsert(ctx, setting); err != nil {
			return nil, err
		}
	}

	s.invalidateCache(ctx)
	return s.ListAll(ctx)
}

// normalizeValue 按登记类型校验并归一化字符串值，返回入库的规范形态
func normalizeValue(valueType, val string) (string, error) {
	trimmed := strings.TrimSpace(val)
	switch valueType {
	case model.ValueTypeBool:
		b, err := strconv.ParseBool(trimmed)
		if err != nil {
			return "", fmt.Errorf("值 %q 不是合法布尔", val)
		}
		return strconv.FormatBool(b), nil
	case model.ValueTypeInt:
		n, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return "", fmt.Errorf("值 %q 不是合法整数", val)
		}
		return strconv.FormatInt(n, 10), nil
	case model.ValueTypeDuration:
		d, err := time.ParseDuration(trimmed)
		if err != nil {
			return "", fmt.Errorf("值 %q 不是合法时长", val)
		}
		return d.String(), nil
	default:
		return val, nil
	}
}

// SeedDefaults 幂等初始化：仅插入注册表中缺失的键，绝不覆盖用户已修改的值。
// 返回本次新建的键数。
func (s *SettingService) SeedDefaults(ctx context.Context) (int, error) {
	created := 0
	for _, def := range model.Registry() {
		existing, err := s.repo.GetByKey(ctx, def.Key)
		if err != nil {
			return created, err
		}
		if existing != nil {
			continue
		}
		if err := s.repo.Create(ctx, &model.Setting{
			Key:       def.Key,
			Value:     def.DefaultValue,
			ValueType: def.ValueType,
			IsPublic:  def.IsPublic,
		}); err != nil {
			return created, err
		}
		created++
	}
	if created > 0 {
		s.invalidateCache(ctx)
	}
	return created, nil
}
