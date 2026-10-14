package storage

import (
	"testing"

	"meteorx/internal/config"
)

// fullCloudCfg 构造一个各字段齐备的云存储配置，按需覆盖个别字段以触发校验分支。
func fullCloudCfg(storageType string) config.FileConfig {
	cfg := config.FileConfig{StorageType: storageType}
	cfg.Cloud.Endpoint = "oss-cn-hangzhou.aliyuncs.com"
	cfg.Cloud.AccessKey = "test-access-key"
	cfg.Cloud.SecretKey = "test-secret-key"
	cfg.Cloud.Bucket = "test-bucket"
	cfg.Cloud.Region = "cn-hangzhou"
	return cfg
}

func TestNewStorage_LocalAndUnknown(t *testing.T) {
	cases := map[string]string{"": "", "explicit": "local", "uppercase": "LOCAL", "unknown": "ftp", "padded": "  local "}
	for name, storageType := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := config.FileConfig{StorageType: storageType, UploadPath: "./uploads", UploadURL: "http://localhost/uploads"}
			s, err := NewStorage(cfg, "sign-key")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, ok := s.(*LocalStorage); !ok {
				t.Fatalf("expected *LocalStorage, got %T", s)
			}
			if s.Type() != "local" {
				t.Fatalf("expected type local, got %q", s.Type())
			}
		})
	}
}

func TestNewStorage_OSS(t *testing.T) {
	t.Run("valid config returns oss", func(t *testing.T) {
		s, err := NewStorage(fullCloudCfg("oss"), "")
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := s.(*OSSStorage); !ok {
			t.Fatalf("expected *OSSStorage, got %T", s)
		}
		if s.Type() != "oss" {
			t.Fatalf("expected type oss, got %q", s.Type())
		}
	})
	t.Run("missing endpoint errors", func(t *testing.T) {
		cfg := fullCloudCfg("oss")
		cfg.Cloud.Endpoint = ""
		if _, err := NewStorage(cfg, ""); err == nil {
			t.Fatal("expected error for missing endpoint")
		}
	})
	t.Run("missing credentials errors", func(t *testing.T) {
		cfg := fullCloudCfg("oss")
		cfg.Cloud.SecretKey = ""
		if _, err := NewStorage(cfg, ""); err == nil {
			t.Fatal("expected error for missing secret key")
		}
	})
}

func TestNewStorage_S3(t *testing.T) {
	t.Run("valid config returns s3", func(t *testing.T) {
		s, err := NewStorage(fullCloudCfg("s3"), "")
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := s.(*S3Storage); !ok {
			t.Fatalf("expected *S3Storage, got %T", s)
		}
		if s.Type() != "s3" {
			t.Fatalf("expected type s3, got %q", s.Type())
		}
	})
	t.Run("missing bucket errors", func(t *testing.T) {
		cfg := fullCloudCfg("s3")
		cfg.Cloud.Bucket = ""
		if _, err := NewStorage(cfg, ""); err == nil {
			t.Fatal("expected error for missing bucket")
		}
	})
	t.Run("missing access key errors", func(t *testing.T) {
		cfg := fullCloudCfg("s3")
		cfg.Cloud.AccessKey = ""
		if _, err := NewStorage(cfg, ""); err == nil {
			t.Fatal("expected error for missing access key")
		}
	})
}
