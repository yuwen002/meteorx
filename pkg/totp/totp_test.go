package totp

import (
	"testing"
	"time"
)

// RFC 6238 Appendix B 测试向量（HMAC-SHA1，密钥为 ASCII "12345678901234567890"，
// Base32 = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"）。原始为 8 位，此处取后 6 位对齐 Digits=6。
const testSecret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

func TestAt_RFC6238Vectors(t *testing.T) {
	cases := []struct {
		unix int64
		want string
	}{
		{59, "287082"},          // 94287082
		{1111111109, "081804"},  // 07081804
		{1111111111, "050471"},  // 14050471
		{1234567890, "005924"},  // 89005924
		{2000000000, "279037"},  // 69279037
		{20000000000, "353130"}, // 65353130（T=200 亿，非 20 亿）
	}
	for _, c := range cases {
		got, err := At(testSecret, time.Unix(c.unix, 0).UTC())
		if err != nil {
			t.Fatalf("At(%d) error: %v", c.unix, err)
		}
		if got != c.want {
			t.Errorf("At(%d) = %q, want %q", c.unix, got, c.want)
		}
	}
}

func TestValidate_RoundTripAndSkew(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	code, err := At(testSecret, now)
	if err != nil {
		t.Fatalf("At error: %v", err)
	}

	// 当前时刻命中
	ok, err := ValidateAt(testSecret, code, now)
	if err != nil || !ok {
		t.Fatalf("expected current code to validate, ok=%v err=%v", ok, err)
	}

	// 前一个时间步的代码在 skew=1 内仍应命中
	prev, err := At(testSecret, now.Add(-time.Duration(Period)*time.Second))
	if err != nil {
		t.Fatalf("At prev error: %v", err)
	}
	ok, err = ValidateAt(testSecret, prev, now)
	if err != nil || !ok {
		t.Fatalf("expected within-skew code to validate, ok=%v err=%v", ok, err)
	}

	// 超出 skew（前 3 步）应不命中
	old, err := At(testSecret, now.Add(-3 * time.Duration(Period) * time.Second))
	if err != nil {
		t.Fatalf("At old error: %v", err)
	}
	ok, err = ValidateAt(testSecret, old, now)
	if err != nil {
		t.Fatalf("validate old error: %v", err)
	}
	if ok {
		t.Errorf("expected out-of-skew code to be rejected")
	}
}

func TestValidate_EmptySecretIsNoOp(t *testing.T) {
	// 未启用 2FA（空密钥）：不报错，返回 false，由调用方决定放行
	ok, err := ValidateAt("", "123456", time.Now())
	if err != nil {
		t.Fatalf("empty secret should not error, got %v", err)
	}
	if ok {
		t.Errorf("empty secret should never match")
	}
}

func TestValidate_InvalidCode(t *testing.T) {
	// 位数不对 / 含非数字 → ErrInvalidCode
	if _, err := ValidateAt(testSecret, "12345", time.Now()); err != ErrInvalidCode {
		t.Errorf("short code: want ErrInvalidCode, got %v", err)
	}
	if _, err := ValidateAt(testSecret, "abcdef", time.Now()); err != ErrInvalidCode {
		t.Errorf("non-digit code: want ErrInvalidCode, got %v", err)
	}
}

func TestGenerateSecret_UniqueAndValid(t *testing.T) {
	s1, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}
	s2, _ := GenerateSecret()
	if s1 == s2 {
		t.Errorf("two generated secrets should differ")
	}
	// 生成的密钥应可解码并生成有效口令
	if _, err := At(s1, time.Now()); err != nil {
		t.Errorf("generated secret should be usable: %v", err)
	}
}

func TestDecodeSecret_FormatTolerance(t *testing.T) {
	// 大小写、空格、连字符、缺失填充都应被容忍并解析出相同结果
	base, err := decodeSecret(testSecret)
	if err != nil {
		t.Fatalf("decode base error: %v", err)
	}
	messy := "gezdgnbvgY3TQOJQ-GEZDGNBVGY3TQOJQ "
	got, err := decodeSecret(messy)
	if err != nil {
		t.Fatalf("decode messy error: %v", err)
	}
	if string(base) != string(got) {
		t.Errorf("tolerant decode should equal canonical decode")
	}
}

func TestOTPAuthURL(t *testing.T) {
	u := OTPAuthURL(testSecret, "MeteorX", "alice@example.com")
	if u == "" {
		t.Fatal("otpauth url should not be empty")
	}
	for _, frag := range []string{"otpauth://totp/", "secret=" + testSecret, "issuer=MeteorX", "digits=6", "period=30"} {
		if !contains(u, frag) {
			t.Errorf("otpauth url %q missing fragment %q", u, frag)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
