// Package main Gateway服务启动入口，组装依赖注入并启动gRPC server。
package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"insectworld/server/gateway/application/command"
	"insectworld/server/gateway/application/query"
	domainconfig "insectworld/server/gateway/domain/config"
	domainsecurity "insectworld/server/gateway/domain/security"
	authaccount "insectworld/server/gateway/infrastructure/auth/account"
	authsecurity "insectworld/server/gateway/infrastructure/auth/security"
	authtoken "insectworld/server/gateway/infrastructure/auth/token"
	"insectworld/server/gateway/infrastructure/config"
	"insectworld/server/gateway/infrastructure/eventbus"
	"insectworld/server/gateway/infrastructure/idgen"
	accountinfra "insectworld/server/gateway/infrastructure/persistence/account"
	auditinfra "insectworld/server/gateway/infrastructure/persistence/audit"
	sessioninfra "insectworld/server/gateway/infrastructure/persistence/session"
	"insectworld/server/gateway/infrastructure/ratelimit"
	"insectworld/server/gateway/infrastructure/websocket"
)

// GatewayStartupConfig Gateway服务启动配置，包含外部依赖连接参数。
type GatewayStartupConfig struct {
	MySQLDSN        string // MySQL连接串
	RedisAddr       string // Redis地址
	RedisPassword   string // Redis密码
	TokenSigningKey string // 令牌签名密钥
	WorkerID        int64  // 雪花算法机器ID
}

// GatewayDeps Gateway服务依赖容器，封装全部已组装的依赖。
type GatewayDeps struct {
	RegisterCmd    *command.RegisterCommand
	LoginCmd       *command.LoginCommand
	LogoutCmd      *command.LogoutCommand
	HeartbeatCmd   *command.HeartbeatCommand
	AuthQuery      *query.AuthenticateQuery
	BanCmd         *command.BanCommand
	UnbanCmd       *command.UnbanCommand
	SessionCleaner *command.SessionTimeoutCleaner
	ConnManager    *websocket.ConnectionManager
	AuditLogger    *auditinfra.AuditLoggerImpl
	AuthCfg        domainconfig.AuthConfig
}

// bootstrap 组装Gateway服务全部依赖注入。
//
// 依赖方向：interfaces → application → domain ← infrastructure（规范3）。
// application不直接import infrastructure（规范3），通过接口+DI组装。
func bootstrap(ctx context.Context, startupCfg GatewayStartupConfig, logger *zap.Logger) (*GatewayDeps, error) {
	db, err := initMySQL(startupCfg.MySQLDSN, logger)
	if err != nil {
		return nil, fmt.Errorf("MySQL初始化失败: %w", err)
	}

	redisClient, err := initRedis(startupCfg.RedisAddr, startupCfg.RedisPassword, logger)
	if err != nil {
		return nil, fmt.Errorf("Redis初始化失败: %w", err)
	}

	cfgLoader := config.NewAuthConfigLoader(logger)
	authCfg := cfgLoader.Get()
	if startupCfg.TokenSigningKey != "" {
		tmpCfg := authCfg
		tmpCfg.TokenSigningKey = startupCfg.TokenSigningKey
		authCfg = tmpCfg
	}

	idGen, err := idgen.NewSnowflakeIDGen(startupCfg.WorkerID)
	if err != nil {
		return nil, fmt.Errorf("雪花ID生成器初始化失败: %w", err)
	}

	hasher := authaccount.NewBcryptHasher(10, logger)

	tokenSigner, err := authtoken.NewTokenSignerImpl([]byte(authCfg.TokenSigningKey), logger)
	if err != nil {
		return nil, fmt.Errorf("令牌签发器初始化失败: %w", err)
	}

	accountRepo := accountinfra.NewAccountRepoMySQL(db, logger)
	sessionRepo := sessioninfra.NewSessionRepoRedis(redisClient, authCfg.SessionTimeoutMs, logger)
	tokenBlacklist := authtoken.NewTokenBlacklistImpl(redisClient, logger)
	failureTracker := authsecurity.NewLoginFailureTrackerImpl(redisClient, authCfg.LoginFailMaxCount, authCfg.LoginLockDurationMs, logger)
	bruteForce := domainsecurity.NewBruteForceProtector(failureTracker, authCfg.LoginFailMaxCount, authCfg.LoginLockDurationMs)

	rateLimiter := ratelimit.NewRateLimiterImpl(map[string]ratelimit.RateConfig{
		"register:ip":   {Rate: authCfg.RegisterRateLimitPerIP, Burst: authCfg.RegisterRateLimitPerIP},
		"login:ip":      {Rate: authCfg.LoginRateLimitPerIP, Burst: authCfg.LoginRateLimitPerIP},
		"login:account": {Rate: authCfg.LoginRateLimitPerAcc, Burst: authCfg.LoginRateLimitPerAcc},
	}, logger)

	auditLogger := auditinfra.NewAuditLoggerImpl(db, logger)
	eventBus := eventbus.NewInMemoryEventBus(logger)
	connManager := websocket.NewConnectionManager(logger)

	registerCmd := command.NewRegisterCommand(accountRepo, rateLimiter, idGen, hasher, auditLogger, authCfg, logger)
	loginCmd := command.NewLoginCommand(accountRepo, sessionRepo, rateLimiter, bruteForce, hasher, tokenSigner, eventBus, auditLogger, connManager, authCfg, logger)
	logoutCmd := command.NewLogoutCommand(tokenSigner, tokenBlacklist, sessionRepo, eventBus, auditLogger, logger)
	heartbeatCmd := command.NewHeartbeatCommand(tokenSigner, sessionRepo, logger)
	authQuery := query.NewAuthenticateQuery(tokenSigner, tokenBlacklist, sessionRepo, logger)
	banCmd := command.NewBanCommand(accountRepo, sessionRepo, tokenBlacklist, eventBus, auditLogger, connManager, authCfg, logger)
	unbanCmd := command.NewUnbanCommand(accountRepo, auditLogger, logger)
	sessionCleaner := command.NewSessionTimeoutCleaner(sessionRepo, eventBus, authCfg, logger)

	return &GatewayDeps{
		RegisterCmd:    registerCmd,
		LoginCmd:       loginCmd,
		LogoutCmd:      logoutCmd,
		HeartbeatCmd:   heartbeatCmd,
		AuthQuery:      authQuery,
		BanCmd:         banCmd,
		UnbanCmd:       unbanCmd,
		SessionCleaner: sessionCleaner,
		ConnManager:    connManager,
		AuditLogger:    auditLogger,
		AuthCfg:        authCfg,
	}, nil
}

// initMySQL 初始化MySQL数据库连接。
func initMySQL(dsn string, logger *zap.Logger) (*sql.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("MySQL DSN未配置")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("MySQL DSN解析失败: %w", err)
	}
	_ = cfg
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("MySQL连接打开失败: %w", err)
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("MySQL连接ping失败: %w", err)
	}
	logger.Info("MySQL连接初始化成功")
	return db, nil
}

// initRedis 初始化Redis客户端连接。
func initRedis(addr, password string, logger *zap.Logger) (*redis.Client, error) {
	if addr == "" {
		return nil, fmt.Errorf("Redis地址未配置")
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("Redis连接ping失败: %w", err)
	}
	logger.Info("Redis连接初始化成功", zap.String("addr", addr))
	return client, nil
}
