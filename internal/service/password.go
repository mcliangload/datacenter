package service

import (
	"unicode"

	"datacenter/internal/errno"
)

// validatePasswordStrength 密码强度校验（安全增强 P0-3）：
// 至少 8 位，且同时包含字母与数字。创建用户 / admin 重置 / 个人改密三处统一使用。
func validatePasswordStrength(pw string) *errno.Error {
	if len(pw) < 8 {
		return errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "密码长度至少 8 位")
	}
	hasLetter, hasDigit := false, false
	for _, r := range pw {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errno.New(errno.ErrParam.Code, errno.ErrParam.HTTPStatus, "密码需同时包含字母和数字")
	}
	return nil
}
