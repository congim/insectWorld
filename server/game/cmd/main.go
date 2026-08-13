package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"go.uber.org/zap"
)

const (
	defaultEngineVersion = "0.1.0"  // 当前开发期引擎兼容版本
	defaultWorkerID      = int64(2) // 默认避开Gateway使用的节点编号1
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	exitCode := 0
	if err := run(logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("game进程退出", zap.Error(err))
		exitCode = 1
	}
	_ = logger.Sync()
	os.Exit(exitCode)
}

func run(logger *zap.Logger) error {
	config, err := startupConfigFromEnv()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	runtime, err := Bootstrap(ctx, config, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err := runtime.Close(); err != nil {
			logger.Error("关闭game数据库连接失败", zap.Error(err))
		}
	}()
	logger.Info("game进程启动", zap.String("gamepack_id", runtime.packID), zap.String("config_version", runtime.version), zap.Int64("worker_id", config.WorkerID))
	err = runtime.Run(ctx)
	logger.Info("game进程停止", zap.String("gamepack_id", runtime.packID), zap.String("config_version", runtime.version))
	return err
}

func startupConfigFromEnv() (StartupConfig, error) {
	workerID := defaultWorkerID
	if raw := os.Getenv("GAME_WORKER_ID"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return StartupConfig{}, fmt.Errorf("解析GAME_WORKER_ID失败: %w", err)
		}
		workerID = value
	}
	engineVersion := os.Getenv("GAME_ENGINE_VERSION")
	if engineVersion == "" {
		engineVersion = defaultEngineVersion
	}
	return StartupConfig{
		MySQLDSN:      os.Getenv("GAME_MYSQL_DSN"),
		GamePackRoot:  os.Getenv("GAME_PACK_ROOT"),
		EngineVersion: engineVersion,
		WorkerID:      workerID,
	}, nil
}
