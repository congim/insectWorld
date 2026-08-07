// Package command Gateway服务application层命令，编排用户认证操作。
package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	gatewayerr "insectworld/server/gateway/domain/errors"
	domainsession "insectworld/server/gateway/domain/session"
	domaintoken "insectworld/server/gateway/domain/token"
)

// HeartbeatCommand 心跳命令，编排会话保活流程。
//
// 编排顺序遵循spec 5.4.2：校验令牌→查询会话→更新心跳时间→返回。
// 心跳不创建新会话（spec 5.4.1 规则6），无效令牌或无会话的心跳请求被拒绝。
type HeartbeatCommand struct {
	tokenSigner domaintoken.TokenSigner         // 令牌签发器
	sessionRepo domainsession.SessionRepository // 会话仓储
	logger      *zap.Logger                     // 结构化日志
}

// NewHeartbeatCommand 创建心跳命令实例。
func NewHeartbeatCommand(
	tokenSigner domaintoken.TokenSigner,
	sessionRepo domainsession.SessionRepository,
	logger *zap.Logger,
) *HeartbeatCommand {
	return &HeartbeatCommand{
		tokenSigner: tokenSigner,
		sessionRepo: sessionRepo,
		logger:      logger,
	}
}

// Handle 执行心跳命令。
//
// 心跳仅更新已存在会话的活跃时间，不创建新会话。
// 无效令牌返回TOKEN_INVALID，无会话返回NOT_LOGGED_IN。
func (c *HeartbeatCommand) Handle(ctx context.Context, req HeartbeatRequest) error {
	now := time.Now().UnixMilli()

	_, err := c.tokenSigner.Verify(ctx, req.AccessToken)
	if err != nil {
		c.logger.Warn("心跳令牌校验失败", zap.Int64("player_id", req.PlayerID), zap.Error(err))
		return gatewayerr.ErrTokenInvalid
	}

	session, err := c.sessionRepo.FindByPlayerID(ctx, req.PlayerID)
	if err != nil {
		if errors.Is(err, gatewayerr.ErrSessionNotFound) {
			c.logger.Warn("心跳会话不存在", zap.Int64("player_id", req.PlayerID))
			return gatewayerr.ErrNotLoggedIn
		}
		return fmt.Errorf("会话查询失败: %w", err)
	}

	if err := session.UpdateHeartbeat(now); err != nil {
		return err
	}
	if err := c.sessionRepo.Save(ctx, session); err != nil {
		return fmt.Errorf("会话心跳更新失败: %w", err)
	}

	c.logger.Debug("心跳更新成功", zap.Int64("player_id", req.PlayerID))
	return nil
}
