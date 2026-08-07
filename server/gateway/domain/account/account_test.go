// Package account 玩家账号聚合根与凭证值对象，维护账号档案的一致性边界。
package account

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gatewayerr "insectworld/server/gateway/domain/errors"
)

// mockHasher 测试用密码哈希器mock。
type mockHasher struct {
	verifyResult bool
	verifyErr    error
}

func (m *mockHasher) Hash(ctx context.Context, password string) (string, string, error) {
	return "hashed_" + password, "salt", nil
}

func (m *mockHasher) Verify(ctx context.Context, password string, hash string, salt string) (bool, error) {
	return m.verifyResult, m.verifyErr
}

// TestNewPlayerAccount 测试创建玩家账号聚合根。
func TestNewPlayerAccount(t *testing.T) {
	t.Run("正常创建", func(t *testing.T) {
		account := NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)
		assert.Equal(t, int64(1001), account.PlayerID())
		assert.Equal(t, "testuser", account.Username())
		assert.Equal(t, "hash", account.PasswordHash())
		assert.Equal(t, "salt", account.Salt())
		assert.Equal(t, AccountStatusNormal, account.Status())
		assert.Equal(t, "", account.BanReason())
		assert.Equal(t, int64(0), account.BanExpireTime())
		assert.Equal(t, int64(1700000000000), account.RegisterTime())
		assert.Equal(t, "127.0.0.1", account.RegisterIP())
	})
}

// TestPlayerAccount_Ban 测试封禁与解封状态机流转。
func TestPlayerAccount_Ban(t *testing.T) {
	t.Run("封禁账号", func(t *testing.T) {
		account := NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)
		err := account.Ban("违规行为", 1700000000000+3600000)
		require.NoError(t, err)
		assert.Equal(t, AccountStatusBanned, account.Status())
		assert.Equal(t, "违规行为", account.BanReason())
		assert.Equal(t, int64(1700000000000+3600000), account.BanExpireTime())
	})

	t.Run("永久封禁", func(t *testing.T) {
		account := NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)
		err := account.Ban("严重违规", 0)
		require.NoError(t, err)
		assert.True(t, account.IsBanned(1700000000000+9999999999))
	})

	t.Run("解封账号", func(t *testing.T) {
		account := NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)
		_ = account.Ban("违规", 0)
		err := account.Unban()
		require.NoError(t, err)
		assert.Equal(t, AccountStatusNormal, account.Status())
		assert.Equal(t, "", account.BanReason())
		assert.Equal(t, int64(0), account.BanExpireTime())
	})
}

// TestPlayerAccount_IsBanned 测试封禁状态判定。
func TestPlayerAccount_IsBanned(t *testing.T) {
	t.Run("正常账号未封禁", func(t *testing.T) {
		account := NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)
		assert.False(t, account.IsBanned(1700000000000))
	})

	t.Run("临时封禁未过期", func(t *testing.T) {
		account := NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)
		_ = account.Ban("违规", 1700000000000+3600000)
		assert.True(t, account.IsBanned(1700000000000+1800000))
	})

	t.Run("临时封禁已过期", func(t *testing.T) {
		account := NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)
		_ = account.Ban("违规", 1700000000000+3600000)
		assert.False(t, account.IsBanned(1700000000000+7200000))
	})

	t.Run("永久封禁", func(t *testing.T) {
		account := NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)
		_ = account.Ban("严重违规", 0)
		assert.True(t, account.IsBanned(1700000000000))
	})
}

// TestPlayerAccount_VerifyPassword 测试密码校验委托。
func TestPlayerAccount_VerifyPassword(t *testing.T) {
	t.Run("密码匹配", func(t *testing.T) {
		account := NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)
		hasher := &mockHasher{verifyResult: true}
		match, err := account.VerifyPassword(context.Background(), "password", hasher)
		require.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("密码不匹配", func(t *testing.T) {
		account := NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)
		hasher := &mockHasher{verifyResult: false}
		match, err := account.VerifyPassword(context.Background(), "wrongpassword", hasher)
		require.NoError(t, err)
		assert.False(t, match)
	})

	t.Run("哈希器未注入", func(t *testing.T) {
		account := NewPlayerAccount(1001, "testuser", "hash", "salt", "127.0.0.1", 1700000000000)
		_, err := account.VerifyPassword(context.Background(), "password", nil)
		assert.Error(t, err)
	})
}

// TestCredential 测试凭证值对象。
func TestCredential(t *testing.T) {
	t.Run("创建与访问", func(t *testing.T) {
		cred := NewCredential("testuser", "password123")
		assert.Equal(t, "testuser", cred.Username())
		assert.Equal(t, "password123", cred.Password())
	})

	t.Run("置零密码", func(t *testing.T) {
		cred := NewCredential("testuser", "password123")
		cred.ZeroPassword()
		assert.Equal(t, "", cred.Password())
	})
}

// 确保mockHasher实现PasswordHasher接口（编译期检查）。
var _ PasswordHasher = (*mockHasher)(nil)

// 确保错误码变量可用（避免未使用import告警）。
var _ = gatewayerr.ErrPasswordHashFailed
