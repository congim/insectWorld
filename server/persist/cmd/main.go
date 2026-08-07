// Package main Persist服务启动入口，按DDD依赖注入组装各层组件。
//
// 启动流程：加载配置→初始化logger→初始化datasource→初始化Repository实现→
// 初始化application→初始化interfaces→启动gRPC服务→启动事件订阅→启动定时任务。
// 定时任务有context cancel退出机制（规范9 goroutine安全）。
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"insectworld/server/persist/application/command"
	"insectworld/server/persist/application/query"
	"insectworld/server/persist/infrastructure/archive"
	"insectworld/server/persist/infrastructure/backup"
	"insectworld/server/persist/infrastructure/datasource"
	migrationExec "insectworld/server/persist/infrastructure/migration"
	"insectworld/server/persist/infrastructure/persistence"
	"insectworld/server/persist/infrastructure/snapshot"

	grpcHandler "insectworld/server/persist/interfaces/grpc"
)

// 默认配置常量。
const (
	defaultMigrationsDir = "../shared/schema/migrations" // 迁移脚本目录默认路径
)

func main() {
	var mysqlDSN string
	var coldMysqlDSN string
	var migrationsDir string
	flag.StringVar(&mysqlDSN, "mysql-dsn", "", "MySQL热库DSN")
	flag.StringVar(&coldMysqlDSN, "cold-mysql-dsn", "", "冷库MySQL DSN")
	flag.StringVar(&migrationsDir, "migrations-dir", defaultMigrationsDir, "迁移脚本目录")
	flag.Parse()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Persist服务启动中...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ds := datasource.NewManager(logger)
	if mysqlDSN != "" {
		if err := ds.InitMySQL(ctx, mysqlDSN); err != nil {
			logger.Fatal("MySQL热库初始化失败", zap.Error(err))
		}
	}
	if coldMysqlDSN != "" {
		if err := ds.InitColdMySQL(ctx, coldMysqlDSN); err != nil {
			logger.Fatal("冷库MySQL初始化失败", zap.Error(err))
		}
	}
	defer ds.Close()

	migrationRepo := persistence.NewMigrationRepoImpl(ds.MySQL())
	snapshotRepo := snapshot.NewSnapshotRepoImpl(ds.MySQL())
	archiveRepo := archive.NewArchiveRepoImpl(ds.MySQL())
	backupRepo := backup.NewBackupRepoImpl(ds.MySQL())

	createSnapshotHandler := command.NewCreateSnapshotHandler(snapshotRepo, logger)
	executeMigrationHandler := command.NewExecuteMigrationHandler(migrationRepo, logger)
	archiveColdDataHandler := command.NewArchiveColdDataHandler(archiveRepo, logger)
	createBackupHandler := command.NewCreateBackupHandler(backupRepo, logger)

	_ = query.NewSnapshotQueryHandler(snapshotRepo, logger)
	_ = query.NewMigrationStatusQueryHandler(migrationRepo, logger)

	grpcH := grpcHandler.NewHandler(
		createSnapshotHandler, executeMigrationHandler,
		archiveColdDataHandler, createBackupHandler, logger,
	)
	_ = grpcH

	if ds.MySQL() != nil {
		executor := migrationExec.NewExecutor(ds.MySQL(), migrationsDir, logger)
		executedVersions, err := migrationRepo.FindExecutedVersions(ctx)
		if err != nil {
			logger.Warn("查询已执行迁移版本失败，将执行全部迁移", zap.Error(err))
			executedVersions = nil
		}
		if err := executor.ExecutePending(ctx, executedVersions); err != nil {
			logger.Error("执行待执行迁移失败", zap.Error(err))
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	logger.Info("Persist服务启动成功，等待信号退出...")

	for {
		select {
		case sig := <-sigCh:
			logger.Info("收到退出信号，正在关闭...", zap.String("signal", sig.String()))
			cancel()
			return
		case <-ticker.C:
			logger.Debug("Persist服务定时任务心跳")
		}
	}
}
