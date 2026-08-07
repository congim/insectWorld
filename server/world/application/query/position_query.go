// Package query World服务application层读侧查询，CQRS读模型查询handler。
// 对接interfaces/grpc层的查询请求，从读模型查询数据（不经过domain聚合根）。
package query

import (
	"context"

	"go.uber.org/zap"
)

// EntityPositionQuery 实体位置查询DTO。
type EntityPositionQuery struct {
	EntityID int64 // 实体ID（规范8用int64）
}

// EntityPositionResult 实体位置查询结果DTO。
type EntityPositionResult struct {
	X int32 // X坐标（规范8用int32）
	Y int32 // Y坐标（规范8用int32）
}

// MapSnapshotQuery 地图快照查询DTO。
type MapSnapshotQuery struct {
	MapID int64 // 地图ID
}

// PositionQueryHandler 位置查询handler，CQRS读侧。
type PositionQueryHandler struct {
	logger *zap.Logger // 结构化日志器（规范7）
}

// NewPositionQueryHandler 创建位置查询handler实例。
func NewPositionQueryHandler(logger *zap.Logger) *PositionQueryHandler {
	return &PositionQueryHandler{logger: logger}
}

// Handle 处理实体位置查询。
func (h *PositionQueryHandler) Handle(ctx context.Context, q EntityPositionQuery) (*EntityPositionResult, error) {
	h.logger.Debug("查询实体位置", zap.Int64("entity_id", q.EntityID))
	// TODO 后续从读模型查询实体?体位置
	return &EntityPositionResult{}, nil
}

// MapSnapshotQueryHandler 地图快照查询handler，CQRS读侧。
type MapSnapshotQueryHandler struct {
	logger *zap.Logger // 结构化日志器（规范7）
}

// NewMapSnapshotQueryHandler 创建地图快照查询handler实例。
func NewMapSnapshotQueryHandler(logger *zap.Logger) *MapSnapshotQueryHandler {
	return &MapSnapshotQueryHandler{logger: logger}
}

// Handle 处理地图快照查询。
func (h *MapSnapshotQueryHandler) Handle(ctx context.Context, q MapSnapshotQuery) error {
	h.logger.Debug("查询地图快照", zap.Int64("map_id", q.MapID))
	// TODO 后续从读模型查询地图快照
	return nil
}
