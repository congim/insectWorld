// Package main 用户认证接口测试端启动入口，组装依赖并启动HTTP服务器。
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"

	"insectworld/tests/authtest/internal/config"
	"insectworld/tests/authtest/internal/e2e"
	"insectworld/tests/authtest/internal/envinit"
	"insectworld/tests/authtest/internal/logutil"
	"insectworld/tests/authtest/internal/sutmgr"
	"insectworld/tests/authtest/internal/webserver"
	"insectworld/tests/authtest/internal/wsclient"
)

func main() {
	logger, err := logutil.NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "日志初始化失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	cfg, err := config.LoadConfig("testdata/config.json", logger)
	if err != nil {
		logger.Error("配置加载失败", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("测试端启动",
		zap.String("listenAddr", cfg.WebListenAddr),
		zap.String("testDatabase", cfg.TestDatabase),
		zap.String("gatewayWSURL", cfg.GatewayWSURL),
	)

	db, err := sql.Open("mysql", cfg.MySQLDSNNoDB())
	if err != nil {
		logger.Error("MySQL连接初始化失败", zap.Error(err))
		os.Exit(1)
	}
	defer db.Close()

	ddlLoader := envinit.NewDDLLoader(logger)
	guard := envinit.NewLocalMySQLGuard()
	initializer := envinit.NewDatabaseInitializer(db, ddlLoader, guard, cfg.TestDatabase, cfg.DDLScriptPath, logger)

	envBuilder := sutmgr.NewEnvironmentBuilder()
	sutMgr := sutmgr.NewSUTManager(envBuilder, cfg.GatewayDir, logger)

	wsClient := wsclient.NewAuthWSClient(cfg.GatewayWSURL, logger)

	assertion := e2e.NewAssertion(logger)
	e2eRunner := e2e.NewE2ERunner(wsClient, assertion, logger)
	failureRunner := e2e.NewFailureCaseRunner(wsClient, assertion, logger)

	envH := webserver.NewEnvHandlers(initializer, logger)
	sutH := webserver.NewSUTHandlers(sutMgr, cfg, logger)
	authH := webserver.NewAuthHandlers(wsClient, logger)
	e2eH := webserver.NewE2EHandlers(e2eRunner, failureRunner, logger)

	webSrv := webserver.NewWebServer(cfg.WebListenAddr, envH, sutH, authH, e2eH, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := webSrv.Start(ctx); err != nil {
		logger.Error("Web服务器启动失败", zap.Error(err))
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("收到退出信号，正在关闭")
	cancel()
}
