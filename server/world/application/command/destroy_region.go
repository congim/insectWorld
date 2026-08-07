// Package command World服务application层命令，编排domain层聚合根与跨服务调用。
// 本文件定义DestroyRegionCommand区域销毁命令。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/world/domain/region"
)

// DestroyRegionCommand 区域销毁命令DTO。
type DestroyRegionCommand struct {
	RegionID int64 // 区域ID
}

// DestroyRegionHandler 区域销毁命令处理器。
type DestroyRegionHandler struct {
	regionRepo    region.RegionRepository // Region聚合根仓储接口
	outbox        Outbox                  // 领域事件Outbox接口
	entityChecker EntityChecker           // 实体检查接口，查询区域内是否有活跃实体
	logger        *zap.Logger             // 结构化日志器（规范7）
}

// EntityChecker 实体检查接口，查询区域内是否有活跃实体。
// 由infrastructure层实现，通过gRPC查询World Service内部实体状态。
type EntityChecker interface {
	// HasActiveEntities 检查区域内是否有活跃实体
	HasActiveEntities(ctx context.Context, regionID int64) (bool, error)
}

// NewDestroyRegionHandler 创建区域销毁命令处理器实例。
func NewDestroyRegionHandler(
	regionRepo region.RegionRepository,
	outbox Outbox,
	entityChecker EntityChecker,
	logger *zap.Logger,
) *DestroyRegionHandler {
	return &DestroyRegionHandler{
		regionRepo:    regionRepo,
		outbox:        outbox,
		entityChecker: entityChecker,
		logger:        logger,
	}
}

// Handle 处理区域销毁命令。
// 编排：校验区域内无活跃实体（本服务查询）→聚合根销毁→发布region.destroyed事件。
func (h *DestroyRegionHandler) Handle(ctx context.Context, cmd DestroyRegionCommand) error {
	// 1. 加载Region聚合根
	r, err := h.regionRepo.LoadRegion(ctx, cmd.RegionID)
	if err != nil {
		return fmt.Errorf("区域销毁失败，加载区域失败，regionID=%d: %w", cmd.RegionID, err)
	}

	// 2. 查询区域内是否有活跃实体
	hasActive, err := h.entityChecker.HasActiveEntities(ctx, cmd.RegionID)
	if err != nil {
		return fmt.Errorf("区域销毁失败，查询活跃实体失败，regionID=%d: %w", cmd.RegionID, err)
	}

	// 3. 聚合根销毁（校验无活跃实体）
	event, err := r.Destroy(hasActive)
	if err != nil {
		h.logger.Warn("区域销毁失败，区域内有活跃实体",
			zap.Int64("region_id", cmd.RegionID),
			zap.Error(err),
		)
		return fmt.Errorf("区域销毁失败，regionID=%d: %w", cmd.RegionID, err)
	}

	// 4. 保存聚合根
	if err := h.regionRepo.SaveRegion(ctx, r); err != nil {
		return fmt.Errorf("区域销毁失败，保存区域失败，regionID=%d: %w", cmd.RegionID, err)
	}

	// 5. 写Outbox投递领域事件
	if err := h.outbox.Append(ctx, event); err != nil {
		return fmt.Errorf("区域销毁失败，写Outbox失败，regionID=%d: %w", cmd.RegionID, err)
	}

	h.logger.Info("区域销毁成功",
		zap.Int64("region_id", cmd.RegionID),
	)

	return nil
}
