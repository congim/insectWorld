// Package grpc Inventory服务接口层gRPC handler。
// 对接proto/inventory/inventory.proto定义的gRPC契约，编排application层command与query。
// TODO 后续protoc生成inventory.pb.go后接入，当前提供handler骨架。
package grpc

import (
	"go.uber.org/zap"

	"insectworld/server/inventory/application/query"
)

// InventoryHandler Inventory服务gRPC handler。
// TODO 后续嵌入inventorypb.UnimplementedInventoryServiceServer，待proto生成后补充。
type InventoryHandler struct {
	inventoryQueryHandler *query.InventoryQueryHandler // 背包查询handler
	logger                *zap.Logger                  // 结构化日志器（规范7）
}

// NewInventoryHandler 创建Inventory服务gRPC handler实例。
func NewInventoryHandler(
	inventoryQueryHandler *query.InventoryQueryHandler,
	logger *zap.Logger,
) *InventoryHandler {
	return &InventoryHandler{
		inventoryQueryHandler: inventoryQueryHandler,
		logger:                logger,
	}
}
