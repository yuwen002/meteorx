package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"meteorx/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Storage 基于 AWS S3（及 S3 兼容端点，如 MinIO）的对象存储实现。
type S3Storage struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
	publicBaseURL string
	presignTTL    int64
	pathPrefix    string
}

// NewS3Storage 依据配置创建 S3 存储实例。
// Endpoint 留空走 AWS 默认端点；填写则适配 MinIO 等 S3 兼容服务（通常配合 ForcePathStyle）。
// 凭证：access_key/secret_key 均填写走静态凭证；均留空走默认凭证链（环境变量/共享配置/实例角色 IRSA）；
// 仅填其一视为配置错误。
func NewS3Storage(ctx context.Context, cfg config.FileConfig) (*S3Storage, error) {
	cloud := cfg.Cloud
	if cloud.Bucket == "" {
		return nil, errors.New("s3 storage: bucket is required")
	}

	region := cloud.Region
	if region == "" {
		region = "us-east-1"
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}
	switch {
	case cloud.AccessKey != "" && cloud.SecretKey != "":
		// 显式静态凭证：LoadDefaultConfig 不会再探测 IMDS / 环境等其他凭证链
		loadOpts = append(loadOpts,
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cloud.AccessKey, cloud.SecretKey, "")))
	case cloud.AccessKey == "" && cloud.SecretKey == "":
		// 均留空：走默认凭证链（env / shared config / IMDS / IRSA 等），无需额外配置
	default:
		return nil, errors.New("s3 storage: access_key and secret_key must both be set, or both empty to use the default credential chain")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("s3 storage: load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cloud.Endpoint != "" {
			o.BaseEndpoint = aws.String(cloud.Endpoint)
		}
		o.UsePathStyle = cloud.ForcePathStyle
	})

	return &S3Storage{
		client:        client,
		presignClient: s3.NewPresignClient(client),
		bucket:        cloud.Bucket,
		publicBaseURL: cloud.PublicBaseURL,
		presignTTL:    normalizeTTL(cloud.PresignTTL),
		pathPrefix:    cloud.PathPrefix,
	}, nil
}

// Upload 上传对象到 S3，返回对象键。
func (s *S3Storage) Upload(ctx context.Context, reader io.Reader, originalName string) (string, error) {
	key := buildObjectKey(s.pathPrefix, originalName)
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   reader,
	}
	if ct := mime.TypeByExtension(filepath.Ext(originalName)); ct != "" {
		input.ContentType = aws.String(ct)
	}
	if _, err := s.client.PutObject(ctx, input); err != nil {
		return "", fmt.Errorf("s3 upload: %w", err)
	}
	return key, nil
}

// Download 从 S3 下载对象，返回读取流（调用方负责 Close）。
func (s *S3Storage) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 download: %w", err)
	}
	return out.Body, nil
}

// Delete 删除 S3 对象；对象不存在视为成功。
func (s *S3Storage) Delete(ctx context.Context, path string) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}); err != nil {
		return fmt.Errorf("s3 delete: %w", err)
	}
	return nil
}

// Exists 通过 HeadObject 判断对象是否存在。
func (s *S3Storage) Exists(ctx context.Context, path string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		var nf *types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, fmt.Errorf("s3 exists: %w", err)
	}
	return true, nil
}

// GetURL 返回对象访问 URL：配置了公开基址则返回公开直链，否则尽力返回带默认有效期的预签名 URL。
func (s *S3Storage) GetURL(path string) string {
	if s.publicBaseURL != "" {
		return joinBaseURL(s.publicBaseURL, path)
	}
	// 私有桶尽力签名；失败时回退为规范对象地址（尽力而为，GetURL 无 error 语义）
	url, err := s.PresignURL(context.Background(), path, s.presignTTL)
	if err != nil {
		return s.canonicalURL(path)
	}
	return url
}

// PresignURL 生成带签名的临时访问 URL（expiration 为有效期秒数）。
func (s *S3Storage) PresignURL(ctx context.Context, path string, expiration int64) (string, error) {
	ttl := normalizeTTL(expiration)
	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}, func(o *s3.PresignOptions) { o.Expires = time.Duration(ttl) * time.Second })
	if err != nil {
		return "", fmt.Errorf("s3 presign: %w", err)
	}
	return req.URL, nil
}

// Type 返回存储类型。
func (s *S3Storage) Type() string {
	return "s3"
}

// canonicalURL 构造规范对象地址（虚拟主机风格），作为 GetURL 签名失败时的兜底。
func (s *S3Storage) canonicalURL(path string) string {
	key := strings.TrimPrefix(path, "/")
	if s.publicBaseURL != "" {
		return joinBaseURL(s.publicBaseURL, key)
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, key)
}
