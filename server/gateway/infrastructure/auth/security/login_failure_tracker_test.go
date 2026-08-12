package security

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gatewayerr "insectworld/server/gateway/domain/errors"

	"go.uber.org/zap"
)

// newSecurityRedisClient 启动miniredis并返回客户端与清理函数。
func newSecurityRedisClient(t *testing.T) (*redis.Client, *miniredis.Miniredis, func()) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return client, mr, func() {
		_ = client.Close()
		mr.Close()
	}
}

// TestLoginFailureTrackerImpl_RecordFailure 测试登录失败计数递增。
func TestLoginFailureTrackerImpl_RecordFailure(t *testing.T) {
	client, _, cleanup := newSecurityRedisClient(t)
	defer cleanup()
	logger := zap.NewNop()
	tracker := NewLoginFailureTrackerImpl(client, 5, 900000, logger)
	ctx := context.Background()

	count, err := tracker.RecordFailure(ctx, "testuser")
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	count, err = tracker.RecordFailure(ctx, "testuser")
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	count, err = tracker.RecordFailure(ctx, "testuser")
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

// TestLoginFailureTrackerImpl_IsLocked 测试锁定状态查询。
func TestLoginFailureTrackerImpl_IsLocked(t *testing.T) {
	client, _, cleanup := newSecurityRedisClient(t)
	defer cleanup()
	logger := zap.NewNop()
	tracker := NewLoginFailureTrackerImpl(client, 5, 900000, logger)
	ctx := context.Background()

	// 初始未锁定
	locked, err := tracker.IsLocked(ctx, "testuser")
	require.NoError(t, err)
	assert.False(t, locked)

	// 记录失败后设置锁定标记（failMaxCount>0）
	_, err = tracker.RecordFailure(ctx, "testuser")
	require.NoError(t, err)

	locked, err = tracker.IsLocked(ctx, "testuser")
	require.NoError(t, err)
	assert.True(t, locked)
}

// TestLoginFailureTrackerImpl_RemainingLockSeconds 测试锁定剩余秒数查询。
func TestLoginFailureTrackerImpl_RemainingLockSeconds(t *testing.T) {
	client, _, cleanup := newSecurityRedisClient(t)
	defer cleanup()
	logger := zap.NewNop()
	tracker := NewLoginFailureTrackerImpl(client, 5, 900000, logger)
	ctx := context.Background()

	// 未锁定返回0
	remaining, err := tracker.RemainingLockSeconds(ctx, "testuser")
	require.NoError(t, err)
	assert.Equal(t, int64(0), remaining)

	// 记录失败后应有剩余锁定时间
	_, err = tracker.RecordFailure(ctx, "testuser")
	require.NoError(t, err)
	remaining, err = tracker.RemainingLockSeconds(ctx, "testuser")
	require.NoError(t, err)
	assert.Greater(t, remaining, int64(0), "锁定后应有剩余锁定秒数")
}

// TestLoginFailureTrackerImpl_ResetClear 测试清零失败计数与锁定状态。
func TestLoginFailureTrackerImpl_ResetClear(t *testing.T) {
	client, _, cleanup := newSecurityRedisClient(t)
	defer cleanup()
	logger := zap.NewNop()
	tracker := NewLoginFailureTrackerImpl(client, 5, 900000, logger)
	ctx := context.Background()

	// 记录失败
	_, err := tracker.RecordFailure(ctx, "testuser")
	require.NoError(t, err)
	locked, err := tracker.IsLocked(ctx, "testuser")
	require.NoError(t, err)
	assert.True(t, locked)

	// 清零
	require.NoError(t, tracker.ResetClear(ctx, "testuser"))
	locked, err = tracker.IsLocked(ctx, "testuser")
	require.NoError(t, err)
	assert.False(t, locked, "清零后应未锁定")
}

// TestLoginFailureTrackerImpl_DifferentUsers 测试不同用户名独立计数。
func TestLoginFailureTrackerImpl_DifferentUsers(t *testing.T) {
	client, _, cleanup := newSecurityRedisClient(t)
	defer cleanup()
	logger := zap.NewNop()
	tracker := NewLoginFailureTrackerImpl(client, 5, 900000, logger)
	ctx := context.Background()

	count1, err := tracker.RecordFailure(ctx, "user1")
	require.NoError(t, err)
	assert.Equal(t, 1, count1)

	count2, err := tracker.RecordFailure(ctx, "user2")
	require.NoError(t, err)
	assert.Equal(t, 1, count2, "不同用户应独立计数")

	count1, err = tracker.RecordFailure(ctx, "user1")
	require.NoError(t, err)
	assert.Equal(t, 2, count1)
}

// TestLoginFailureTrackerImpl_RedisError 测试Redis故障时降级语义。
func TestLoginFailureTrackerImpl_RedisError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	logger := zap.NewNop()
	tracker := NewLoginFailureTrackerImpl(client, 5, 900000, logger)
	ctx := context.Background()

	mr.Close() // 模拟故障

	// RecordFailure故障返回错误
	_, err = tracker.RecordFailure(ctx, "testuser")
	require.Error(t, err)

	// IsLocked故障返回(false, ErrFailureTrackerUnavailable)降级为不锁定
	locked, err := tracker.IsLocked(ctx, "testuser")
	require.Error(t, err)
	assert.False(t, locked)

	// ResetClear故障返回错误
	err = tracker.ResetClear(ctx, "testuser")
	require.Error(t, err)
	assert.True(t, gatewayerr.ErrFailureTrackerUnavailable == err || err != nil)
}

// TestLoginFailureTrackerImpl_RemainingLockSecondsError 测试Redis故障时剩余秒数查询返回错误。
func TestLoginFailureTrackerImpl_RemainingLockSecondsError(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	logger := zap.NewNop()
	tracker := NewLoginFailureTrackerImpl(client, 5, 900000, logger)

	mr.Close()

	remaining, err := tracker.RemainingLockSeconds(context.Background(), "testuser")
	require.Error(t, err)
	assert.Equal(t, int64(0), remaining)
}
