// Package errors Gateway服务错误码定义。
package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGatewayError_Error 测试GatewayError实现error接口的字符串格式。
func TestGatewayError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *GatewayError
		want string
	}{
		{"Token无效", ErrTokenInvalid, "[17001] Token无效"},
		{"用户名格式不合法", ErrInvalidUsernameFormat, "[17010] 用户名格式不合法"},
		{"账号不存在", ErrAccountNotFound, "[17020] 账号不存在"},
		{"会话不存在", ErrSessionNotFound, "[17052] 会话不存在"},
		{"密码哈希失败", ErrPasswordHashFailed, "[17059] 密码哈希失败"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

// TestErrorCodeUniqueness 测试全部错误码取值唯一，防止重复定义导致语义歧义。
func TestErrorCodeUniqueness(t *testing.T) {
	codes := []int{
		ErrCodeTokenInvalid, ErrCodeRateLimited, ErrCodeConnectionClosed,
		ErrCodeInvalidUsernameFormat, ErrCodeInvalidPasswordStrength, ErrCodeUsernameAlreadyExists,
		ErrCodeRegisterRateLimited, ErrCodeRegisterInternalError,
		ErrCodeAccountNotFound, ErrCodePasswordIncorrect, ErrCodeAccountBanned, ErrCodeAccountLocked,
		ErrCodeLoginRateLimited, ErrCodeLoginInternalError,
		ErrCodeLogoutInternalError,
		ErrCodeTokenMissing, ErrCodeTokenExpired, ErrCodeNotLoggedIn,
		ErrCodeAccountRepoUnavailable, ErrCodeSessionRepoUnavailable, ErrCodeSessionNotFound,
		ErrCodeAccountNotFoundSentinel, ErrCodeTokenSignerUnavailable, ErrCodeTokenBlacklistUnavailable,
		ErrCodeFailureTrackerUnavailable, ErrCodeAuditLogUnavailable, ErrCodeIDGenClockBack,
		ErrCodePasswordHashFailed,
	}
	seen := make(map[int]bool, len(codes))
	for _, c := range codes {
		assert.False(t, seen[c], "错误码重复定义: %d", c)
		seen[c] = true
	}
}

// TestErrorSentinelComparison 测试哨兵错误可通过errors.Is比较，验证GatewayError指针语义。
func TestErrorSentinelComparison(t *testing.T) {
	// 同一哨兵变量比较
	assert.True(t, errors.Is(ErrAccountNotFoundSentinel, ErrAccountNotFoundSentinel))
	assert.True(t, errors.Is(ErrSessionNotFound, ErrSessionNotFound))

	// 包裹后的错误仍可通过errors.Is识别
	wrapped := wrapErr(ErrAccountNotFoundSentinel)
	assert.True(t, errors.Is(wrapped, ErrAccountNotFoundSentinel))

	// 不同哨兵不应匹配
	assert.False(t, errors.Is(ErrAccountNotFound, ErrAccountNotFoundSentinel))
}

// wrapErr 模拟错误包裹辅助函数。
func wrapErr(err error) error {
	return &wrappedError{inner: err}
}

// wrappedError 测试用包裹错误类型。
type wrappedError struct{ inner error }

func (w *wrappedError) Error() string { return "wrapped: " + w.inner.Error() }
func (w *wrappedError) Unwrap() error { return w.inner }

// TestErrorCodeRanges 测试错误码落在Gateway服务保留区间17000-17999。
func TestErrorCodeRanges(t *testing.T) {
	codes := []int{
		ErrCodeTokenInvalid, ErrCodeRateLimited, ErrCodeConnectionClosed,
		ErrCodeInvalidUsernameFormat, ErrCodeInvalidPasswordStrength, ErrCodeUsernameAlreadyExists,
		ErrCodeRegisterRateLimited, ErrCodeRegisterInternalError,
		ErrCodeAccountNotFound, ErrCodePasswordIncorrect, ErrCodeAccountBanned, ErrCodeAccountLocked,
		ErrCodeLoginRateLimited, ErrCodeLoginInternalError,
		ErrCodeLogoutInternalError,
		ErrCodeTokenMissing, ErrCodeTokenExpired, ErrCodeNotLoggedIn,
		ErrCodeAccountRepoUnavailable, ErrCodeSessionRepoUnavailable, ErrCodeSessionNotFound,
		ErrCodeAccountNotFoundSentinel, ErrCodeTokenSignerUnavailable, ErrCodeTokenBlacklistUnavailable,
		ErrCodeFailureTrackerUnavailable, ErrCodeAuditLogUnavailable, ErrCodeIDGenClockBack,
		ErrCodePasswordHashFailed,
	}
	for _, c := range codes {
		assert.GreaterOrEqual(t, c, 17000, "错误码低于Gateway区间下限: %d", c)
		assert.LessOrEqual(t, c, 17999, "错误码高于Gateway区间上限: %d", c)
	}
}
