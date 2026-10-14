package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"meteorx/internal/config"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSStorage 基于阿里云 OSS 的对象存储实现。
type OSSStorage struct {
	bucket        *oss.Bucket
	bucketName    string
	endpoint      string
	publicBaseURL string
	presignTTL    int64
	pathPrefix    string
}

// NewOSSStorage 依据配置创建 OSS 存储实例。Endpoint 形如 oss-cn-hangzhou.aliyuncs.com。
func NewOSSStorage(cfg config.FileConfig) (*OSSStorage, error) {
	cloud := cfg.Cloud
	if cloud.Endpoint == "" {
		return nil, fmt.Errorf("oss storage: endpoint is required")
	}
	if cloud.Bucket == "" {
		return nil, fmt.Errorf("oss storage: bucket is required")
	}
	if cloud.AccessKey == "" || cloud.SecretKey == "" {
		return nil, fmt.Errorf("oss storage: access_key and secret_key are required")
	}

	client, err := oss.New(cloud.Endpoint, cloud.AccessKey, cloud.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("oss storage: new client: %w", err)
	}
	bucket, err := client.Bucket(cloud.Bucket)
	if err != nil {
		return nil, fmt.Errorf("oss storage: open bucket: %w", err)
	}

	return &OSSStorage{
		bucket:        bucket,
		bucketName:    cloud.Bucket,
		endpoint:      cloud.Endpoint,
		publicBaseURL: cloud.PublicBaseURL,
		presignTTL:    normalizeTTL(cloud.PresignTTL),
		pathPrefix:    cloud.PathPrefix,
	}, nil
}

// Upload 上传对象到 OSS，返回对象键。
func (s *OSSStorage) Upload(_ context.Context, reader io.Reader, originalName string) (string, error) {
	key := buildObjectKey(s.pathPrefix, originalName)
	var opts []oss.Option
	if ct := mime.TypeByExtension(filepath.Ext(originalName)); ct != "" {
		opts = append(opts, oss.ContentType(ct))
	}
	if err := s.bucket.PutObject(key, reader, opts...); err != nil {
		return "", fmt.Errorf("oss upload: %w", err)
	}
	return key, nil
}

// Download 从 OSS 下载对象，返回读取流（调用方负责 Close）。
func (s *OSSStorage) Download(_ context.Context, path string) (io.ReadCloser, error) {
	body, err := s.bucket.GetObject(path)
	if err != nil {
		return nil, fmt.Errorf("oss download: %w", err)
	}
	return body, nil
}

// Delete 删除 OSS 对象。
func (s *OSSStorage) Delete(_ context.Context, path string) error {
	if err := s.bucket.DeleteObject(path); err != nil {
		return fmt.Errorf("oss delete: %w", err)
	}
	return nil
}

// Exists 判断 OSS 对象是否存在。
func (s *OSSStorage) Exists(_ context.Context, path string) (bool, error) {
	ok, err := s.bucket.IsObjectExist(path)
	if err != nil {
		return false, fmt.Errorf("oss exists: %w", err)
	}
	return ok, nil
}

// GetURL 返回对象访问 URL：配置了公开基址则返回公开直链，否则尽力返回带默认有效期的预签名 URL。
func (s *OSSStorage) GetURL(path string) string {
	if s.publicBaseURL != "" {
		return joinBaseURL(s.publicBaseURL, path)
	}
	url, err := s.PresignURL(context.Background(), path, s.presignTTL)
	if err != nil {
		return s.canonicalURL(path)
	}
	return url
}

// PresignURL 生成带签名的临时访问 URL（expiration 为有效期秒数）。
func (s *OSSStorage) PresignURL(_ context.Context, path string, expiration int64) (string, error) {
	ttl := normalizeTTL(expiration)
	url, err := s.bucket.SignURL(path, oss.HTTPGet, ttl)
	if err != nil {
		return "", fmt.Errorf("oss presign: %w", err)
	}
	return url, nil
}

// Type 返回存储类型。
func (s *OSSStorage) Type() string {
	return "oss"
}

// canonicalURL 构造规范对象地址，作为 GetURL 签名失败时的兜底。
func (s *OSSStorage) canonicalURL(path string) string {
	key := strings.TrimPrefix(path, "/")
	if s.publicBaseURL != "" {
		return joinBaseURL(s.publicBaseURL, key)
	}
	return fmt.Sprintf("https://%s.%s/%s", s.bucketName, strings.TrimPrefix(s.endpoint, "https://"), key)
}
