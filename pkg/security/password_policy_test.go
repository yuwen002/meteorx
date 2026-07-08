package security

import (
	"testing"

	"meteorx/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestValidatePassword(t *testing.T) {
	policy := config.PasswordPolicyConfig{
		Enabled:          true,
		MinLength:        8,
		MaxLength:        32,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:     true,
		RequireSpecial:   true,
	}

	tests := []struct {
		name     string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid password",
			password: "Hello1!world",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "He1!",
			wantErr:  true,
			errMsg:   "密码长度不能少于",
		},
		{
			name:     "too long",
			password: "Hello1!worldHello1!worldHello1!world",
			wantErr:  true,
			errMsg:   "密码长度不能超过",
		},
		{
			name:     "no uppercase",
			password: "hello1!world",
			wantErr:  true,
			errMsg:   "大写字母",
		},
		{
			name:     "no lowercase",
			password: "HELLO1!WORLD",
			wantErr:  true,
			errMsg:   "小写字母",
		},
		{
			name:     "no digit",
			password: "Hello!world",
			wantErr:  true,
			errMsg:   "数字",
		},
		{
			name:     "no special char",
			password: "Hello1world",
			wantErr:  true,
			errMsg:   "特殊字符",
		},
		{
			name:     "exactly min length",
			password: "He1!wo2@",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, policy)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePasswordDisabled(t *testing.T) {
	policy := config.PasswordPolicyConfig{
		Enabled: false,
	}

	// 即使密码不符合任何规则，策略禁用时也应通过
	err := ValidatePassword("123", policy)
	assert.NoError(t, err)
}

func TestValidatePasswordWithoutSpecial(t *testing.T) {
	policy := config.PasswordPolicyConfig{
		Enabled:          true,
		MinLength:        8,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:     true,
		RequireSpecial:   false,
	}

	err := ValidatePassword("Hello1world", policy)
	assert.NoError(t, err)
}

func TestValidatePasswordOnlyLength(t *testing.T) {
	policy := config.PasswordPolicyConfig{
		Enabled:   true,
		MinLength: 6,
		MaxLength: 20,
	}

	err := ValidatePassword("abcdef", policy)
	assert.NoError(t, err)
}

func BenchmarkValidatePassword(b *testing.B) {
	policy := config.PasswordPolicyConfig{
		Enabled:          true,
		MinLength:        8,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:     true,
		RequireSpecial:   true,
	}

	password := "Hello1!world"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidatePassword(password, policy)
	}
}