// Package security 登录安全infrastructure层实现，提供暴力破解防护的Redis适配。
//
// LoginFailureTrackerImpl基于Redis实现失败计数与锁定状态管理，
// Redis key设计遵循design.md 2.3.2.3节，TTL与锁定时长对齐。
package security

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	gatewayerr "insectworld/server/gateway/domain/errors"
)

// LoginFailureTrackerImpl Redis登录失败计数器，实现LoginFailureTracker接口。
//
// Redis key设计：
// - 失败计数：login_fail_count:{username}，TTL=lockDurationMs
// - 锁定状态：login_locked:{username}，TTL=lockDurationMs
// Redis故障时各方法返回ErrFailureTrackerUnavailable，降级为"不锁定"（允许登录，依赖限流器兜底）。
type LoginFailureTrackerImpl struct {
	client         *redis.Client // Redis客户端
	failMaxCount   int           // 登录失败最大次数
	lockDurationMs int64         // 登录锁定时长，毫秒级
	logger         *zap.Logger   // 结构化日志
}

// NewLoginFailureTrackerImpl 创建登录失败计数器Redis实现实例。
func NewLoginFailureTrackerImpl(client *redis.Client, failMaxCount int, lockDurationMs int64, logger *zap.Logger) *LoginFailureTrackerImpl {
	return &LoginFailureTrackerImpl{
		client:         client,
		failMaxCount:   failMaxCount,
		lockDurationMs: lockDurationMs,
		logger:         logger,
	}
}

// RecordFailure 记录一次登录失败，递增失败计数。
//
// 返回当前失败次数，达failMaxCount时设置锁定标记。
// Redis故障返回ErrFailureTrackerUnavailable，降级为"不锁定"。
func (t *LoginFailureTrackerImpl) RecordFailure(ctx context.Context, username string) (int, error) {
	countKey := fmt.Sprintf("login_fail_count:%s", username)
	lockKey := fmt.Sprintf("login_locked:%s", username)
	lockTTL := time.Duration(t.lockDurationMs) * time.Millisecond

	pipe := t.client.Pipeline()
	incrCmd := pipe.Incr(ctx, countKey)
	pipe.Expire(ctx, countKey, lockTTL)
	if t.failMaxCount > 0 {
		pipe.Set(ctx, lockKey, 1, lockTTL)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		t.logger.Error("登录失败计数写入失败",
			zap.String("username", username),
			zap.Error(err),
		)
		return 0, fmt.Errorf("登录失败计数写入失败: %w", gatewayerr.ErrFailureTrackerUnavailable)
	}

	count := int(incrCmd.Val())
	return count, nil
}

// IsLocked 查询账号是否处于锁定状态。
//
// 返回true表示已锁定，Redis故障返回(false, ErrFailureTrackerUnavailable)降级为"不锁定"。
func (t *LoginFailureTrackerImpl) IsLocked(ctx context.Context, username string) (bool, error) {
	lockKey := fmt.Sprintf("login_locked:%s", username)
	result, err := t.client.Exists(ctx, lockKey).Result()
	if err != nil {
		t.logger.Error("锁定状态查询失败",
			zap.String("username", username),
			zap.Error(err),
		)
		return false, gatewayerr.ErrFailureTrackerUnavailable
	}
	return result > 0, nil
}

// RemainingLockSeconds 查询账号锁定剩余秒数。
//
// 未锁定返回0，Redis故障返回(0, ErrFailureTrackerUnavailable)。
func (t *LoginFailureTrackerImpl) RemainingLockSeconds(ctx context.Context, username string) (int64, error) {
	lockKey := fmt.Sprintf("login_locked:%s", username)
	ttl, err := t.client.TTL(ctx, lockKey).Result()
	if err != nil {
		return 0, gatewayerr.ErrFailureTrackerUnavailable
	}
	if ttl < 0 {
		return 0, nil
	}
	return int64(ttl.Seconds()), nil
}

// ResetClear 清零失败计数与锁定状态，登录成功时调用。
func (t *LoginFailureTrackerImpl) ResetClear(ctx context.Context, username string) error {
	countKey := fmt.Sprintf("login_fail_count:%s", username)
	lockKey := fmt.Sprintf("login_locked:%s", username)

	_, err := t.client.Del(ctx, countKey, lockKey).Result()
	if err != nil {
		t.logger.Error("失败计数清零失败",
			zap.String("username", username),
			zap.Error(err),
		)
		return fmt.Errorf("失败计数清零失败: %w", gatewayerr.ErrFailureTrackerUnavailable)
	}
	return nil
}
