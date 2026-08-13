// Package main Inventory服务启动入口，组装依赖注入并启动gRPC server。
// 对应design.md 2.1.4.2节Inventory上下文落地，按玩家ID分片。
package main

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"insectworld/server/inventory/application/query"
	"insectworld/server/inventory/domain/inventory"
)

// InventoryServicePort Inventory服务监听端口。
const InventoryServicePort = ":50061"

// main 启动Inventory服务。
func main() {
	// 1. 日志初始化
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Inventory服务启动中")

	// 2. 依赖注入组装
	// TODO 后续接入gRPC server、Repository实现、ConfigQueryAPI、EventBus
	// inventoryRepo := persistence.NewInventoryRepoImpl(db, logger)
	var inventoryRepo inventory.InventoryRepository // TODO 待infrastructure层实现后注入
	inventoryQueryHandler := query.NewInventoryQueryHandler(inventoryRepo, logger)

	_ = inventoryQueryHandler // 待gRPC handler接入后使用

	// 3. 信号监听优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Inventory服务关闭中")
}
