package service

import "testing"

func TestValidatePasswordStrength(t *testing.T) {
	cases := []struct {
		name string
		pw   string
		want bool
	}{
		{"合法：8位字母数字", "abc12345", true},
		{"合法：长密码含符号", "Passw0rd!@#", true},
		{"合法：中文+数字（unicode 字母）", "密码abc12345", true},
		{"过短：6位", "abc123", false},
		{"过短：7位", "abc1234", false},
		{"纯字母", "abcdefgh", false},
		{"纯数字", "12345678", false},
		{"空密码", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validatePasswordStrength(c.pw)
			if (err == nil) != c.want {
				t.Fatalf("validatePasswordStrength(%q) 错误=%v, 期望成功=%v", c.pw, err, c.want)
			}
		})
	}
}
