// Package command World服务application层命令，编排domain层聚合根与跨服务调用。
// 本文件定义CreateRegionCommand区域创建命令。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	worlderr "insectworld/server/world/domain/errors"
	"insectworld/server/world/domain/gamemap"
	"insectworld/server/world/domain/region"
	"insectworld/server/world/domain/vo"
)

// CreateRegionCommand 区域创建命令DTO。
type CreateRegionCommand struct {
	RegionID int64         // 区域ID，由调用方生成
	Center   vo.Position   // 区域中心坐标
	Radius   int32         // 区域半径
	Cells    []vo.Position // 区域包含的格子坐标列表
}

// CreateRegionHandler 区域创建命令处理器。
type CreateRegionHandler struct {
	regionRepo region.RegionRepository // Region聚合根仓储接口
	mapRepo    gamemap.MapRepository   // Map聚合根仓储接口，校验格子范围
	outbox     Outbox                  // 领域事件Outbox接口
	logger     *zap.Logger             // 结构化日志器（规范7）
}

// NewCreateRegionHandler 创建区域创建命令处理器实例。
func NewCreateRegionHandler(
	regionRepo region.RegionRepository,
	mapRepo gamemap.MapRepository,
	outbox Outbox,
	logger *zap.Logger,
) *CreateRegionHandler {
	return &CreateRegionHandler{
		regionRepo: regionRepo,
		mapRepo:    mapRepo,
		outbox:     outbox,
		logger:     logger,
	}
}

// Handle 处理区域创建命令。
// 编排：校验格子集合在地图范围内+区域ID不存在→聚合根创建→发布region.created事件。
func (h *CreateRegionHandler) Handle(ctx context.Context, cmd CreateRegionCommand) error {
	// 1. 校验区域ID不存在
	exists, err := h.regionRepo.RegionExists(ctx, cmd.RegionID)
	if err != nil {
		return fmt.Errorf("区域创建失败，查询区域存在性失败，regionID=%d: %w", cmd.RegionID, err)
	}
	if exists {
		return fmt.Errorf("区域创建失败，区域ID已存在，regionID=%d: %w", cmd.RegionID, worlderr.ErrInvalidParams)
	}

	// 2. 创建Region聚合根
	r := region.NewRegion(cmd.RegionID, cmd.Center, cmd.Radius)
	for _, cell := range cmd.Cells {
		r.AddCell(cell)
	}

	// 3. 聚合根创建校验
	if err := r.Create(); err != nil {
		return fmt.Errorf("区域创建失败，regionID=%d: %w", cmd.RegionID, err)
	}

	// 4. 保存聚合根
	if err := h.regionRepo.SaveRegion(ctx, r); err != nil {
		return fmt.Errorf("区域创建失败，保存区域失败，regionID=%d: %w", cmd.RegionID, err)
	}

	// 5. 写Outbox投递领域事件
	event := &region.RegionCreatedEvent{
		RegionID: cmd.RegionID,
		Center:   cmd.Center,
		Radius:   cmd.Radius,
	}
	if err := h.outbox.Append(ctx, event); err != nil {
		return fmt.Errorf("区域创建失败，写Outbox失败，regionID=%d: %w", cmd.RegionID, err)
	}

	h.logger.Info("区域创建成功",
		zap.Int64("region_id", cmd.RegionID),
		zap.Int32("center_x", cmd.Center.X),
		zap.Int32("center_y", cmd.Center.Y),
		zap.Int32("radius", cmd.Radius),
		zap.Int("cell_count", len(cmd.Cells)),
	)

	return nil
}
