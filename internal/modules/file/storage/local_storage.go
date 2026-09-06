package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"meteorx/internal/common/signedurl"
	"meteorx/pkg/idgen"
)

// LocalStorage 本地文件存储实现
type LocalStorage struct {
	basePath string
	baseURL  string
	// signKey 为空时 URL 不带签名（保持历史公开行为，仅建议内网/调试场景）
	signKey string
}

// NewLocalStorage 创建本地存储实例；signKey 用于对访问 URL 做 HMAC 签名
func NewLocalStorage(basePath, baseURL, signKey string) *LocalStorage {
	return &LocalStorage{
		basePath: basePath,
		baseURL:  baseURL,
		signKey:  signKey,
	}
}

// Upload 上传文件到本地存储
func (s *LocalStorage) Upload(ctx context.Context, reader io.Reader, originalName string) (string, error) {
	// 生成唯一文件名
	ext := filepath.Ext(originalName)
	fileName := idgen.New() + ext

	// 构建完整路径
	fullPath := filepath.Join(s.basePath, fileName)

	// 确保目录存在
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建文件
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// 复制数据
	if _, err := io.Copy(file, reader); err != nil {
		// 如果复制失败，删除已创建的文件
		os.Remove(fullPath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fileName, nil
}

// Download 从本地存储下载文件
func (s *LocalStorage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, path)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// Delete 删除本地文件
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.basePath, path)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// Exists 检查文件是否存在
func (s *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(s.basePath, path)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// GetURL 获取文件访问URL；配置了签名密钥时返回带短时效签名的 URL
func (s *LocalStorage) GetURL(path string) string {
	if s.signKey == "" {
		return strings.TrimSuffix(s.baseURL, "/") + "/" + path
	}
	return signedurl.Build(s.baseURL, path, s.signKey, signedurl.Expires(signedurl.DefaultTTL))
}

// PresignURL 生成带签名的临时访问 URL（expiration 为有效期秒数）
func (s *LocalStorage) PresignURL(_ context.Context, path string, expiration int64) (string, error) {
	if s.signKey == "" {
		return s.GetURL(path), nil
	}
	if expiration <= 0 {
		expiration = int64(signedurl.DefaultTTL.Seconds())
	}
	expires := time.Now().Add(time.Duration(expiration) * time.Second).Unix()
	return signedurl.Build(s.baseURL, path, s.signKey, expires), nil
}

// Type 返回存储类型
func (s *LocalStorage) Type() string {
	return "local"
}

// UploadFromMultipartFile 从multipart.FileHeader上传文件
func (s *LocalStorage) UploadFromMultipartFile(ctx context.Context, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	return s.Upload(ctx, src, file.Filename)
}
