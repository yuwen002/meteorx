package signedurl

import (
	"net/url"
	"testing"
	"time"
)

const testSecret = "test-secret-abcdefghijklmnopqrstuvwxyz-1234567890"

func TestSignVerifyValid(t *testing.T) {
	expires := Expires(DefaultTTL)
	path := "01JZYZabcdefghijklmnopqrst.pdf"
	sig := Sign(path, testSecret, expires)
	if !Verify(path, testSecret, expires, sig) {
		t.Fatal("valid signature should pass verification")
	}
}

func TestSignVerifyRejectsTampering(t *testing.T) {
	expires := Expires(DefaultTTL)
	path := "01JZYZabcdefghijklmnopqrst.pdf"
	sig := Sign(path, testSecret, expires)

	cases := []struct {
		name    string
		path    string
		secret  string
		expires int64
		sig     string
	}{
		{"wrong path", "other.pdf", testSecret, expires, sig},
		{"wrong secret", path, "another-secret", expires, sig},
		{"tampered expires", path, testSecret, expires + 1, sig},
		{"tampered sig", path, testSecret, expires, "0000000000000000000000000000000000000000000000000000000000000000"},
		{"empty sig", path, testSecret, expires, ""},
		{"zero expires", path, testSecret, 0, sig},
	}
	for _, c := range cases {
		if Verify(c.path, c.secret, c.expires, c.sig) {
			t.Errorf("%s: verification should fail", c.name)
		}
	}
}

func TestBuild(t *testing.T) {
	path := "demo.png"
	expires := time.Now().Add(time.Minute).Unix()
	raw := Build("http://localhost:8081/uploads/", path, testSecret, expires)

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("Build returned invalid URL: %v", err)
	}
	if u.Path != "/uploads/demo.png" {
		t.Errorf("unexpected path: %s", u.Path)
	}
	q := u.Query()
	if q.Get(QueryExpires) == "" || q.Get(QuerySig) == "" {
		t.Fatalf("missing signature query params: %s", raw)
	}
	if !Verify(path, testSecret, expires, q.Get(QuerySig)) {
		t.Fatal("query signature in Build URL should verify")
	}
}

// TestVerifyAnyRotation 验证密钥链校验：旧密钥签发的签名在密钥轮换期仍可通过链式校验，
// 但剔除旧密钥后失效；与当前密钥不匹配的历史签名同样被拒绝。
func TestVerifyAnyRotation(t *testing.T) {
	path := "rotation.txt"
	expires := Expires(time.Minute)
	oldKey := testSecret
	newKey := "newly-rotated-secret-abcdefghijklmnopqrstuvwxyz-1234567890"
	oldSig := Sign(path, oldKey, expires)
	newSig := Sign(path, newKey, expires)

	// 轮换宽限期：新旧密钥并存
	if !VerifyAny(path, []string{newKey, oldKey}, expires, oldSig) {
		t.Fatal("old signature should verify during rotation grace period")
	}
	if !VerifyAny(path, []string{newKey, oldKey}, expires, newSig) {
		t.Fatal("new signature should verify during rotation grace period")
	}

	// 剔除旧密钥后，旧签名失效
	if VerifyAny(path, []string{newKey}, expires, oldSig) {
		t.Fatal("old signature must fail after old key removed from chain")
	}

	// 边界：空链 / 空签名 / 非法过期时间一律拒绝
	if VerifyAny(path, nil, expires, oldSig) {
		t.Fatal("empty secret chain must reject")
	}
	if VerifyAny(path, []string{newKey}, expires, "") {
		t.Fatal("empty signature must reject")
	}
	if VerifyAny(path, []string{newKey}, 0, oldSig) {
		t.Fatal("zero expires must reject")
	}
	if VerifyAny(path, []string{""}, expires, oldSig) {
		t.Fatal("chain with only empty entries must reject")
	}
}
