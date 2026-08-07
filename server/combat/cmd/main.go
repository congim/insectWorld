// Package main Combat服务启动入口，组装依赖注入并启动gRPC server。
package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	combatgrpc "insectworld/server/combat/interfaces/grpc"
	combatpb "insectworld/server/shared/proto/combat"
)

// CombatServicePort Combat服务监听端口。
const CombatServicePort = ":50052"

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("Combat服务启动中")

	lis, err := net.Listen("tcp", CombatServicePort)
	if err != nil {
		logger.Fatal("监听失败", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	combatHandler := combatgrpc.NewCombatHandler(logger)
	combatpb.RegisterCombatServiceServer(grpcServer, combatHandler)

	go func() {
		logger.Info("Combat服务启动", zap.String("addr", CombatServicePort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server启动失败", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Combat服务关闭中")
	grpcServer.GracefulStop()
	fmt.Println("Combat服务已关闭")
}
