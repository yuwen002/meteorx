package security

import (
	"strings"
	"testing"
	"unicode"

	"meteorx/internal/config"
)

// defaultPolicy 与 config.yaml 中的默认密码策略保持一致，用于验证生成的密码必然合规。
var defaultPolicy = config.PasswordPolicyConfig{
	Enabled:          true,
	MinLength:        8,
	MaxLength:        32,
	RequireUppercase: true,
	RequireLowercase: true,
	RequireDigit:     true,
	RequireSpecial:   true,
}

func TestGeneratePassword_Length(t *testing.T) {
	for _, n := range []int{4, 8, 12, 32} {
		pwd, err := GeneratePassword(n)
		if err != nil {
			t.Fatalf("GeneratePassword(%d) 返回错误: %v", n, err)
		}
		if len(pwd) != n {
			t.Errorf("GeneratePassword(%d) 长度 = %d", n, len(pwd))
		}
	}
}

func TestGeneratePassword_RejectsInvalidLength(t *testing.T) {
	for _, n := range []int{-1, 0, 3, 129} {
		if _, err := GeneratePassword(n); err == nil {
			t.Errorf("GeneratePassword(%d) 期望返回错误，实际为 nil", n)
		}
	}
}

func TestGeneratePassword_SatisfiesDefaultPolicy(t *testing.T) {
	for i := 0; i < 500; i++ {
		pwd, err := GeneratePassword(12)
		if err != nil {
			t.Fatalf("生成密码失败: %v", err)
		}
		if err := ValidatePassword(pwd, defaultPolicy); err != nil {
			t.Fatalf("生成的密码 %q 不满足默认策略: %v", pwd, err)
		}
	}
}

// TestGeneratePassword_HasAllCharClasses 校验四类字符齐全，且字符必须落在刻意剔除易混淆字符后的集合内。
func TestGeneratePassword_HasAllCharClasses(t *testing.T) {
	allowed := map[rune]bool{}
	for _, r := range pwdAll {
		allowed[r] = true
	}

	for i := 0; i < 200; i++ {
		pwd, err := GeneratePassword(16)
		if err != nil {
			t.Fatalf("生成密码失败: %v", err)
		}
		var hasUpper, hasLower, hasDigit, hasSpecial bool
		for _, r := range pwd {
			if !allowed[r] {
				t.Fatalf("密码 %q 包含字符集之外的字符 %q", pwd, r)
			}
			switch {
			case unicode.IsUpper(r):
				hasUpper = true
			case unicode.IsLower(r):
				hasLower = true
			case unicode.IsDigit(r):
				hasDigit = true
			default:
				hasSpecial = true
			}
		}
		if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
			t.Fatalf("密码 %q 未包含全部四类字符 (upper=%v lower=%v digit=%v special=%v)",
				pwd, hasUpper, hasLower, hasDigit, hasSpecial)
		}
	}
}

// TestGeneratePassword_NoEasyToConfuseChars 防止有人把 l/I/1/O/0 加回字符集（复制粘贴易错）。
func TestGeneratePassword_NoEasyToConfuseChars(t *testing.T) {
	for i := 0; i < 300; i++ {
		pwd, err := GeneratePassword(24)
		if err != nil {
			t.Fatalf("生成密码失败: %v", err)
		}
		if strings.ContainsAny(pwd, "lI1O0") {
			t.Fatalf("密码 %q 含有易混淆字符", pwd)
		}
	}
}

// TestGeneratePassword_Unique 采样 1000 次不应出现重复（crypto/rand 源）。
func TestGeneratePassword_Unique(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		pwd, err := GeneratePassword(12)
		if err != nil {
			t.Fatalf("生成密码失败: %v", err)
		}
		if _, ok := seen[pwd]; ok {
			t.Fatalf("出现重复密码: %q", pwd)
		}
		seen[pwd] = struct{}{}
	}
}
