// Package command Gateway服务application层命令，编排用户认证操作。
package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"insectworld/server/shared/pkg/eventbus"

	domainaccount "insectworld/server/gateway/domain/account"
	domainaudit "insectworld/server/gateway/domain/audit"
	gatewayerr "insectworld/server/gateway/domain/errors"
	domainevent "insectworld/server/gateway/domain/event"
	domainsession "insectworld/server/gateway/domain/session"
	domaintoken "insectworld/server/gateway/domain/token"
	"insectworld/server/gateway/infrastructure/websocket"
)

// 封禁踢下线通知消息内容。
const banKickOutMessage = `{"type":"kick_out","reason":"banned"}`

// BanCommand 封禁命令，编排封禁+踢下线流程。
//
// 编排顺序：查询账号→Ban更新封禁状态→持久化→查询在线会话→
// 若存在会话则销毁+令牌失效+推送踢下线+发布下线事件→审计日志→返回。
type BanCommand struct {
	accountRepo    domainaccount.AccountRepository // 账号仓储
	sessionRepo    domainsession.SessionRepository // 会话仓储
	tokenBlacklist domaintoken.TokenBlacklist      // 令牌黑名单
	eventBus       eventbus.EventBus               // 事件总线
	auditLogger    domainaudit.AuditLogger         // 审计日志
	connManager    *websocket.ConnectionManager    // 连接管理器
	logger         *zap.Logger                     // 结构化日志
}

// NewBanCommand 创建封禁命令实例。
func NewBanCommand(
	accountRepo domainaccount.AccountRepository,
	sessionRepo domainsession.SessionRepository,
	tokenBlacklist domaintoken.TokenBlacklist,
	eventBus eventbus.EventBus,
	auditLogger domainaudit.AuditLogger,
	connManager *websocket.ConnectionManager,
	logger *zap.Logger,
) *BanCommand {
	return &BanCommand{
		accountRepo:    accountRepo,
		sessionRepo:    sessionRepo,
		tokenBlacklist: tokenBlacklist,
		eventBus:       eventBus,
		auditLogger:    auditLogger,
		connManager:    connManager,
		logger:         logger,
	}
}

// Handle 执行封禁命令。
//
// 封禁即时生效：账号状态更新成功后立即销毁会话，即使会话销毁失败，
// 封禁账号再次登录时"账号状态校验"会拒绝，已在线旧会话依赖Redis TTL最终过期。
func (c *BanCommand) Handle(ctx context.Context, playerID int64, durationMs int64, reason string, adminID string) error {
	now := time.Now().UnixMilli()
	c.logger.Info("封禁操作接收",
		zap.Int64("player_id", playerID),
		zap.Int64("duration_ms", durationMs),
		zap.String("reason", reason),
		zap.String("admin_id", adminID),
	)

	account, err := c.accountRepo.FindByID(ctx, playerID)
	if err != nil {
		if errors.Is(err, gatewayerr.ErrAccountNotFoundSentinel) {
			return gatewayerr.ErrAccountNotFound
		}
		return fmt.Errorf("账号查询失败: %w", gatewayerr.ErrAccountRepoUnavailable)
	}

	banExpireTime := int64(0)
	if durationMs > 0 {
		banExpireTime = now + durationMs
	}
	if err := account.Ban(reason, banExpireTime); err != nil {
		return err
	}
	if err := c.accountRepo.Save(ctx, account); err != nil {
		return fmt.Errorf("封禁状态持久化失败: %w", gatewayerr.ErrAccountRepoUnavailable)
	}

	c.kickOutIfOnline(ctx, playerID, now)

	_ = c.auditLogger.LogRecord(ctx, &domainaudit.AuditRecord{
		OpType:  domainaudit.OpTypeBanIntercept,
		Subject: fmt.Sprintf("%d", playerID),
		Result:  true,
		OpTime:  now,
		Extra:   fmt.Sprintf(`{"admin_id":"%s","reason":"%s","duration_ms":%d}`, adminID, reason, durationMs),
	})

	c.logger.Info("封禁操作成功",
		zap.Int64("player_id", playerID),
		zap.String("admin_id", adminID),
	)
	return nil
}

// kickOutIfOnline 若玩家在线则踢下线：销毁会话+令牌失效+推送通知+发布下线事件。
func (c *BanCommand) kickOutIfOnline(ctx context.Context, playerID int64, now int64) {
	session, err := c.sessionRepo.FindByPlayerID(ctx, playerID)
	if err != nil {
		return
	}
	_ = c.sessionRepo.Delete(ctx, playerID)
	_ = c.tokenBlacklist.Invalidate(ctx, playerID, session.TokenVersion(), (session.LoginTime()+c.cfgSessionTTL(now)-now)/1000)
	_ = c.connManager.Send(ctx, playerID, []byte(banKickOutMessage))

	event := &domainevent.PlayerOfflineEvent{
		PlayerID:    playerID,
		OfflineTime: now,
		Reason:      domainevent.OfflineReasonBanned,
	}
	domainEvt, _ := event.ToDomainEvent(fmt.Sprintf("offline-%d-%d", playerID, now), 1)
	_ = c.eventBus.Publish(ctx, domainEvt)
}

// cfgSessionTTL 占位方法，实际TTL应由配置注入。此处返回默认5分钟。
//
// TODO 后续从AuthConfig注入SessionTTLMs，当前简化为默认值。
func (c *BanCommand) cfgSessionTTL(now int64) int64 {
	return 300000
}
