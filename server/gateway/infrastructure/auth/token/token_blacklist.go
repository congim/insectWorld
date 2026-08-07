// Package token 令牌签发与黑名单infrastructure层实现。
package token

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	gatewayerr "insectworld/server/gateway/domain/errors"
)

// TokenBlacklistImpl Redis令牌黑名单实现，实现TokenBlacklist接口。
//
// Redis key设计：token_blacklist:{playerID}:{tokenVersion}，TTL与令牌剩余有效期对齐自动清理。
// Redis故障时IsInvalid返回(false, ErrTokenBlacklistUnavailable)，鉴权层降级为"不拒绝"。
type TokenBlacklistImpl struct {
	client *redis.Client // Redis客户端
	logger *zap.Logger   // 结构化日志
}

// NewTokenBlacklistImpl 创建令牌黑名单Redis实现实例。
func NewTokenBlacklistImpl(client *redis.Client, logger *zap.Logger) *TokenBlacklistImpl {
	return &TokenBlacklistImpl{
		client: client,
		logger: logger,
	}
}

// Invalidate 将令牌加入黑名单，设置TTL为令牌剩余有效期，自动清理。
func (b *TokenBlacklistImpl) Invalidate(ctx context.Context, playerID int64, tokenVersion int, remainingTTLSeconds int64) error {
	key := fmt.Sprintf("token_blacklist:%d:%d", playerID, tokenVersion)
	ttl := time.Duration(remainingTTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 1 * time.Second
	}
	if err := b.client.Set(ctx, key, 1, ttl).Err(); err != nil {
		b.logger.Error("令牌黑名单写入失败",
			zap.Int64("player_id", playerID),
			zap.Int("token_version", tokenVersion),
			zap.Error(err),
		)
		return fmt.Errorf("令牌黑名单写入失败: %w", gatewayerr.ErrTokenBlacklistUnavailable)
	}
	return nil
}

// IsInvalid 查询令牌是否在黑名单中。
//
// 返回true表示已失效，Redis故障返回(false, ErrTokenBlacklistUnavailable)降级为"不拒绝"。
func (b *TokenBlacklistImpl) IsInvalid(ctx context.Context, playerID int64, tokenVersion int) (bool, error) {
	key := fmt.Sprintf("token_blacklist:%d:%d", playerID, tokenVersion)
	result, err := b.client.Exists(ctx, key).Result()
	if err != nil {
		b.logger.Error("令牌黑名单查询失败",
			zap.Int64("player_id", playerID),
			zap.Int("token_version", tokenVersion),
			zap.Error(err),
		)
		return false, gatewayerr.ErrTokenBlacklistUnavailable
	}
	return result > 0, nil
}
