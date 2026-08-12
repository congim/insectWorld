// Package main Config服务启动入口，按DDD依赖注入组装各层组件。
//
// 启动流程：加载flag→初始化logger→初始化数据库(可选)→初始化etcd客户端→
// 初始化配置编译器/校验器/热更watcher→初始化Repository实现→
// 初始化application层command/query→初始化interfaces层gRPC handler→
// 启动热更watcher goroutine→启动gRPC服务。
// 热更watcher goroutine有context cancel退出机制（规范9 goroutine安全）。
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"insectworld/server/config/application/command"
	"insectworld/server/config/application/query"
	"insectworld/server/config/infrastructure/audit"
	"insectworld/server/config/infrastructure/etcd"
	"insectworld/server/config/infrastructure/versionstore"
	configgrpc "insectworld/server/config/interfaces/grpc"

	"insectworld/server/shared/pkg/config"
	configpb "insectworld/server/shared/proto/config"
)

// Config服务监听端口与默认配置常量。
const (
	configServicePort = ":50057" // gRPC监听端口
)

func main() {
	var mysqlDSN string
	flag.StringVar(&mysqlDSN, "mysql-dsn", "", "MySQL DSN，未设置时版本历史与审计日志降级")
	flag.Parse()

	logger, _ := zap.NewProduction()
	defer logger.Sync()
	logger.Info("Config服务启动中")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 初始化数据库（可选，DSN未设置时跳过，版本历史与审计日志降级）
	var db *sql.DB
	if mysqlDSN != "" {
		var err error
		db, err = sql.Open("mysql", mysqlDSN)
		if err != nil {
			logger.Fatal("MySQL连接池初始化失败", zap.Error(err))
		}
		if err := db.PingContext(ctx); err != nil {
			logger.Fatal("MySQL连接测试失败", zap.Error(err))
		}
		defer db.Close()
		logger.Info("MySQL连接池初始化成功")
	}

	// 2. 初始化etcd客户端（内存版，Put时产生ConfigChangeEvent通知热更watcher）
	etcdClient := etcd.NewClient(nil, 0, logger)

	// 3. 初始化配置编译器/校验器/热更watcher（共享内核config模块）
	registry := config.NewExtensionRegistry(logger)
	validator := config.NewValidator(logger)
	compiler := config.NewConfigCompiler(registry, validator, logger)
	hotReloader := config.NewConfigHotReloader(compiler, validator, logger)

	// 4. 启动热更watcher goroutine，消费etcd watch channel执行编译+校验+原子替换+回调
	go hotReloader.Run(ctx, etcdClient.WatchChan())
	logger.Info("配置热更watcher已启动")

	// 5. 初始化Repository实现（infrastructure层，db可为nil时降级）
	versionStore := versionstore.NewVersionStore(db, logger)
	auditRepo := audit.NewRepository(db, logger)

	// 6. 初始化application层command/query handler（注入domain层接口，规范3 DI）
	cmdHandler := command.NewConfigCommandHandler(etcdClient, versionStore, auditRepo, logger)
	queryHandler := query.NewConfigVersionQueryHandler(versionStore, logger)

	// 7. 初始化interfaces层gRPC handler
	configHandler := configgrpc.NewConfigHandler(cmdHandler, queryHandler, logger)

	// 8. 启动gRPC server
	lis, err := net.Listen("tcp", configServicePort)
	if err != nil {
		logger.Fatal("监听失败", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	configpb.RegisterConfigServiceServer(grpcServer, configHandler)

	go func() {
		logger.Info("Config服务启动", zap.String("addr", configServicePort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server启动失败", zap.Error(err))
		}
	}()

	// 9. 等待退出信号，优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Config服务关闭中")
	cancel() // 停止热更watcher goroutine
	grpcServer.GracefulStop()
	fmt.Println("Config服务已关闭")
}
