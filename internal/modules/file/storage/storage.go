// Package storage 定义文件存储抽象接口，支持本地磁盘和云存储等多种后端实现。
package storage

import (
	"context"
	"io"
	"strings"
	"sync"

	"meteorx/internal/config"
	"meteorx/pkg/logger"
)

// Storage 文件存储接口
type Storage interface {
	// Upload 上传文件，返回存储路径（相对路径）
	Upload(ctx context.Context, reader io.Reader, originalName string) (string, error)

	// Download 下载文件
	Download(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete 删除文件
	Delete(ctx context.Context, path string) error

	// Exists 检查文件是否存在
	Exists(ctx context.Context, path string) (bool, error)

	// GetURL 获取文件访问URL（公开访问地址）
	GetURL(path string) string

	// PresignURL 生成临时签名 URL（用于前端直传/直链访问）
	// expiration 为签名有效期；local 存储实现返回 GetURL，云存储返回带签名的 URL
	PresignURL(ctx context.Context, path string, expiration int64) (string, error)

	// Type 返回存储类型标识（local / oss / s3 等）
	Type() string
}

// NewStorage 根据配置创建对应的存储实例（默认后端）。
// signKey 用于本地文件访问 URL 签名（取 file.sign_key，与 JWT 登录密钥解耦）；为空则不签名（仅限调试场景）
// 支持 local / oss / s3；云存储初始化失败（如缺少必填配置）时返回 error，由调用方决定降级策略；
// 未知类型降级为本地存储，避免启动失败。
func NewStorage(cfg config.FileConfig, signKey string) (Storage, error) {
	return buildStorage(normalizeStorageType(cfg.StorageType), cfg, signKey)
}

// normalizeStorageType 归一化存储类型：转小写、去空白，空值回退 local。
func normalizeStorageType(s string) string {
	t := strings.ToLower(strings.TrimSpace(s))
	if t == "" {
		return "local"
	}
	return t
}

// buildStorage 按具体类型构造存储实例；未知类型降级为本地存储，避免启动失败。
func buildStorage(storageType string, cfg config.FileConfig, signKey string) (Storage, error) {
	switch storageType {
	case "oss":
		return NewOSSStorage(cfg)
	case "s3":
		return NewS3Storage(context.Background(), cfg)
	case "local":
		return NewLocalStorage(cfg.UploadPath, cfg.UploadURL, signKey), nil
	default:
		// 未知类型时降级为本地存储，避免启动失败
		return NewLocalStorage(cfg.UploadPath, cfg.UploadURL, signKey), nil
	}
}

// Registry 存储后端注册表：持有一个「默认后端」（当前配置类型），并可按记录中的
// StorageType 惰性解析出对应后端。用于支持「切换存储类型后仍能读取历史文件」：
// 上传等写操作走默认后端；读取/删除某条记录时按其 StorageType 路由到原后端，
// 若该后端无法构造（如缺少配置）则告警并回退默认后端（尽力而为）。
type Registry struct {
	def     Storage
	cfg     config.FileConfig
	signKey string

	mu    sync.Mutex
	extra map[string]Storage // 已成功构造的非默认后端
	fails map[string]bool    // 构造失败过的类型，避免重复尝试
}

// NewRegistry 基于默认后端与完整配置创建注册表。
func NewRegistry(def Storage, cfg config.FileConfig, signKey string) *Registry {
	return &Registry{
		def:     def,
		cfg:     cfg,
		signKey: signKey,
		extra:   make(map[string]Storage),
		fails:   make(map[string]bool),
	}
}

// Default 返回当前配置的默认后端（上传等写操作使用）。
func (r *Registry) Default() Storage { return r.def }

// Resolve 按存储类型返回后端：空类型或默认类型直接返回默认后端（常见路径，无锁）；
// 否则惰性构造对应后端，失败则告警并回退默认后端。
func (r *Registry) Resolve(storageType string) Storage {
	if storageType == "" || storageType == r.def.Type() {
		return r.def
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.extra[storageType]; ok {
		return b
	}
	if r.fails[storageType] {
		return r.def
	}
	b, err := buildStorage(storageType, r.cfg, r.signKey)
	if err != nil {
		r.fails[storageType] = true
		logger.Warnf("[file/storage] resolve backend %q failed, falling back to %q: %v", storageType, r.def.Type(), err)
		return r.def
	}
	r.extra[storageType] = b
	return b
}
