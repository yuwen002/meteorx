package security

import (
	"strings"
	"testing"
)

func TestGenerateRecoveryCodes_Format(t *testing.T) {
	codes, err := GenerateRecoveryCodes(RecoveryCodeCount)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes: %v", err)
	}
	if len(codes) != RecoveryCodeCount {
		t.Fatalf("want %d codes, got %d", RecoveryCodeCount, len(codes))
	}
	for _, c := range codes {
		groups := strings.Split(c, "-")
		if len(groups) != recoveryCodeGroups {
			t.Fatalf("code %q: want %d groups", c, recoveryCodeGroups)
		}
		for _, g := range groups {
			if len(g) != recoveryCodeGroupLen {
				t.Fatalf("code %q: group %q wrong length", c, g)
			}
			// 去歧义字符集内且为大写
			for _, r := range g {
				if !strings.ContainsRune(recoveryAlphabet, r) {
					t.Fatalf("code %q: rune %q not in alphabet", c, r)
				}
			}
		}
	}
}

func TestGenerateRecoveryCodes_Unique(t *testing.T) {
	seen := map[string]bool{}
	codes, err := GenerateRecoveryCodes(200)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes: %v", err)
	}
	for _, c := range codes {
		if seen[c] {
			t.Fatalf("duplicate recovery code generated: %q", c)
		}
		seen[c] = true
	}
}

func TestGenerateRecoveryCodes_InvalidCount(t *testing.T) {
	if _, err := GenerateRecoveryCodes(0); err == nil {
		t.Fatal("want error for non-positive count")
	}
}

func TestNormalizeRecoveryCode(t *testing.T) {
	got := NormalizeRecoveryCode(" abcde-fghij \n")
	if want := "ABCDEFGHIJ"; got != want {
		t.Fatalf("normalize: want %q got %q", want, got)
	}
}

func TestHashAndVerifyRecoveryCode(t *testing.T) {
	codes, err := GenerateRecoveryCodes(1)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes: %v", err)
	}
	plain := codes[0]
	hash := HashRecoveryCode(plain)
	if hash == plain {
		t.Fatal("hash must not equal plaintext")
	}
	if !VerifyRecoveryCode(plain, hash) {
		t.Fatal("verify should pass for matching code")
	}
	// 大小写/空格/横线差异不影响比对
	if !VerifyRecoveryCode(strings.ToLower(strings.ReplaceAll(plain, "-", " ")), hash) {
		t.Fatal("verify should tolerate format differences")
	}
	if VerifyRecoveryCode("WRONG-CODEX", hash) {
		t.Fatal("verify should fail for wrong code")
	}
}
