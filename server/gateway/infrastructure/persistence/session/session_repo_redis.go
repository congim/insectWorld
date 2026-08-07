// Package session 会话仓储infrastructure层实现，提供Redis与内存适配。
//
// SessionRepoRedis基于Redis实现SessionRepository接口，会话JSON序列化存储。
// Redis key设计：session:{playerID}，TTL与会话超时对齐。
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	gatewayerr "insectworld/server/gateway/domain/errors"
	domainsession "insectworld/server/gateway/domain/session"
)

// SessionRepoRedis Redis会话仓储，实现SessionRepository接口。
type SessionRepoRedis struct {
	client           *redis.Client // Redis客户端
	sessionTimeoutMs int64         // 会话超时时间，毫秒级，用于TTL
	logger           *zap.Logger   // 结构化日志
}

// NewSessionRepoRedis 创建Redis会话仓储实例。
func NewSessionRepoRedis(client *redis.Client, sessionTimeoutMs int64, logger *zap.Logger) *SessionRepoRedis {
	return &SessionRepoRedis{
		client:           client,
		sessionTimeoutMs: sessionTimeoutMs,
		logger:           logger,
	}
}

// Save 保存会话聚合根，写入Redis。
func (r *SessionRepoRedis) Save(ctx context.Context, sess *domainsession.OnlineSession) error {
	key := fmt.Sprintf("session:%d", sess.PlayerID())
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("会话序列化失败: %w", gatewayerr.ErrSessionRepoUnavailable)
	}
	ttl := time.Duration(r.sessionTimeoutMs) * time.Millisecond
	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		r.logger.Error("会话保存失败",
			zap.Int64("player_id", sess.PlayerID()),
			zap.Error(err),
		)
		return fmt.Errorf("会话保存失败: %w", gatewayerr.ErrSessionRepoUnavailable)
	}
	return nil
}

// FindByPlayerID 按玩家ID查询在线会话。
//
// 会话不存在返回ErrSessionNotFound。
func (r *SessionRepoRedis) FindByPlayerID(ctx context.Context, playerID int64) (*domainsession.OnlineSession, error) {
	key := fmt.Sprintf("session:%d", playerID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, gatewayerr.ErrSessionNotFound
		}
		r.logger.Error("会话查询失败",
			zap.Int64("player_id", playerID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("会话查询失败: %w", gatewayerr.ErrSessionRepoUnavailable)
	}
	var sess domainsession.OnlineSession
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("会话反序列化失败: %w", gatewayerr.ErrSessionRepoUnavailable)
	}
	return &sess, nil
}

// Delete 删除会话，玩家登出或踢下线时调用。
func (r *SessionRepoRedis) Delete(ctx context.Context, playerID int64) error {
	key := fmt.Sprintf("session:%d", playerID)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		r.logger.Error("会话删除失败",
			zap.Int64("player_id", playerID),
			zap.Error(err),
		)
		return fmt.Errorf("会话删除失败: %w", gatewayerr.ErrSessionRepoUnavailable)
	}
	return nil
}

// FindExpired 查询超时会话，供SessionTimeoutCleaner周期清理。
//
// Redis实现通过SCAN匹配session:*逐个检查heartbeatTime，limit控制单次返回数。
func (r *SessionRepoRedis) FindExpired(ctx context.Context, thresholdTime int64, limit int) ([]*domainsession.OnlineSession, error) {
	var expiredSessions []*domainsession.OnlineSession
	iter := r.client.Scan(ctx, 0, "session:*", int64(limit*2)).Iterator()
	count := 0
	for iter.Next(ctx) {
		if count >= limit {
			break
		}
		key := iter.Val()
		data, err := r.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var sess domainsession.OnlineSession
		if err := json.Unmarshal(data, &sess); err != nil {
			continue
		}
		if sess.HeartbeatTime() < thresholdTime {
			expiredSessions = append(expiredSessions, &sess)
			count++
		}
	}
	if err := iter.Err(); err != nil {
		r.logger.Error("超时会话扫描失败", zap.Error(err))
		return nil, fmt.Errorf("超时会话扫描失败: %w", gatewayerr.ErrSessionRepoUnavailable)
	}
	return expiredSessions, nil
}
