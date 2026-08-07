// Package command Gateway服务application层命令，编排用户认证操作。
package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"insectworld/server/shared/pkg/eventbus"

	domainaudit "insectworld/server/gateway/domain/audit"
	gatewayerr "insectworld/server/gateway/domain/errors"
	domainevent "insectworld/server/gateway/domain/event"
	domainsession "insectworld/server/gateway/domain/session"
	domaintoken "insectworld/server/gateway/domain/token"
)

// LogoutCommand 登出命令，编排用户登出流程。
//
// 编排顺序遵循spec 5.3.2：校验令牌→查询会话→销毁会话→标记令牌失效→
// 发布下线事件→审计日志→返回。令牌无效或会话不存在的登出请求视为幂等返回成功。
type LogoutCommand struct {
	tokenSigner    domaintoken.TokenSigner         // 令牌签发器
	tokenBlacklist domaintoken.TokenBlacklist      // 令牌黑名单
	sessionRepo    domainsession.SessionRepository // 会话仓储
	eventBus       eventbus.EventBus               // 事件总线
	auditLogger    domainaudit.AuditLogger         // 审计日志
	logger         *zap.Logger                     // 结构化日志
}

// NewLogoutCommand 创建登出命令实例。
func NewLogoutCommand(
	tokenSigner domaintoken.TokenSigner,
	tokenBlacklist domaintoken.TokenBlacklist,
	sessionRepo domainsession.SessionRepository,
	eventBus eventbus.EventBus,
	auditLogger domainaudit.AuditLogger,
	logger *zap.Logger,
) *LogoutCommand {
	return &LogoutCommand{
		tokenSigner:    tokenSigner,
		tokenBlacklist: tokenBlacklist,
		sessionRepo:    sessionRepo,
		eventBus:       eventBus,
		auditLogger:    auditLogger,
		logger:         logger,
	}
}

// Handle 执行登出命令。
//
// 令牌无效或会话不存在的登出请求视为无操作直接返回成功（幂等语义，spec 5.3.1 规则1）。
// 登出不修改账号档案（spec 5.3.1 规则5），令牌即刻失效（spec 5.3.1 规则3）。
func (c *LogoutCommand) Handle(ctx context.Context, req LogoutRequest) (*LogoutResponse, error) {
	now := time.Now().UnixMilli()
	c.logger.Info("登出请求接收", zap.Int64("player_id", req.PlayerID))

	payload, err := c.tokenSigner.Verify(ctx, req.AccessToken)
	if err != nil {
		c.logger.Info("登出令牌无效，幂等返回成功", zap.Int64("player_id", req.PlayerID))
		return &LogoutResponse{Success: true}, nil
	}

	_, err = c.sessionRepo.FindByPlayerID(ctx, req.PlayerID)
	if err != nil {
		if errors.Is(err, gatewayerr.ErrSessionNotFound) {
			c.logger.Info("登出会话不存在，幂等返回成功", zap.Int64("player_id", req.PlayerID))
			return &LogoutResponse{Success: true}, nil
		}
		return nil, fmt.Errorf("会话查询失败: %w", gatewayerr.ErrLogoutInternalError)
	}

	if err := c.sessionRepo.Delete(ctx, req.PlayerID); err != nil {
		return nil, fmt.Errorf("会话销毁失败: %w", gatewayerr.ErrLogoutInternalError)
	}

	remainingTTL := (payload.ExpireTime - now) / 1000
	if remainingTTL > 0 {
		_ = c.tokenBlacklist.Invalidate(ctx, req.PlayerID, payload.Version, remainingTTL)
	}

	c.publishOfflineEvent(ctx, req.PlayerID, now, domainevent.OfflineReasonLogout)

	_ = c.auditLogger.LogRecord(ctx, &domainaudit.AuditRecord{
		OpType:  domainaudit.OpTypeLogout,
		Subject: fmt.Sprintf("%d", req.PlayerID),
		Result:  true,
		OpTime:  now,
	})

	c.logger.Info("登出成功", zap.Int64("player_id", req.PlayerID))
	return &LogoutResponse{Success: true}, nil
}

// publishOfflineEvent 发布玩家下线事件。
func (c *LogoutCommand) publishOfflineEvent(ctx context.Context, playerID, offlineTime int64, reason int) {
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
