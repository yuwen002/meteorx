package model

import "testing"

// TestDirectKey_Deterministic 验证 1:1 私聊去重键与参数顺序无关，(a,b) 与 (b,a) 生成同一 key。
func TestDirectKey_Deterministic(t *testing.T) {
	k1 := DirectKey("userB", "userA")
	k2 := DirectKey("userA", "userB")
	if k1 != k2 {
		t.Fatalf("DirectKey 应对参数顺序不敏感: %q != %q", k1, k2)
	}
	// 规范化后应为 "direct:" + 较小ID + ":" + 较大ID
	want := "direct:userA:userB"
	if k1 != want {
		t.Fatalf("DirectKey 规范化结果错误: got %q want %q", k1, want)
	}
}

// TestDirectKey_DifferentPairs 不同用户对应不同 key。
func TestDirectKey_DifferentPairs(t *testing.T) {
	if DirectKey("u1", "u2") == DirectKey("u1", "u3") {
		t.Fatal("不同参与者应生成不同的会话 key")
	}
}
