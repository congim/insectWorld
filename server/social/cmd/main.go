// Package main Social服务启动入口，组装依赖注入并启动gRPC server。
package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	socialpb "insectworld/server/shared/proto/social"
	socialgrpc "insectworld/server/social/interfaces/grpc"
)

// SocialServicePort Social服务监听端口。
const SocialServicePort = ":50054"

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("Social服务启动中")

	lis, err := net.Listen("tcp", SocialServicePort)
	if err != nil {
		logger.Fatal("监听失败", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	socialHandler := socialgrpc.NewSocialHandler(logger)
	socialpb.RegisterSocialServiceServer(grpcServer, socialHandler)

	go func() {
		logger.Info("Social服务启动", zap.String("addr", SocialServicePort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server启动失败", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Social服务关闭中")
	grpcServer.GracefulStop()
	fmt.Println("Social服务已关闭")
}
