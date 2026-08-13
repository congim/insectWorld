// Package command CrossServer服务application层写侧命令，编排聚合根调用、事务边界与领域事件Outbox投递。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/crossserver/domain/crossserveractivity"
	"insectworld/server/crossserver/domain/mergetask"
	"insectworld/server/crossserver/domain/servernode"
)

// RegisterNodeCommand 节点注册命令DTO。
type RegisterNodeCommand struct {
	ZoneID  int64  // 区ID
	Role    int    // 节点角色
	Version string // 节点版本号
	Host    string // 主机地址
	Port    int32  // 监听端口
	MaxLoad int64  // 最大负载
	Now     int64  // 当前时间戳（毫秒）
}

// StartActivityCommand 开始跨服活动命令DTO。
type StartActivityCommand struct {
	ActivityID int64 // 活动ID
	Now        int64 // 当前时间戳（毫秒）
}

// StartMergeCommand 开始合服命令DTO。
type StartMergeCommand struct {
	TaskID int64 // 合服任务ID
}

// RegisterNodeHandler 节点注册命令handler。
type RegisterNodeHandler struct {
	serverNodeRepo servernode.ServerNodeRepository // 节点仓储接口，infrastructure层注入
	logger         *zap.Logger                     // 结构化日志器（规范7）
}

// NewRegisterNodeHandler 创建节点注册命令handler实例。
func NewRegisterNodeHandler(serverNodeRepo servernode.ServerNodeRepository, logger *zap.Logger) *RegisterNodeHandler {
	return &RegisterNodeHandler{serverNodeRepo: serverNodeRepo, logger: logger}
}

// Handle 处理节点注册命令。
// 编排：创建节点聚合根→保存→更新路由表+Outbox。
func (h *RegisterNodeHandler) Handle(ctx context.Context, cmd RegisterNodeCommand) error {
	// 1. 构造节点ID（TODO 后续接入雪花算法ID生成器）
	var nodeID int64 = 0

	// 2. 创建节点聚合根
	node := servernode.NewServerNode(nodeID, cmd.ZoneID, cmd.Role, cmd.Version, cmd.Host, cmd.Port, cmd.MaxLoad, cmd.Now)

	// 3. 保存聚合根
	if err := h.serverNodeRepo.SaveServerNode(ctx, node); err != nil {
		return fmt.Errorf("保存节点失败: %w", err)
	}

	h.logger.Info("节点注册",
		zap.Int64("zone_id", cmd.ZoneID),
		zap.Int("role", cmd.Role),
		zap.String("version", cmd.Version),
	)
	// TODO 后续接入节点路由表更新、Outbox投递事件
	return nil
}

// StartActivityHandler 开始跨服活动命令handler。
type StartActivityHandler struct {
	activityRepo crossserveractivity.CrossServerActivityRepository // 跨服活动仓储接口，infrastructure层注入
	logger       *zap.Logger                                       // 结构化日志器（规范7）
}

// NewStartActivityHandler 创建开始跨服活动命令handler实例。
func NewStartActivityHandler(activityRepo crossserveractivity.CrossServerActivityRepository, logger *zap.Logger) *StartActivityHandler {
	return &StartActivityHandler{activityRepo: activityRepo, logger: logger}
}

// Handle 处理开始跨服活动命令。
// 编排：加载活动聚合根→调用聚合根Start→保存+Outbox。
func (h *StartActivityHandler) Handle(ctx context.Context, cmd StartActivityCommand) error {
	// 1. 加载跨服活动聚合根
	activity, err := h.activityRepo.LoadActivity(ctx, cmd.ActivityID)
	if err != nil {
		return fmt.Errorf("加载跨服活动失败，activityID=%d: %w", cmd.ActivityID, err)
	}

	// 2. 调用聚合根Start
	if err := activity.Start(cmd.Now); err != nil {
		h.logger.Warn("开始跨服活动失败",
			zap.Int64("activity_id", cmd.ActivityID),
			zap.Error(err),
		)
		return err
	}

	// 3. 保存聚合根
	if err := h.activityRepo.SaveActivity(ctx, activity); err != nil {
		return fmt.Errorf("保存跨服活动失败，activityID=%d: %w", cmd.ActivityID, err)
	}

	h.logger.Info("开始跨服活动", zap.Int64("activity_id", cmd.ActivityID))
	// TODO 后续接入Outbox投递事件
	return nil
}

// StartMergeHandler 开始合服命令handler。
type StartMergeHandler struct {
	mergeTaskRepo mergetask.MergeTaskRepository // 合服任务仓储接口，infrastructure层注入
	logger        *zap.Logger                   // 结构化日志器（规范7）
}

// NewStartMergeHandler 创建开始合服命令handler实例。
func NewStartMergeHandler(mergeTaskRepo mergetask.MergeTaskRepository, logger *zap.Logger) *StartMergeHandler {
	return &StartMergeHandler{mergeTaskRepo: mergeTaskRepo, logger: logger}
}

// Handle 处理开始合服命令。
// 编排：加载合服任务聚合根→调用聚合根Start→保存+Outbox。
func (h *StartMergeHandler) Handle(ctx context.Context, cmd StartMergeCommand) error {
	// 1. 加载合服任务聚合根
	task, err := h.mergeTaskRepo.LoadMergeTask(ctx, cmd.TaskID)
	if err != nil {
		return fmt.Errorf("加载合服任务失败，taskID=%d: %w", cmd.TaskID, err)
	}

	// 2. 调用聚合根Start
	if err := task.Start(); err != nil {
		h.logger.Warn("开始合服失败",
			zap.Int64("task_id", cmd.TaskID),
			zap.Error(err),
		)
		return err
	}

	// 3. 保存聚合根
	if err := h.mergeTaskRepo.SaveMergeTask(ctx, task); err != nil {
		return fmt.Errorf("保存合服任务失败，taskID=%d: %w", cmd.TaskID, err)
	}

	h.logger.Info("开始合服", zap.Int64("task_id", cmd.TaskID))
	// TODO 后续接入数据迁移编排、Outbox投递事件
	return nil
}
