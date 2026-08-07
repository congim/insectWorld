// Package main World服务启动入口，组装依赖注入并启动gRPC server。
// 对应design.md 2.7.6节cmd启动入口设计，统一7步组装流程。
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	worldpb "insectworld/server/shared/proto/world"
	"insectworld/server/world/application/query"
	worldgrpc "insectworld/server/world/interfaces/grpc"
)

// WorldServicePort World服务监听端口。
const WorldServicePort = ":50051"

// main 启动World服务。
func main() {
	// 1. 日志初始化
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("World服务启动中")

	// 2. gRPC server初始化
	lis, err := net.Listen("tcp", WorldServicePort)
	if err != nil {
		logger.Fatal("监听失败", zap.Error(err))
	}

	grpcServer := grpc.NewServer()

	// 3. 依赖注入组装
	positionQueryHandler := query.NewPositionQueryHandler(logger)
	mapSnapshotQueryHandler := query.NewMapSnapshotQueryHandler(logger)
	worldHandler := worldgrpc.NewWorldHandler(positionQueryHandler, mapSnapshotQueryHandler, logger)

	// 4. 注册gRPC服务
	worldpb.RegisterWorldServiceServer(grpcServer, worldHandler)

	// 5. 启动gRPC server
	go func() {
		logger.Info("World服务启动", zap.String("addr", WorldServicePort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server启动失败", zap.Error(err))
		}
	}()

	// 6. 信号监听优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("World服务关闭中")
	grpcServer.GracefulStop()

	// 7. 资源清理
	_ = context.Background()
	fmt.Println("World服务已关闭")
}
