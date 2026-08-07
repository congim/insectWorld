// Package command Economy服务application层命令，编排domain层聚合根与配置查询。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/economy/domain/wallet"
	"insectworld/server/shared/pkg/config"
)

// CollectResourceCommand 采集资源命令DTO。
type CollectResourceCommand struct {
	PlayerID        int64 // 玩家ID
	ResourcePointID int64 // 资源点ID
	ResourceType    int64 // 资源类型ID
}

// CollectResourceHandler 采集资源命令处理器。
type CollectResourceHandler struct {
	walletRepo  wallet.WalletRepository // 玩家钱包仓储接口
	configQuery config.ConfigQueryAPI   // 配置查询接口
	outbox      Outbox                  // 领域事件Outbox接口
	logger      *zap.Logger             // 结构化日志器（规范7）
}

// Outbox 领域事件Outbox接口。
type Outbox interface {
	Append(ctx context.Context, event any) error
}

// NewCollectResourceHandler 创建采集资源命令处理器实例。
func NewCollectResourceHandler(
	walletRepo wallet.WalletRepository,
	configQuery config.ConfigQueryAPI,
	outbox Outbox,
	logger *zap.Logger,
) *CollectResourceHandler {
	return &CollectResourceHandler{
		walletRepo:  walletRepo,
		configQuery: configQuery,
		outbox:      outbox,
		logger:      logger,
	}
}

// Handle 处理采集资源命令。
func (h *CollectResourceHandler) Handle(ctx context.Context, cmd CollectResourceCommand) error {
	w, err := h.walletRepo.LoadWallet(ctx, cmd.PlayerID)
	if err != nil {
		return fmt.Errorf("采集失败，加载钱包失败，playerID=%d: %w", cmd.PlayerID, err)
	}

	// TODO 后续从config查询采集规则计算产出量
	produced := int64(100)
	w.AddBalance(cmd.ResourceType, produced)

	if err := h.walletRepo.SaveWallet(ctx, w); err != nil {
		return fmt.Errorf("采集失败，保存钱包失败: %w", err)
	}

	h.logger.Info("采集资源成功",
		zap.Int64("player_id", cmd.PlayerID),
		zap.Int64("resource_point_id", cmd.ResourcePointID),
		zap.Int64("produced", produced),
	)
	return nil
}
