// Package main Operation服务启动入口，组装依赖注入并启动gRPC server。
package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	operationgrpc "insectworld/server/operation/interfaces/grpc"
	operationpb "insectworld/server/shared/proto/operation"
)

// OperationServicePort Operation服务监听端口。
const OperationServicePort = ":50055"

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("Operation服务启动中")

	lis, err := net.Listen("tcp", OperationServicePort)
	if err != nil {
		logger.Fatal("监听失败", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	operationHandler := operationgrpc.NewOperationHandler(logger)
	operationpb.RegisterOperationServiceServer(grpcServer, operationHandler)

	go func() {
		logger.Info("Operation服务启动", zap.String("addr", OperationServicePort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server启动失败", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Operation服务关闭中")
	grpcServer.GracefulStop()
	fmt.Println("Operation服务已关闭")
}
