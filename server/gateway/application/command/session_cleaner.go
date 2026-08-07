// Package command Gateway服务application层命令，编排用户认证操作。
package command

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"insectworld/server/shared/pkg/eventbus"

	domainevent "insectworld/server/gateway/domain/event"
	domainsession "insectworld/server/gateway/domain/session"
	"insectworld/server/gateway/infrastructure/config"
)

// 会话超时清理单次批量上限，spec 4.1 性能5要求单次周期处理不少于10000个。
const sessionCleanerBatchSize = 10000

// SessionTimeoutCleaner 会话超时清理后台任务。
//
// 周期触发查询超时会话，遍历销毁并发布下线事件（原因=会话超时）。
// goroutine安全：通过ctx cancel退出（规范9），无裸go func。
type SessionTimeoutCleaner struct {
	sessionRepo domainsession.SessionRepository // 会话仓储
	eventBus    eventbus.EventBus               // 事件总线
	cfg         config.AuthConfig               // 认证配置
	logger      *zap.Logger                     // 结构化日志
}

// NewSessionTimeoutCleaner 创建会话超时清理器实例。
func NewSessionTimeoutCleaner(
	sessionRepo domainsession.SessionRepository,
	eventBus eventbus.EventBus,
	cfg config.AuthConfig,
	logger *zap.Logger,
) *SessionTimeoutCleaner {
	return &SessionTimeoutCleaner{
		sessionRepo: sessionRepo,
		eventBus:    eventBus,
		cfg:         cfg,
		logger:      logger,
	}
}

// Run 启动会话超时清理循环，阻塞执行直到ctx cancel。
//
// ticker周期触发（cleanIntervalMs间隔），每次触发调用FindExpired查询超时会话，
// 遍历销毁并发布下线事件。清理失败时记录Error日志，下一周期重试。
func (c *SessionTimeoutCleaner) Run(ctx context.Context) {
	cleanInterval := time.Duration(c.cfg.SessionTimeoutMs/4) * time.Millisecond
	if cleanInterval < time.Second {
		cleanInterval = time.Second
	}
	ticker := time.NewTicker(cleanInterval)
	defer ticker.Stop()

	c.logger.Info("会话超时清理任务启动",
		zap.Int64("session_timeout_ms", c.cfg.SessionTimeoutMs),
		zap.Duration("clean_interval", cleanInterval),
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("会话超时清理任务退出")
			return
		case <-ticker.C:
			c.cleanOnce(ctx)
		}
	}
}

// cleanOnce 执行一次超时会话清理。
func (c *SessionTimeoutCleaner) cleanOnce(ctx context.Context) {
	now := time.Now().UnixMilli()
	threshold := now - c.cfg.SessionTimeoutMs

	expiredSessions, err := c.sessionRepo.FindExpired(ctx, threshold, sessionCleanerBatchSize)
	if err != nil {
		c.logger.Error("超时会话查询失败", zap.Error(err))
		return
	}

	cleanedCount := 0
	for _, sess := range expiredSessions {
		if err := c.sessionRepo.Delete(ctx, sess.PlayerID()); err != nil {
			c.logger.Error("超时会话销毁失败",
				zap.Int64("player_id", sess.PlayerID()),
				zap.Error(err),
			)
			continue
		}
		c.publishOfflineEvent(ctx, sess.PlayerID(), now, domainevent.OfflineReasonSessionTimeout)
		cleanedCount++
	}

	if cleanedCount > 0 {
		c.logger.Info("超时会话清理完成",
			zap.Int("cleaned_count", cleanedCount),
			zap.Int("total_expired", len(expiredSessions)),
		)
	}
}

// publishOfflineEvent 发布玩家下线事件。
func (c *SessionTimeoutCleaner) publishOfflineEvent(ctx context.Context, playerID, offlineTime int64, reason int) {
	event := &domainevent.PlayerOfflineEvent{
		PlayerID:    playerID,
		OfflineTime: offlineTime,
		Reason:      reason,
	}
	domainEvt, err := event.ToDomainEvent(fmt.Sprintf("offline-%d-%d", playerID, offlineTime), 1)
	if err != nil {
		c.logger.Error("下线事件序列化失败", zap.Error(err))
		return
	}
	if err := c.eventBus.Publish(ctx, domainEvt); err != nil {
		c.logger.Error("下线事件发布失败", zap.Int64("player_id", playerID), zap.Error(err))
	}
}
