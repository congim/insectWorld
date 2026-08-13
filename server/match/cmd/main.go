// Package main Match服务启动入口，组装依赖注入并启动gRPC server。
// 对应design.md 2.1.4.2节Match上下文落地，跨服匹配按匹配池分片。
package main

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"insectworld/server/match/application/query"
)

// MatchServicePort Match服务监听端口。
const MatchServicePort = ":50081"

// main 启动Match服务。
func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Match服务启动中")

	// TODO 后续接入gRPC server、Repository实现、匹配池引擎、NATS跨服通信
	var rankReadModel query.RankReadModel // TODO 待infrastructure层实现后注入
	rankListQueryHandler := query.NewRankListQueryHandler(rankReadModel, logger)
	_ = rankListQueryHandler

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Match服务关闭中")
}
