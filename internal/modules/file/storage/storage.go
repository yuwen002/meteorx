package storage

import (
	"context"
	"io"

	"meteorx/internal/config"
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

// NewStorage 根据配置创建对应的存储实例
// signKey 用于本地文件访问 URL 签名（取 file.sign_key，与 JWT 登录密钥解耦）；为空则不签名（仅限调试场景）
// 当前默认实现 local；后续扩展 oss / s3 时在此处增加分支
func NewStorage(cfg config.FileConfig, signKey string) Storage {
	storageType := cfg.StorageType
	if storageType == "" {
		storageType = "local"
	}

	switch storageType {
	case "local", "":
		return NewLocalStorage(cfg.UploadPath, cfg.UploadURL, signKey)
	default:
		// 未知类型时降级为本地存储，避免启动失败
		return NewLocalStorage(cfg.UploadPath, cfg.UploadURL, signKey)
	}
}
