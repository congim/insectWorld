// Package main Gateway服务启动入口，组装依赖注入并启动gRPC server。
package main

import (
	"context"

	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	gatewaygrpc "insectworld/server/gateway/interfaces/grpc"
	"insectworld/server/gateway/interfaces/interceptor"
	wshandler "insectworld/server/gateway/interfaces/websocket"
	adminpb "insectworld/server/shared/proto/admin"
)

// Gateway服务监听端口与默认配置常量（规范1就近归属）。
const (
	GatewayServicePort = ":50056" // Gateway服务gRPC监听端口
	WSPort             = ":50057" // WebSocket认证服务监听端口
	DefaultWorkerID    = 1        // 默认雪花算法机器ID
)

// main Gateway服务启动入口。
//
// 组装依赖注入（bootstrap），启动gRPC server与会话超时清理后台任务，
// 接收SIGINT/SIGTERM信号优雅退出（规范9 goroutine安全）。
func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("Gateway服务启动中")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startupCfg := GatewayStartupConfig{
		MySQLDSN:        os.Getenv("GATEWAY_MYSQL_DSN"),
		RedisAddr:       os.Getenv("GATEWAY_REDIS_ADDR"),
		RedisPassword:   os.Getenv("GATEWAY_REDIS_PASSWORD"),
		TokenSigningKey: os.Getenv("GATEWAY_TOKEN_SIGNING_KEY"),
		WorkerID:        DefaultWorkerID,
	}

	deps, err := bootstrap(ctx, startupCfg, logger)
	if err != nil {
		logger.Fatal("依赖注入组装失败", zap.Error(err))
	}

	wsAuthHandler := wshandler.NewWSAuthHandler(
		deps.RegisterCmd, deps.LoginCmd, deps.LogoutCmd, deps.HeartbeatCmd, deps.AuthQuery, logger,
	)
	authInterceptor := interceptor.NewAuthInterceptor(deps.AuthQuery, logger)
	_ = authInterceptor

	wsServer := wshandler.NewWSServer(wsAuthHandler, logger)
	if err := wsServer.Start(ctx, WSPort); err != nil {
		logger.Fatal("WebSocket服务启动失败", zap.Error(err))
	}

	playerAdminAdapter := gatewaygrpc.NewPlayerAdminAdapter(deps.BanCmd, deps.UnbanCmd, "system")
	adminHandler := gatewaygrpc.NewAdminHandler(nil, nil, playerAdminAdapter, logger)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		deps.SessionCleaner.Run(ctx)
	}()

	lis, err := net.Listen("tcp", GatewayServicePort)
	if err != nil {
		logger.Fatal("监听失败", zap.Error(err))
	}
	grpcServer := grpc.NewServer()
	adminpb.RegisterAdminServiceServer(grpcServer, adminHandler)

	go func() {
		logger.Info("Gateway服务启动", zap.String("addr", GatewayServicePort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server启动失败", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Gateway服务关闭中")

	cancel()
	grpcServer.GracefulStop()
	deps.AuditLogger.Close()
	wg.Wait()
	logger.Info("Gateway服务已关闭")
}
