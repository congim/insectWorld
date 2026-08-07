// Package token 访问令牌值对象与签发/校验能力接口。
package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewAccessToken 测试创建访问令牌。
func TestNewAccessToken(t *testing.T) {
	t.Run("正常创建", func(t *testing.T) {
		token := NewAccessToken(1001, 1700000000000, 1700000000000+3600000, 1)
		assert.Equal(t, int64(1001), token.PlayerID())
		assert.Equal(t, int64(1700000000000), token.IssueTime())
		assert.Equal(t, int64(1700000000000+3600000), token.ExpireTime())
		assert.Equal(t, 1, token.Version())
		assert.Equal(t, "", token.Signature())
	})
}

// TestAccessToken_IsExpired 测试令牌过期判定。
func TestAccessToken_IsExpired(t *testing.T) {
	token := NewAccessToken(1001, 1700000000000, 1700000000000+3600000, 1)
	t.Run("未过期", func(t *testing.T) {
		assert.False(t, token.IsExpired(1700000000000+1800000))
	})
	t.Run("已过期", func(t *testing.T) {
		assert.True(t, token.IsExpired(1700000000000+7200000))
	})
	t.Run("临界过期", func(t *testing.T) {
		assert.True(t, token.IsExpired(1700000000000+3600000))
	})
}

// TestAccessToken_SetSignature 测试设置签名。
func TestAccessToken_SetSignature(t *testing.T) {
	token := NewAccessToken(1001, 1700000000000, 1700000000000+3600000, 1)
	token.SetSignature("abc123")
	assert.Equal(t, "abc123", token.Signature())
}

// TestAccessToken_ToPayload 测试获取负载拷贝。
func TestAccessToken_ToPayload(t *testing.T) {
	token := NewAccessToken(1001, 1700000000000, 1700000000000+3600000, 1)
	payload := token.ToPayload()
	assert.Equal(t, int64(1001), payload.PlayerID)
	assert.Equal(t, int64(1700000000000), payload.IssueTime)
	assert.Equal(t, int64(1700000000000+3600000), payload.ExpireTime)
	assert.Equal(t, 1, payload.Version)
}
