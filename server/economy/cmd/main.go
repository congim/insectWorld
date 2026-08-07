// Package main Economy服务启动入口，组装依赖注入并启动gRPC server。
package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	economygrpc "insectworld/server/economy/interfaces/grpc"
	economypb "insectworld/server/shared/proto/economy"
)

// EconomyServicePort Economy服务监听端口。
const EconomyServicePort = ":50053"

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("Economy服务启动中")

	lis, err := net.Listen("tcp", EconomyServicePort)
	if err != nil {
		logger.Fatal("监听失败", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	economyHandler := economygrpc.NewEconomyHandler(logger)
	economypb.RegisterEconomyServiceServer(grpcServer, economyHandler)

	go func() {
		logger.Info("Economy服务启动", zap.String("addr", EconomyServicePort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server启动失败", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Economy服务关闭中")
	grpcServer.GracefulStop()
	fmt.Println("Economy服务已关闭")
}
