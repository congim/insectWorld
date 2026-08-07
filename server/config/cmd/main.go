// Package main Config服务启动入口，组装依赖注入并启动gRPC server。
package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	configgrpc "insectworld/server/config/interfaces/grpc"
	configpb "insectworld/server/shared/proto/config"
)

// ConfigServicePort Config服务监听端口。
const ConfigServicePort = ":50057"

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("Config服务启动中")

	lis, err := net.Listen("tcp", ConfigServicePort)
	if err != nil {
		logger.Fatal("监听失败", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	configHandler := configgrpc.NewConfigHandler(logger)
	configpb.RegisterConfigServiceServer(grpcServer, configHandler)

	go func() {
		logger.Info("Config服务启动", zap.String("addr", ConfigServicePort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server启动失败", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Config服务关闭中")
	grpcServer.GracefulStop()
	fmt.Println("Config服务已关闭")
}
