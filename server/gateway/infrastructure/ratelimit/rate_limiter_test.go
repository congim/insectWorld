// Package ratelimit 多维度限流器infrastructure层实现，整合现有令牌桶算法。
package ratelimit

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.uber.org/zap"
)

// TestRateLimiterImpl_AllowWithinBurst 测试桶容量内请求全部允许。
func TestRateLimiterImpl_AllowWithinBurst(t *testing.T) {
	logger := zap.NewNop()
	cfg := map[string]RateConfig{
		"register:ip": {Rate: 1, Burst: 5},
	}
	limiter := NewRateLimiterImpl(cfg, logger)

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		assert.True(t, limiter.Allow(ctx, "register:ip", "1.2.3.4"), "桶容量5内第%d次请求应允许", i+1)
	}
}

// TestRateLimiterImpl_AllowExceedBurst 测试超出桶容量请求被拒绝。
func TestRateLimiterImpl_AllowExceedBurst(t *testing.T) {
	logger := zap.NewNop()
	cfg := map[string]RateConfig{
		"register:ip": {Rate: 1, Burst: 3},
	}
	limiter := NewRateLimiterImpl(cfg, logger)

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		assert.True(t, limiter.Allow(ctx, "register:ip", "1.2.3.4"))
	}
	// 第4次应被拒绝
	assert.False(t, limiter.Allow(ctx, "register:ip", "1.2.3.4"), "超出桶容量应拒绝")
}

// TestRateLimiterImpl_UnknownDimension 测试未配置维度降级为允许。
func TestRateLimiterImpl_UnknownDimension(t *testing.T) {
	logger := zap.NewNop()
	cfg := map[string]RateConfig{
		"register:ip": {Rate: 1, Burst: 1},
	}
	limiter := NewRateLimiterImpl(cfg, logger)
	ctx := context.Background()

	// 未配置的维度应允许（降级语义）
	assert.True(t, limiter.Allow(ctx, "unknown:dim", "key1"))
	assert.True(t, limiter.Allow(ctx, "unknown:dim", "key1"))
}

// TestRateLimiterImpl_DifferentKeysIsolated 测试不同key的令牌桶相互隔离。
func TestRateLimiterImpl_DifferentKeysIsolated(t *testing.T) {
	logger := zap.NewNop()
	cfg := map[string]RateConfig{
		"register:ip": {Rate: 1, Burst: 2},
	}
	limiter := NewRateLimiterImpl(cfg, logger)
	ctx := context.Background()

	// IP1耗尽令牌
	assert.True(t, limiter.Allow(ctx, "register:ip", "1.1.1.1"))
	assert.True(t, limiter.Allow(ctx, "register:ip", "1.1.1.1"))
	assert.False(t, limiter.Allow(ctx, "register:ip", "1.1.1.1"))

	// IP2应有独立令牌桶
	assert.True(t, limiter.Allow(ctx, "register:ip", "2.2.2.2"))
	assert.True(t, limiter.Allow(ctx, "register:ip", "2.2.2.2"))
	assert.False(t, limiter.Allow(ctx, "register:ip", "2.2.2.2"))
}

// TestRateLimiterImpl_DifferentDimensionsIsolated 测试不同维度的令牌桶相互隔离。
func TestRateLimiterImpl_DifferentDimensionsIsolated(t *testing.T) {
	logger := zap.NewNop()
	cfg := map[string]RateConfig{
		"register:ip":   {Rate: 1, Burst: 1},
		"login:account": {Rate: 1, Burst: 1},
	}
	limiter := NewRateLimiterImpl(cfg, logger)
	ctx := context.Background()

	// 同一key不同维度应隔离
	assert.True(t, limiter.Allow(ctx, "register:ip", "shared-key"))
	assert.False(t, limiter.Allow(ctx, "register:ip", "shared-key"))

	assert.True(t, limiter.Allow(ctx, "login:account", "shared-key"))
	assert.False(t, limiter.Allow(ctx, "login:account", "shared-key"))
}

// TestRateLimiterImpl_ConcurrentSafe 测试限流器并发安全。
func TestRateLimiterImpl_ConcurrentSafe(t *testing.T) {
	logger := zap.NewNop()
	cfg := map[string]RateConfig{
		"register:ip": {Rate: 100, Burst: 1000},
	}
	limiter := NewRateLimiterImpl(cfg, logger)
	ctx := context.Background()

	var wg sync.WaitGroup
	allowCount := 0
	var mu sync.Mutex

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if limiter.Allow(ctx, "register:ip", "1.2.3.4") {
					mu.Lock()
					allowCount++
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	// 并发下不应panic，且允许数应大于0
	mu.Lock()
	defer mu.Unlock()
	assert.Greater(t, allowCount, 0, "并发请求应有部分被允许")
}
