// Package command Task服务application层写侧命令，编排聚合根调用、事务边界与领域事件Outbox投递。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/task/domain/taskprogress"
)

// AdvanceTaskCommand 任务进度推进命令DTO。
type AdvanceTaskCommand struct {
	TaskID   int64 // 任务ID
	PlayerID int64 // 玩家ID
	Delta    int64 // 增量进度
	Now      int64 // 当前时间戳（毫秒）
}

// ClaimTaskRewardCommand 领取任务奖励命令DTO。
type ClaimTaskRewardCommand struct {
	TaskID   int64 // 任务ID
	PlayerID int64 // 玩家ID
	Now      int64 // 当前时间戳（毫秒）
}

// AdvanceTaskHandler 任务进度推进命令handler。
type AdvanceTaskHandler struct {
	taskProgressRepo taskprogress.TaskProgressRepository // 任务进度仓储接口，infrastructure层注入
	logger           *zap.Logger                         // 结构化日志器（规范7）
}

// NewAdvanceTaskHandler 创建任务进度推进命令handler实例。
func NewAdvanceTaskHandler(taskProgressRepo taskprogress.TaskProgressRepository, logger *zap.Logger) *AdvanceTaskHandler {
	return &AdvanceTaskHandler{taskProgressRepo: taskProgressRepo, logger: logger}
}

// Handle 处理任务进度推进命令。
// 编排：加载任务进度聚合根→调用聚合根Advance→保存+Outbox。
func (h *AdvanceTaskHandler) Handle(ctx context.Context, cmd AdvanceTaskCommand) error {
	// 1. 加载任务进度聚合根
	tp, err := h.taskProgressRepo.LoadTaskProgress(ctx, cmd.TaskID, cmd.PlayerID)
	if err != nil {
		return fmt.Errorf("加载任务进度失败，taskID=%d，playerID=%d: %w", cmd.TaskID, cmd.PlayerID, err)
	}

	// 2. 调用聚合根Advance
	event, err := tp.Advance(ctx, cmd.Delta, cmd.Now)
	if err != nil {
		h.logger.Warn("任务进度推进失败",
			zap.Int64("task_id", cmd.TaskID),
			zap.Int64("player_id", cmd.PlayerID),
			zap.Error(err),
		)
		return err
	}

	// 3. 保存聚合根
	if err := h.taskProgressRepo.SaveTaskProgress(ctx, tp); err != nil {
		return fmt.Errorf("保存任务进度失败，taskID=%d: %w", cmd.TaskID, err)
	}

	h.logger.Info("任务进度推进",
		zap.Int64("task_id", cmd.TaskID),
		zap.Int64("player_id", cmd.PlayerID),
		zap.Int64("delta", cmd.Delta),
		zap.Bool("completed", event.Completed),
	)
	// TODO 后续接入Outbox投递ProgressChangedEvent
	return nil
}

// ClaimTaskRewardHandler 领取任务奖励命令handler。
type ClaimTaskRewardHandler struct {
	taskProgressRepo taskprogress.TaskProgressRepository // 任务进度仓储接口，infrastructure层注入
	logger           *zap.Logger                         // 结构化日志器（规范7）
}

// NewClaimTaskRewardHandler 创建领取任务奖励命令handler实例。
func NewClaimTaskRewardHandler(taskProgressRepo taskprogress.TaskProgressRepository, logger *zap.Logger) *ClaimTaskRewardHandler {
	return &ClaimTaskRewardHandler{taskProgressRepo: taskProgressRepo, logger: logger}
}

// Handle 处理领取任务奖励命令。
// 编排：加载任务进度聚合根→调用聚合根ClaimReward→调用Inventory/Economy发放奖励→保存+Outbox。
func (h *ClaimTaskRewardHandler) Handle(ctx context.Context, cmd ClaimTaskRewardCommand) error {
	// 1. 加载任务进度聚合根
	tp, err := h.taskProgressRepo.LoadTaskProgress(ctx, cmd.TaskID, cmd.PlayerID)
	if err != nil {
		return fmt.Errorf("加载任务进度失败，taskID=%d，playerID=%d: %w", cmd.TaskID, cmd.PlayerID, err)
	}

	// 2. 调用聚合根ClaimReward
	event, err := tp.ClaimReward(cmd.Now)
	if err != nil {
		h.logger.Warn("领取任务奖励失败",
			zap.Int64("task_id", cmd.TaskID),
			zap.Int64("player_id", cmd.PlayerID),
			zap.Error(err),
		)
		return err
	}

	// 3. 保存聚合根
	if err := h.taskProgressRepo.SaveTaskProgress(ctx, tp); err != nil {
		return fmt.Errorf("保存任务进度失败，taskID=%d: %w", cmd.TaskID, err)
	}

	h.logger.Info("领取任务奖励",
		zap.Int64("task_id", cmd.TaskID),
		zap.Int64("player_id", cmd.PlayerID),
		zap.Int64("def_id", event.DefID),
	)
	// TODO 后续接入Inventory gRPC发放道具奖励、Economy gRPC发放资源奖励、Outbox投递RewardClaimedEvent
	return nil
}
