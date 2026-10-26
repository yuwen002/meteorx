package security

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

// 随机密码字符集：刻意剔除易混淆字符（l/I/1、O/0）与 CSV/命令行不友好的引号、反斜杠，
// 保证生成的密码既能通过默认密码策略（需要特殊字符），又便于管理员复制粘贴下发。
const (
	pwdLower   = "abcdefghijkmnopqrstuvwxyz"
	pwdUpper   = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	pwdDigit   = "23456789"
	pwdSpecial = "!@#$%^&*_-+="
)

var pwdAll = pwdLower + pwdUpper + pwdDigit + pwdSpecial

// GeneratePassword 生成指定长度的随机密码，并保证同时包含大写字母、小写字母、数字与特殊字符。
// 生成源为 crypto/rand，取字符时对越界值做拒绝采样，避免取模偏差。
func GeneratePassword(length int) (string, error) {
	if length < 4 {
		return "", errors.New("密码长度不能小于 4")
	}
	if length > 128 {
		return "", errors.New("密码长度不能超过 128")
	}

	// 1. 先各取一个字符，确保四类字符齐全
	chars := make([]byte, 0, length)
	for _, set := range []string{pwdLower, pwdUpper, pwdDigit, pwdSpecial} {
		c, err := randomChar(set)
		if err != nil {
			return "", err
		}
		chars = append(chars, c)
	}

	// 2. 其余位置从全字符集取
	for len(chars) < length {
		c, err := randomChar(pwdAll)
		if err != nil {
			return "", err
		}
		chars = append(chars, c)
	}

	// 3. 打乱顺序，避免"前四位固定为各类字符"的可预测结构
	if err := shuffle(chars); err != nil {
		return "", err
	}
	return string(chars), nil
}

// randomChar 从字符集中等概率取一个字符。
func randomChar(set string) (byte, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
	if err != nil {
		return 0, fmt.Errorf("生成随机密码失败: %w", err)
	}
	return set[n.Int64()], nil
}

// shuffle 使用 crypto/rand 做 Fisher-Yates 洗牌。
func shuffle(chars []byte) error {
	for i := len(chars) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return fmt.Errorf("生成随机密码失败: %w", err)
		}
		j := int(n.Int64())
		chars[i], chars[j] = chars[j], chars[i]
	}
	return nil
}
