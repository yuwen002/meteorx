package storage

import (
	"testing"

	"meteorx/internal/config"
)

// newLocalDefault 构造一个本地后端作为注册表的默认后端，供各用例复用。
func newLocalDefault() Storage {
	return NewLocalStorage("./uploads", "http://localhost/uploads", "sign-key")
}

func TestRegistry_DefaultFastPath(t *testing.T) {
	cfg := fullCloudCfg("local")
	r := NewRegistry(newLocalDefault(), cfg, "sign-key")

	def := r.Default()
	if _, ok := def.(*LocalStorage); !ok {
		t.Fatalf("Default should be the local backend, got %T", def)
	}

	// 空类型与「等于默认类型」都应直接命中默认后端（无锁快路径）。
	for _, typ := range []string{"", "local"} {
		if got := r.Resolve(typ); got != def {
			t.Fatalf("Resolve(%q) should return the default backend identically", typ)
		}
	}
}

func TestRegistry_LazyBuildAndCache(t *testing.T) {
	// fullCloudCfg 的 Cloud 各字段齐备，OSS 与 S3 均可离线构造（构造不发网络请求）。
	cfg := fullCloudCfg("local")
	r := NewRegistry(newLocalDefault(), cfg, "sign-key")

	oss1 := r.Resolve("oss")
	if _, ok := oss1.(*OSSStorage); !ok {
		t.Fatalf("Resolve(oss) should build *OSSStorage, got %T", oss1)
	}
	if oss1 == r.Default() {
		t.Fatal("Resolve(oss) should not fall back to default when config is valid")
	}
	// 第二次解析应命中缓存，返回同一实例。
	if oss2 := r.Resolve("oss"); oss2 != oss1 {
		t.Fatal("Resolve(oss) should return the cached backend instance on second call")
	}

	s3 := r.Resolve("s3")
	if _, ok := s3.(*S3Storage); !ok {
		t.Fatalf("Resolve(s3) should build *S3Storage, got %T", s3)
	}
}

func TestRegistry_BuildFailureFallsBackToDefault(t *testing.T) {
	// 缺少 endpoint/凭证时，oss 构造必然失败：应回退默认后端，并记忆化失败避免重复构造。
	cfg := config.FileConfig{StorageType: "local", UploadPath: "./uploads", UploadURL: "http://localhost/uploads"}
	r := NewRegistry(newLocalDefault(), cfg, "sign-key")

	def := r.Default()
	if got := r.Resolve("oss"); got != def {
		t.Fatalf("Resolve(oss) with invalid config should fall back to default, got %T", got)
	}
	// 失败已被记忆化，再次解析仍稳定返回默认后端。
	if got := r.Resolve("oss"); got != def {
		t.Fatal("Resolve(oss) should keep returning default after failure memoization")
	}
}

func TestRegistry_UnknownTypeDegradesToLocal(t *testing.T) {
	// 未知类型不会报错，buildStorage 的 default 分支降级为本地实例并被缓存。
	cfg := fullCloudCfg("local")
	r := NewRegistry(newLocalDefault(), cfg, "sign-key")

	got := r.Resolve("ftp")
	if _, ok := got.(*LocalStorage); !ok {
		t.Fatalf("Resolve(unknown) should degrade to a local backend, got %T", got)
	}
	if got == r.Default() {
		t.Fatal("degraded local backend should be a separately constructed instance, not the default pointer")
	}
	// 降级结果同样被缓存，二次解析命中同一实例。
	if again := r.Resolve("ftp"); again != got {
		t.Fatal("Resolve(unknown) should cache the degraded backend")
	}
}
