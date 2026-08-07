// Package account 玩家账号聚合根与凭证值对象，维护账号档案的一致性边界。
package account

import (
	"testing"

	"github.com/stretchr/testify/assert"

	gatewayerr "insectworld/server/gateway/domain/errors"
)

// TestValidateUsernameFormat 测试用户名格式校验，覆盖spec 5.1.1 规则1全部验收条件。
func TestValidateUsernameFormat(t *testing.T) {
	tests := []struct {
		name     string
		username string
		minLen   int
		maxLen   int
		wantErr  error
	}{
		{"合法用户名", "testuser", 4, 20, nil},
		{"含下划线", "test_user", 4, 20, nil},
		{"含数字", "user123", 4, 20, nil},
		{"最短长度", "abcd", 4, 20, nil},
		{"最长长度", "abcdefghijklmnopqrst", 4, 20, nil},
		{"过短", "abc", 4, 20, gatewayerr.ErrInvalidUsernameFormat},
		{"过长", "abcdefghijklmnopqrstu", 4, 20, gatewayerr.ErrInvalidUsernameFormat},
		{"纯数字", "12345678", 4, 20, gatewayerr.ErrInvalidUsernameFormat},
		{"含空格", "test user", 4, 20, gatewayerr.ErrInvalidUsernameFormat},
		{"含特殊符号", "test@user", 4, 20, gatewayerr.ErrInvalidUsernameFormat},
		{"含中文", "测试用户", 4, 20, gatewayerr.ErrInvalidUsernameFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsernameFormat(tt.username, tt.minLen, tt.maxLen)
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidatePasswordStrength 测试密码强度校验，覆盖spec 5.1.1 规则2全部验收条件。
func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		minLen   int
		maxLen   int
		wantErr  error
	}{
		{"合法密码", "password123", 8, 32, nil},
		{"最短长度", "pass1234", 8, 32, nil},
		{"最长长度", "abcdefghijklmnopqrstuvwxyz123456", 8, 32, nil},
		{"过短", "pass123", 8, 32, gatewayerr.ErrInvalidPasswordStrength},
		{"仅字母", "password", 8, 32, gatewayerr.ErrInvalidPasswordStrength},
		{"仅数字", "12345678", 8, 32, gatewayerr.ErrInvalidPasswordStrength},
		{"过长", "abcdefghijklmnopqrstuvwxyz1234567", 8, 32, gatewayerr.ErrInvalidPasswordStrength},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password, tt.minLen, tt.maxLen)
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
