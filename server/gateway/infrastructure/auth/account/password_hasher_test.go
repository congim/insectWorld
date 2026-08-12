// Package account 密码哈希infrastructure层实现，提供bcrypt适配。
package account

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// TestBcryptHasher_HashAndVerify 测试bcrypt哈希与校验往返。
func TestBcryptHasher_HashAndVerify(t *testing.T) {
	logger := zap.NewNop()
	hasher := NewBcryptHasher(bcrypt.MinCost, logger)
	ctx := context.Background()

	t.Run("正确密码校验通过", func(t *testing.T) {
		hash, salt, err := hasher.Hash(ctx, "myPassword123")
		require.NoError(t, err)
		assert.NotEmpty(t, hash, "哈希值不应为空")
		assert.Equal(t, "", salt, "bcrypt自带盐，salt字段应为空字符串")

		match, err := hasher.Verify(ctx, "myPassword123", hash, salt)
		require.NoError(t, err)
		assert.True(t, match, "正确密码应校验通过")
	})

	t.Run("错误密码校验失败", func(t *testing.T) {
		hash, salt, err := hasher.Hash(ctx, "correctPassword123")
		require.NoError(t, err)

		match, err := hasher.Verify(ctx, "wrongPassword456", hash, salt)
		require.NoError(t, err)
		assert.False(t, match, "错误密码应校验失败")
	})
}

// TestBcryptHasher_DifferentPasswordsDifferentHash 测试不同密码生成不同哈希。
func TestBcryptHasher_DifferentPasswordsDifferentHash(t *testing.T) {
	logger := zap.NewNop()
	hasher := NewBcryptHasher(bcrypt.MinCost, logger)
	ctx := context.Background()

	hash1, _, err := hasher.Hash(ctx, "passwordOne123")
	require.NoError(t, err)
	hash2, _, err := hasher.Hash(ctx, "passwordTwo456")
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "不同密码应生成不同哈希")
}

// TestBcryptHasher_SamePasswordDifferentHash 测试同一密码两次哈希生成不同哈希（bcrypt自带随机盐）。
func TestBcryptHasher_SamePasswordDifferentHash(t *testing.T) {
	logger := zap.NewNop()
	hasher := NewBcryptHasher(bcrypt.MinCost, logger)
	ctx := context.Background()

	hash1, _, err := hasher.Hash(ctx, "samePassword123")
	require.NoError(t, err)
	hash2, _, err := hasher.Hash(ctx, "samePassword123")
	require.NoError(t, err)

	assert.NotEqual(t, hash1, hash2, "同一密码两次哈希应不同（bcrypt随机盐）")

	// 两个哈希都应能校验同一密码
	match1, err := hasher.Verify(ctx, "samePassword123", hash1, "")
	require.NoError(t, err)
	assert.True(t, match1)
	match2, err := hasher.Verify(ctx, "samePassword123", hash2, "")
	require.NoError(t, err)
	assert.True(t, match2)
}

// TestNewBcryptHasher_InvalidCost 测试无效计算成本降级为默认值10。
func TestNewBcryptHasher_InvalidCost(t *testing.T) {
	logger := zap.NewNop()

	t.Run("过低cost降级", func(t *testing.T) {
		hasher := NewBcryptHasher(-1, logger)
		assert.Equal(t, 10, hasher.cost, "无效cost应降级为10")
	})

	t.Run("过高cost降级", func(t *testing.T) {
		hasher := NewBcryptHasher(100, logger)
		assert.Equal(t, 10, hasher.cost, "无效cost应降级为10")
	})

	t.Run("合法cost保留", func(t *testing.T) {
		hasher := NewBcryptHasher(bcrypt.MinCost, logger)
		assert.Equal(t, bcrypt.MinCost, hasher.cost)
	})
}

// TestBcryptHasher_VerifyInvalidHash 测试校验非法哈希字符串返回错误。
func TestBcryptHasher_VerifyInvalidHash(t *testing.T) {
	logger := zap.NewNop()
	hasher := NewBcryptHasher(bcrypt.MinCost, logger)
	ctx := context.Background()

	_, err := hasher.Verify(ctx, "password", "not-a-valid-bcrypt-hash", "")
	assert.Error(t, err, "非法哈希应返回error")
}
