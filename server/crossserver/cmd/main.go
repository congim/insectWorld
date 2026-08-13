// Package main CrossServer服务启动入口，组装依赖注入并启动gRPC server。
// 对应design.md 2.1.4.2节CrossServer上下文落地，管理跨服节点、活动与合服。
package main

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"insectworld/server/crossserver/application/query"
)

// CrossServerServicePort CrossServer服务监听端口。
const CrossServerServicePort = ":50091"

// main 启动CrossServer服务。
func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("CrossServer服务启动中")

	// TODO 后续接入gRPC server、Repository实现、NATS跨服通信、节点心跳监听
	var nodeReadModel query.NodeReadModel // TODO 待infrastructure层实现后注入
	nodeListQueryHandler := query.NewNodeListQueryHandler(nodeReadModel, logger)
	_ = nodeListQueryHandler

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("CrossServer服务关闭中")
}
