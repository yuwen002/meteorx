package security

import (
	"errors"
	"regexp"
	"unicode"

	"meteorx/internal/config"
)

// ValidatePassword 验证密码是否符合安全策略
func ValidatePassword(password string, policy config.PasswordPolicyConfig) error {
	if !policy.Enabled {
		return nil
	}

	if len(password) < policy.MinLength {
		return errors.New("密码长度不能少于 " + string(rune('0'+policy.MinLength)) + " 位")
	}

	if policy.MaxLength > 0 && len(password) > policy.MaxLength {
		return errors.New("密码长度不能超过 " + string(rune('0'+policy.MaxLength)) + " 位")
	}

	if policy.RequireUppercase {
		hasUpper := false
		for _, r := range password {
			if unicode.IsUpper(r) {
				hasUpper = true
				break
			}
		}
		if !hasUpper {
			return errors.New("密码必须包含至少一个大写字母")
		}
	}

	if policy.RequireLowercase {
		hasLower := false
		for _, r := range password {
			if unicode.IsLower(r) {
				hasLower = true
				break
			}
		}
		if !hasLower {
			return errors.New("密码必须包含至少一个小写字母")
		}
	}

	if policy.RequireDigit {
		hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
		if !hasDigit {
			return errors.New("密码必须包含至少一个数字")
		}
	}

	if policy.RequireSpecial {
		hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)
		if !hasSpecial {
			return errors.New("密码必须包含至少一个特殊字符")
		}
	}

	return nil
}