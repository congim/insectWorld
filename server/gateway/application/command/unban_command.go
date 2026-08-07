// Package command Gateway服务application层命令，编排用户认证操作。
package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	domainaccount "insectworld/server/gateway/domain/account"
	domainaudit "insectworld/server/gateway/domain/audit"
	gatewayerr "insectworld/server/gateway/domain/errors"
)

// UnbanCommand 解封命令，编排账号解封流程。
//
// 仅更新账号状态为正常，不影响在线会话（design整合步骤7）。
type UnbanCommand struct {
	accountRepo domainaccount.AccountRepository // 账号仓储
	auditLogger domainaudit.AuditLogger         // 审计日志
	logger      *zap.Logger                     // 结构化日志
}

// NewUnbanCommand 创建解封命令实例。
func NewUnbanCommand(
	accountRepo domainaccount.AccountRepository,
	auditLogger domainaudit.AuditLogger,
	logger *zap.Logger,
) *UnbanCommand {
	return &UnbanCommand{
		accountRepo: accountRepo,
		auditLogger: auditLogger,
		logger:      logger,
	}
}

// Handle 执行解封命令。
//
// 查询账号→Unban更新状态→持久化→审计日志。不影响在线会话。
func (c *UnbanCommand) Handle(ctx context.Context, playerID int64, adminID string) error {
	now := time.Now().UnixMilli()
	c.logger.Info("解封操作接收",
		zap.Int64("player_id", playerID),
		zap.String("admin_id", adminID),
	)

	account, err := c.accountRepo.FindByID(ctx, playerID)
	if err != nil {
		if errors.Is(err, gatewayerr.ErrAccountNotFoundSentinel) {
			return gatewayerr.ErrAccountNotFound
		}
		return fmt.Errorf("账号查询失败: %w", gatewayerr.ErrAccountRepoUnavailable)
	}

	if err := account.Unban(); err != nil {
		return err
	}
	if err := c.accountRepo.Save(ctx, account); err != nil {
		return fmt.Errorf("解封状态持久化失败: %w", gatewayerr.ErrAccountRepoUnavailable)
	}

	_ = c.auditLogger.LogRecord(ctx, &domainaudit.AuditRecord{
		OpType:  domainaudit.OpTypeBanIntercept,
		Subject: fmt.Sprintf("%d", playerID),
		Result:  true,
		OpTime:  now,
		Extra:   fmt.Sprintf(`{"admin_id":"%s","action":"unban"}`, adminID),
	})

	c.logger.Info("解封操作成功",
		zap.Int64("player_id", playerID),
		zap.String("admin_id", adminID),
	)
	return nil
}
