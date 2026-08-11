// Package command social服务application写侧，编排alliance聚合根的成员变更。
// 本文件定义RemoveMemberCommand移除联盟成员命令。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/social/domain/alliance"
)

// RemoveMemberCommand 移除联盟成员命令DTO。
type RemoveMemberCommand struct {
	AllianceID int64 // 联盟ID
	PlayerID   int64 // 玩家ID
}

// RemoveMemberHandler 移除联盟成员命令处理器。
type RemoveMemberHandler struct {
	allianceRepo alliance.AllianceRepository // Alliance聚合根仓储接口
	outbox       Outbox                       // 领域事件Outbox接口
	logger       *zap.Logger                  // 结构化日志器（规范7）
}

// NewRemoveMemberHandler 创建移除成员命令处理器实例。
func NewRemoveMemberHandler(
	allianceRepo alliance.AllianceRepository,
	outbox Outbox,
	logger *zap.Logger,
) *RemoveMemberHandler {
	return &RemoveMemberHandler{
		allianceRepo: allianceRepo,
		outbox:       outbox,
		logger:       logger,
	}
}

// Handle 处理移除联盟成员命令。
// 编排：加载Alliance聚合根 → 调用RemoveMember → 保存聚合根 → 写Outbox投递MemberChangedEvent。
func (h *RemoveMemberHandler) Handle(ctx context.Context, cmd RemoveMemberCommand) error {
	// 1. 加载Alliance聚合根
	a, err := h.allianceRepo.LoadAlliance(ctx, cmd.AllianceID)
	if err != nil {
		return fmt.Errorf("移除成员失败，加载联盟失败，allianceID=%d: %w", cmd.AllianceID, err)
	}

	// 2. 调用聚合根方法移除成员
	event, err := a.RemoveMember(ctx, cmd.PlayerID)
	if err != nil {
		return fmt.Errorf("移除成员失败，allianceID=%d，playerID=%d: %w",
			cmd.AllianceID, cmd.PlayerID, err)
	}

	// 3. 保存聚合根
	if err := h.allianceRepo.SaveAlliance(ctx, a); err != nil {
		return fmt.Errorf("移除成员失败，保存联盟失败，allianceID=%d: %w", cmd.AllianceID, err)
	}

	// 4. 写Outbox投递领域事件
	if err := h.outbox.Append(ctx, event); err != nil {
		return fmt.Errorf("移除成员失败，写Outbox失败，allianceID=%d: %w", cmd.AllianceID, err)
	}

	h.logger.Info("移除联盟成员成功",
		zap.Int64("alliance_id", cmd.AllianceID),
		zap.Int64("player_id", cmd.PlayerID),
	)

	return nil
}