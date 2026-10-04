package bootstrap

import (
	"testing"
	"time"
)

// TestApplyTimezone 验证按名称强制全局 time.Local：有效名称生效、空/无效名称保持原值不变。
func TestApplyTimezone(t *testing.T) {
	orig := time.Local
	t.Cleanup(func() { time.Local = orig })

	// 有效时区应生效
	applyTimezone("Asia/Shanghai")
	if time.Local.String() != "Asia/Shanghai" {
		t.Fatalf("expected Asia/Shanghai, got %s", time.Local.String())
	}

	// 空名称不改动
	applyTimezone("")
	if time.Local.String() != "Asia/Shanghai" {
		t.Fatalf("empty name should keep timezone, got %s", time.Local.String())
	}

	// 无效名称保持原值不变，不影响进程
	applyTimezone("Not/AZone")
	if time.Local.String() != "Asia/Shanghai" {
		t.Fatalf("invalid name should keep previous timezone, got %s", time.Local.String())
	}
}
