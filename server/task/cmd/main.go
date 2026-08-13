// Package main Task服务启动入口，组装依赖注入并启动gRPC server。
// 对应design.md 2.1.4.2节Task上下文落地，按玩家ID分片。
package main

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"insectworld/server/task/application/query"
)

// TaskServicePort Task服务监听端口。
const TaskServicePort = ":50071"

// main 启动Task服务。
func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Task服务启动中")

	// TODO 后续接入gRPC server、Repository实现、ConfigQueryAPI、EventBus、NATS事件订阅
	var taskReadModel query.TaskReadModel // TODO 待infrastructure层实现后注入
	taskListQueryHandler := query.NewTaskListQueryHandler(taskReadModel, logger)
	_ = taskListQueryHandler

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Task服务关闭中")
}
