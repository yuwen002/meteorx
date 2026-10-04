package config

import (
	"testing"
	"time"
)

func TestUploadSignKeys_FallbackToJWTSecret(t *testing.T) {
	// 未配置 file.sign_key：兼容旧部署，回退 jwt.secret
	f := FileConfig{}
	keys := f.UploadSignKeys("jwt-secret-abcdefghijklmnopqrstuvwxyz")
	if len(keys) != 1 || keys[0] != "jwt-secret-abcdefghijklmnopqrstuvwxyz" {
		t.Fatalf("expected fallback to jwt secret, got %v", keys)
	}

	// jwt secret 也为空 → 无密钥（公开访问模式）
	if keys := f.UploadSignKeys(""); keys != nil {
		t.Fatalf("expected nil keys when jwt secret empty, got %v", keys)
	}
}

func TestUploadSignKeys_RotationChain(t *testing.T) {
	f := FileConfig{SignKey: " new-key , old-key , "}
	keys := f.UploadSignKeys("jwt-unused")
	want := []string{"new-key", "old-key"}
	if len(keys) != len(want) {
		t.Fatalf("expected %v, got %v", want, keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, keys)
		}
	}
	// 空项被剔除；首项为当前签发密钥
	if got := f.UploadSignKey("jwt-unused"); got != "new-key" {
		t.Fatalf("expected signing key to be first entry, got %q", got)
	}
}

func TestUploadSignKeys_GarbageOnlyYieldsNoKeys(t *testing.T) {
	f := FileConfig{SignKey: ",,,"}
	if keys := f.UploadSignKeys("jwt-secret"); keys != nil {
		t.Fatalf("sign_key containing only separators should yield no keys, got %v", keys)
	}
}

func TestTaskReminderConfig_Defaults(t *testing.T) {
	c := TaskReminderConfig{}
	if c.GetHorizon() != 24*time.Hour {
		t.Fatalf("expected default horizon 24h, got %v", c.GetHorizon())
	}
	if c.GetInterval() != 30*time.Minute {
		t.Fatalf("expected default interval 30m, got %v", c.GetInterval())
	}
	if c.GetOverdueCooldown() != 24*time.Hour {
		t.Fatalf("expected default overdue cooldown 24h, got %v", c.GetOverdueCooldown())
	}
	if c.GetBatchLimit() != 200 {
		t.Fatalf("expected default batch limit 200, got %d", c.GetBatchLimit())
	}
}

func TestTaskReminderConfig_Overrides(t *testing.T) {
	c := TaskReminderConfig{
		Horizon:         6 * time.Hour,
		Interval:        15 * time.Minute,
		OverdueCooldown: 12 * time.Hour,
		BatchLimit:      50,
	}
	if c.GetHorizon() != 6*time.Hour || c.GetInterval() != 15*time.Minute ||
		c.GetOverdueCooldown() != 12*time.Hour || c.GetBatchLimit() != 50 {
		t.Fatalf("configured values should be returned as-is, got %+v", c)
	}
}
