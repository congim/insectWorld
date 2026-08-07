// Package grpc World服务接口层gRPC handler，实现WorldServiceServer接口。
// 对接proto/world/world.proto定义的gRPC契约，编排application层command与query。
package grpc

import (
	"context"

	"go.uber.org/zap"

	commonpb "insectworld/server/shared/proto/common"
	worldpb "insectworld/server/shared/proto/world"
	"insectworld/server/world/application/query"
)

// WorldHandler World服务gRPC handler，实现WorldServiceServer接口。
type WorldHandler struct {
	worldpb.UnimplementedWorldServiceServer                                // 嵌入未实现基类，保证前向兼容
	positionQueryHandler                    *query.PositionQueryHandler    // 实体位置查询handler
	mapSnapshotQueryHandler                 *query.MapSnapshotQueryHandler // 地图快照查询handler
	logger                                  *zap.Logger                    // 结构化日志器（规范7）
}

// NewWorldHandler 创建World服务gRPC handler实例。
func NewWorldHandler(
	positionQueryHandler *query.PositionQueryHandler,
	mapSnapshotQueryHandler *query.MapSnapshotQueryHandler,
	logger *zap.Logger,
) *WorldHandler {
	return &WorldHandler{
		positionQueryHandler:    positionQueryHandler,
		mapSnapshotQueryHandler: mapSnapshotQueryHandler,
		logger:                  logger,
	}
}

// GetEntityPosition 查询实体位置。
func (h *WorldHandler) GetEntityPosition(ctx context.Context, req *worldpb.GetPositionRequest) (*worldpb.PositionResponse, error) {
	h.logger.Info("查询实体位置", zap.Int64("entity_id", req.GetEntityId()))

	result, err := h.positionQueryHandler.Handle(ctx, query.EntityPositionQuery{EntityID: req.GetEntityId()})
	if err != nil {
		h.logger.Error("查询实体位置失败", zap.Int64("entity_id", req.GetEntityId()), zap.Error(err))
		return nil, err
	}

	return &worldpb.PositionResponse{
		Position: &commonpb.Position{
			X: result.X,
			Y: result.Y,
		},
	}, nil
}

// GetMapSnapshot 查询地图快照。
func (h *WorldHandler) GetMapSnapshot(ctx context.Context, req *worldpb.MapSnapshotRequest) (*worldpb.MapSnapshotResponse, error) {
	h.logger.Info("查询地图快照", zap.Int64("map_id", req.GetMapId()))

	if err := h.mapSnapshotQueryHandler.Handle(ctx, query.MapSnapshotQuery{MapID: req.GetMapId()}); err != nil {
		h.logger.Error("查询地图快照失败", zap.Int64("map_id", req.GetMapId()), zap.Error(err))
		return nil, err
	}

	return &worldpb.MapSnapshotResponse{}, nil
}

// CheckPassable 校验通行性。
func (h *WorldHandler) CheckPassable(ctx context.Context, req *worldpb.CheckPassableRequest) (*worldpb.CheckPassableResponse, error) {
	h.logger.Info("校验通行性")
	// 通行性校验基于地图格子地形判定，后续接入地图读模型查询
	return &worldpb.CheckPassableResponse{}, nil
}

// GetTerrainInfo 查询地形信息。
func (h *WorldHandler) GetTerrainInfo(ctx context.Context, req *worldpb.TerrainInfoRequest) (*worldpb.TerrainInfoResponse, error) {
	h.logger.Info("查询地形信息")
	// 地形信息查询基于地图格子地形ID映射，后续接入地图读模型查询
	return &worldpb.TerrainInfoResponse{}, nil
}

// GetVisionCells 查询视域格子。
func (h *WorldHandler) GetVisionCells(ctx context.Context, req *worldpb.VisionRequest) (*worldpb.VisionResponse, error) {
	h.logger.Info("查询视域格子")
	// 视域计算基于实体位置与视野范围，后续接入视野服务读模型查询
	return &worldpb.VisionResponse{}, nil
}
