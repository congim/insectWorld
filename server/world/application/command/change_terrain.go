// Package command World服务application层命令，编排domain层聚合根与跨服务调用。
// 本文件定义ChangeTerrainCommand地形变更命令，对应design.md 2.2.2.1节调用示例。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	worlderr "insectworld/server/world/domain/errors"
	"insectworld/server/world/domain/gamemap"
	"insectworld/server/world/domain/vo"
	"insectworld/server/shared/pkg/config"
)

// ChangeTerrainCommand 地形变更命令DTO。
type ChangeTerrainCommand struct {
	MapID        int64       // 地图ID
	Position     vo.Position // 变更坐标
	NewTerrainID int32       // 变更后地形类型ID
	OperatorID   int64       // 操作者ID（玩家或运营管理员）
}

// ChangeTerrainHandler 地形变更命令处理器，编排domain层Map聚合根与跨服务调用。
// 依赖通过接口注入（规范3），不直接import infrastructure。
type ChangeTerrainHandler struct {
	mapRepo       gamemap.MapRepository // Map聚合根仓储接口
	configQuery   config.ConfigQueryAPI // 配置查询接口，校验目标地形存在
	outbox        Outbox                // 领域事件Outbox接口
	economyClient EconomyClient         // Economy服务gRPC客户端接口
	logger        *zap.Logger           // 结构化日志器（规范7）
}

// Outbox 领域事件Outbox接口，保证事件不丢不重（规范3在domain层声明，此处引用）。
type Outbox interface {
	// Append 追加领域事件到Outbox表
	Append(ctx context.Context, event any) error
}

// EconomyClient Economy服务gRPC客户端接口，用于通知地形变更调整产出。
type EconomyClient interface {
	// AdjustProduction 通知Economy调整指定坐标的产出
	AdjustProduction(ctx context.Context, pos vo.Position, terrainID int32) error
}

// NewChangeTerrainHandler 创建地形变更命令处理器实例。
func NewChangeTerrainHandler(
	mapRepo gamemap.MapRepository,
	configQuery config.ConfigQueryAPI,
	outbox Outbox,
	economyClient EconomyClient,
	logger *zap.Logger,
) *ChangeTerrainHandler {
	return &ChangeTerrainHandler{
		mapRepo:       mapRepo,
		configQuery:   configQuery,
		outbox:        outbox,
		economyClient: economyClient,
		logger:        logger,
	}
}

// Handle 处理地形变更命令。
// 编排：从config查询地形定义校验目标地形存在→校验坐标在地图范围内→
// 调用Map.ChangeTerrain聚合根方法→同事务保存聚合根+写Outbox→gRPC通知Economy调整产出。
func (h *ChangeTerrainHandler) Handle(ctx context.Context, cmd ChangeTerrainCommand) error {
	// 1. 从config查询地形定义校验目标地形存在
	terrain := h.configQuery.GetTerrain(ctx, fmt.Sprintf("%d", cmd.NewTerrainID))
	if terrain == nil {
		h.logger.Warn("地形变更失败，目标地形不存在",
			zap.Int64("map_id", cmd.MapID),
			zap.Int32("terrain_id", cmd.NewTerrainID),
			zap.Int64("operator_id", cmd.OperatorID),
		)
		return fmt.Errorf("地形变更失败，目标地形不存在，terrainID=%d: %w", cmd.NewTerrainID, worlderr.ErrInvalidParams)
	}

	// 2. 加载Map聚合根
	m, err := h.mapRepo.LoadMap(ctx, cmd.MapID)
	if err != nil {
		return fmt.Errorf("地形变更失败，加载地图失败，mapID=%d: %w", cmd.MapID, err)
	}

	// 3. 校验坐标在地图范围内（InBounds由聚合根方法内部校验）
	// 4. 调用聚合根变更地形
	event, err := m.ChangeTerrain(cmd.Position, cmd.NewTerrainID)
	if err != nil {
		return fmt.Errorf("地形变更失败，mapID=%d: %w", cmd.MapID, err)
	}

	// 5. 保存聚合根
	if err := h.mapRepo.SaveMap(ctx, m); err != nil {
		return fmt.Errorf("地形变更失败，保存地图失败，mapID=%d: %w", cmd.MapID, err)
	}

	// 6. 写Outbox投递领域事件
	if err := h.outbox.Append(ctx, event); err != nil {
		return fmt.Errorf("地形变更失败，写Outbox失败，mapID=%d: %w", cmd.MapID, err)
	}

	// 7. gRPC通知Economy调整产出
	if err := h.economyClient.AdjustProduction(ctx, cmd.Position, cmd.NewTerrainID); err != nil {
		h.logger.Warn("地形变更后通知Economy调整产出失败",
			zap.Int64("map_id", cmd.MapID),
			zap.Error(err),
		)
		// 跨服务通知失败不回滚本地事务（最终一致，由事件驱动补偿）
	}

	h.logger.Info("地形变更执行成功",
		zap.Int64("map_id", cmd.MapID),
		zap.Int64("operator_id", cmd.OperatorID),
		zap.Int32("x", cmd.Position.X),
		zap.Int32("y", cmd.Position.Y),
		zap.Int32("old_terrain_id", event.OldTerrainID),
		zap.Int32("new_terrain_id", event.NewTerrainID),
	)

	return nil
}
