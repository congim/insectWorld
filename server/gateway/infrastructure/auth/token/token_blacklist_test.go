package token

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

// newMiniredisClient 启动miniredis并返回客户端与清理函数。
func newMiniredisClient(t *testing.T) (*redis.Client, func()) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return client, func() {
		_ = client.Close()
		mr.Close()
	}
}

// TestTokenBlacklistImpl_InvalidateAndIsInvalid 测试令牌黑名单写入与查询往返。
func TestTokenBlacklistImpl_InvalidateAndIsInvalid(t *testing.T) {
	client, cleanup := newMiniredisClient(t)
	defer cleanup()
	logger := zap.NewNop()
	bl := NewTokenBlacklistImpl(client, logger)
	ctx := context.Background()

	// 初始不在黑名单
	isInvalid, err := bl.IsInvalid(ctx, 1001, 1)
	require.NoError(t, err)
	assert.False(t, isInvalid)

	// 加入黑名单
	require.NoError(t, bl.Invalidate(ctx, 1001, 1, 3600))

	// 查询应在黑名单
	isInvalid, err = bl.IsInvalid(ctx, 1001, 1)
	require.NoError(t, err)
	assert.True(t, isInvalid)
}

// TestTokenBlacklistImpl_DifferentVersions 测试不同令牌版本号独立黑名单。
func TestTokenBlacklistImpl_DifferentVersions(t *testing.T) {
	client, cleanup := newMiniredisClient(t)
	defer cleanup()
	logger := zap.NewNop()
	bl := NewTokenBlacklistImpl(client, logger)
	ctx := context.Background()

	// 版本1加入黑名单
	require.NoError(t, bl.Invalidate(ctx, 1001, 1, 3600))
	isInvalid, err := bl.IsInvalid(ctx, 1001, 1)
	require.NoError(t, err)
	assert.True(t, isInvalid)

	// 版本2不在黑名单
	isInvalid, err = bl.IsInvalid(ctx, 1001, 2)
	require.NoError(t, err)
	assert.False(t, isInvalid)
}

// TestTokenBlacklistImpl_DifferentPlayers 测试不同玩家独立黑名单。
func TestTokenBlacklistImpl_DifferentPlayers(t *testing.T) {
	client, cleanup := newMiniredisClient(t)
	defer cleanup()
	logger := zap.NewNop()
	bl := NewTokenBlacklistImpl(client, logger)
	ctx := context.Background()

	require.NoError(t, bl.Invalidate(ctx, 1001, 1, 3600))
	isInvalid, err := bl.IsInvalid(ctx, 1002, 1)
	require.NoError(t, err)
	assert.False(t, isInvalid, "不同玩家不应共享黑名单")
}

// TestTokenBlacklistImpl_TTLExpiry 测试黑名单TTL过期后自动清理。
func TestTokenBlacklistImpl_TTLExpiry(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer client.Close()

	logger := zap.NewNop()
	bl := NewTokenBlacklistImpl(client, logger)
	ctx := context.Background()

	// 加入黑名单，TTL=1秒
	require.NoError(t, bl.Invalidate(ctx, 1001, 1, 1))

	// 立即查询应在黑名单
	isInvalid, err := bl.IsInvalid(ctx, 1001, 1)
	require.NoError(t, err)
	assert.True(t, isInvalid)

	// 快进时间2秒，TTL过期
	mr.FastForward(2 * time.Second)

	isInvalid, err = bl.IsInvalid(ctx, 1001, 1)
	require.NoError(t, err)
	assert.False(t, isInvalid, "TTL过期后应自动清理")
}

// TestTokenBlacklistImpl_InvalidateZeroTTL 测试TTL<=0时降级为1秒。
func TestTokenBlacklistImpl_InvalidateZeroTTL(t *testing.T) {
	client, cleanup := newMiniredisClient(t)
	defer cleanup()
	logger := zap.NewNop()
	bl := NewTokenBlacklistImpl(client, logger)
	ctx := context.Background()

	// TTL=0应降级为1秒，不报错
	require.NoError(t, bl.Invalidate(ctx, 1001, 1, 0))
	isInvalid, err := bl.IsInvalid(ctx, 1001, 1)
	require.NoError(t, err)
	assert.True(t, isInvalid)
}

// TestTokenBlacklistImpl_RedisError 测试Redis故障时降级语义。
func TestTokenBlacklistImpl_RedisError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	logger := zap.NewNop()
	bl := NewTokenBlacklistImpl(client, logger)
	ctx := context.Background()

	// 关闭miniredis模拟Redis故障
	mr.Close()

	// Invalidate故障返回错误
	err = bl.Invalidate(ctx, 1001, 1, 3600)
	require.Error(t, err)

	// IsInvalid故障返回(false, ErrTokenBlacklistUnavailable)降级为不拒绝
	isInvalid, err := bl.IsInvalid(ctx, 1001, 1)
	require.Error(t, err)
	assert.False(t, isInvalid, "Redis故障应降级为不拒绝")
}
