// Package main 组装模块化单体game进程及其生命周期。
package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"

	economyapp "insectworld/server/economy/application/resourceaccount"
	economypersistence "insectworld/server/economy/infrastructure/persistence"
	gamecommand "insectworld/server/game/application/command"
	gamecatalog "insectworld/server/game/infrastructure/catalog"
	gameidentity "insectworld/server/game/infrastructure/identity"
	gamepersistence "insectworld/server/game/infrastructure/persistence"
	gameevent "insectworld/server/game/interfaces/event"
	sharedeventbus "insectworld/server/shared/infrastructure/eventbus"
	"insectworld/server/shared/pkg/eventbus/publisher"
	"insectworld/server/shared/pkg/gamepack"
)

const (
	defaultMaxOpenConnections = 50               // 默认数据库最大连接数，后续由压测校准
	defaultMaxIdleConnections = 10               // 默认数据库最大空闲连接数
	defaultConnectionLifetime = 30 * time.Minute // 默认连接生命周期，避免长期持有失效连接
	defaultBatchSize          = 100              // 默认单轮Outbox领取上限
	defaultPollInterval       = time.Second      // 默认无事件时轮询间隔
	defaultLeaseDuration      = 30 * time.Second // 默认投递租约，应高于本地消费者正常耗时
	defaultBaseRetryDelay     = time.Second      // 默认首次失败重试间隔
	defaultMaxRetryDelay      = time.Minute      // 默认失败重试间隔上限
)

// StartupConfig 是game进程启动配置。
type StartupConfig struct {
	MySQLDSN      string // MySQL连接串，只从运行环境注入
	GamePackRoot  string // 当前实例绑定的游戏包根目录
	EngineVersion string // 当前引擎语义版本，用于校验游戏包兼容范围
	WorkerID      int64  // Growth聚合ID生成节点编号，范围0到1023
}

// Runtime 保存game进程已装配的应用服务和后台任务。
type Runtime struct {
	Growth    *gamecommand.Service // Growth写命令应用服务，供后续协议适配层使用
	publisher *publisher.Publisher // 注册事件Outbox发布器
	db        *sql.DB              // 进程共享MySQL连接池
	packID    string               // 当前装载的游戏包稳定ID
	version   string               // 当前装载的游戏包语义版本
}

// Bootstrap 校验外部配置、连接MySQL并装配game进程。
func Bootstrap(ctx context.Context, config StartupConfig, logger *zap.Logger) (*Runtime, error) {
	if err := validateStartupConfig(config); err != nil {
		return nil, err
	}
	pack, err := gamepack.LoadAndCompile(config.GamePackRoot, config.EngineVersion)
	if err != nil {
		return nil, fmt.Errorf("加载game进程游戏包失败: %w", err)
	}
	db, err := openMySQL(ctx, config.MySQLDSN)
	if err != nil {
		return nil, err
	}
	runtime, err := buildRuntime(ctx, db, pack, config.WorkerID, logger)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return runtime, nil
}

func validateStartupConfig(config StartupConfig) error {
	if config.MySQLDSN == "" || config.GamePackRoot == "" || config.EngineVersion == "" {
		return fmt.Errorf("game启动配置不完整")
	}
	if config.WorkerID < 0 || config.WorkerID > 1023 {
		return fmt.Errorf("game机器ID超出范围，workerID=%d", config.WorkerID)
	}
	return nil
}

func openMySQL(ctx context.Context, dsn string) (*sql.DB, error) {
	if _, err := mysql.ParseDSN(dsn); err != nil {
		return nil, fmt.Errorf("解析game MySQL DSN失败: %w", err)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开game MySQL连接失败: %w", err)
	}
	db.SetMaxOpenConns(defaultMaxOpenConnections)
	db.SetMaxIdleConns(defaultMaxIdleConnections)
	db.SetConnMaxLifetime(defaultConnectionLifetime)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("连接game MySQL失败: %w", err)
	}
	return db, nil
}

func buildRuntime(ctx context.Context, db *sql.DB, pack *gamepack.CompiledPack, workerID int64, logger *zap.Logger) (*Runtime, error) {
	if db == nil || pack == nil {
		return nil, fmt.Errorf("game运行时依赖不能为空")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	catalogReader, err := gamecatalog.NewGamePackReader(pack)
	if err != nil {
		return nil, err
	}
	idGenerator, err := gameidentity.NewSnowflake(workerID)
	if err != nil {
		return nil, err
	}
	resourceRepository := economypersistence.NewResourceAccountRepository(db, logger)
	resourceService := economyapp.NewService(resourceRepository)
	growth := gamecommand.NewService(
		gamepersistence.NewPlayerRepository(db, logger),
		gamepersistence.NewBuildingRepository(db, logger),
		gamepersistence.NewTrainingRepository(db, logger),
		gamepersistence.NewUnitRoster(db, logger),
		catalogReader,
		resourceService,
		idGenerator,
		logger,
	)
	localBus := sharedeventbus.NewLocalBus(logger)
	registrationHandler := gameevent.NewPlayerRegisteredHandler(growth)
	if err := localBus.Subscribe(ctx, gameevent.EventTypePlayerRegistered, registrationHandler.Handle); err != nil {
		return nil, fmt.Errorf("注册玩家创建事件消费者失败: %w", err)
	}
	outboxPublisher, err := publisher.New(
		sharedeventbus.NewOutboxRepository(db),
		localBus,
		publisher.Config{
			EventTypes:     []string{gameevent.EventTypePlayerRegistered},
			BatchSize:      defaultBatchSize,
			PollInterval:   defaultPollInterval,
			LeaseDuration:  defaultLeaseDuration,
			BaseRetryDelay: defaultBaseRetryDelay,
			MaxRetryDelay:  defaultMaxRetryDelay,
		},
		logger,
	)
	if err != nil {
		return nil, err
	}
	return &Runtime{Growth: growth, publisher: outboxPublisher, db: db, packID: pack.Manifest.ID, version: pack.Manifest.Version}, nil
}

// Run 运行game后台任务，直到context取消。
func (r *Runtime) Run(ctx context.Context) error {
	return r.publisher.Run(ctx)
}

// Close 关闭game进程拥有的数据库连接池。
func (r *Runtime) Close() error {
	return r.db.Close()
}
