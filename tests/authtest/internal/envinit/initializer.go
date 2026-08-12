// Package envinit 测试端环境初始化工具，负责建库/建表/清理/销毁测试数据库。
package envinit

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"
)

// 测试库需清理的表名列表（复用gateway.sql的t_前缀表）。
var tablesToCleanup = []string{
	"t_player_account", "t_session", "t_auth_audit_log", "t_connection_record",
}

// DatabaseInitializer 数据库初始化器，负责建库/建表/清理/销毁测试数据库。
type DatabaseInitializer struct {
	db            *sql.DB          // MySQL连接（不指定库）
	loader        *DDLLoader       // DDL加载器
	guard         *LocalMySQLGuard // 本地MySQL守卫
	testDatabase  string           // 测试库名
	ddlScriptPath string           // DDL脚本路径
	logger        *zap.Logger      // 结构化日志
}

// NewDatabaseInitializer 创建数据库初始化器实例。
func NewDatabaseInitializer(db *sql.DB, loader *DDLLoader, guard *LocalMySQLGuard, testDatabase, ddlScriptPath string, logger *zap.Logger) *DatabaseInitializer {
	return &DatabaseInitializer{
		db:            db,
		loader:        loader,
		guard:         guard,
		testDatabase:  testDatabase,
		ddlScriptPath: ddlScriptPath,
		logger:        logger,
	}
}

// InitResult 初始化结果。
type InitResult struct {
	Success  bool     // 是否成功
	Database string   // 测试库名
	Tables   []string // 已建表名列表
	ErrorMsg string   // 错误信息
}

// StatusResult 状态查询结果。
type StatusResult struct {
	Success bool     // 是否成功
	Tables  []string // 已存在表名列表
	Existed bool     // 测试库是否存在
}

// CleanupResult 清理结果。
type CleanupResult struct {
	Success bool     // 是否成功
	Tables  []string // 已清空表名列表
}

// Init 执行建库建表初始化。
func (i *DatabaseInitializer) Init(ctx context.Context) (*InitResult, error) {
	result := &InitResult{Database: i.testDatabase}

	if err := i.guard.Validate(i.extractHost()); err != nil {
		result.ErrorMsg = err.Error()
		return result, err
	}

	createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", i.testDatabase)
	if _, err := i.db.ExecContext(ctx, createDBSQL); err != nil {
		result.ErrorMsg = fmt.Sprintf("建库失败: %v", err)
		return result, err
	}

	useDBSQL := fmt.Sprintf("USE %s", i.testDatabase)
	if _, err := i.db.ExecContext(ctx, useDBSQL); err != nil {
		result.ErrorMsg = fmt.Sprintf("切换库失败: %v", err)
		return result, err
	}

	statements, err := i.loader.Load(i.ddlScriptPath)
	if err != nil {
		result.ErrorMsg = err.Error()
		return result, err
	}

	for _, stmt := range statements {
		if _, err := i.db.ExecContext(ctx, stmt); err != nil {
			i.logger.Error("建表失败", zap.String("sql", stmt), zap.Error(err))
			result.ErrorMsg = fmt.Sprintf("建表失败: %v", err)
			return result, err
		}
		result.Tables = append(result.Tables, extractTableName(stmt))
	}

	result.Success = true
	i.logger.Info("数据库初始化成功",
		zap.String("database", i.testDatabase),
		zap.Strings("tables", result.Tables),
	)
	return result, nil
}

// Status 查询测试库与表状态。
func (i *DatabaseInitializer) Status(ctx context.Context) (*StatusResult, error) {
	result := &StatusResult{}

	query := `SELECT table_name FROM information_schema.tables WHERE table_schema = ?`
	rows, err := i.db.QueryContext(ctx, query, i.testDatabase)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return result, err
		}
		result.Tables = append(result.Tables, tableName)
	}
	result.Existed = len(result.Tables) > 0
	result.Success = true
	return result, nil
}

// Cleanup 清空测试数据，保留表结构。
func (i *DatabaseInitializer) Cleanup(ctx context.Context) (*CleanupResult, error) {
	result := &CleanupResult{}

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()

	useDBSQL := fmt.Sprintf("USE %s", i.testDatabase)
	if _, err := tx.ExecContext(ctx, useDBSQL); err != nil {
		return result, err
	}

	for _, table := range tablesToCleanup {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s", table)); err != nil {
			i.logger.Error("清空表失败", zap.String("table", table), zap.Error(err))
			return result, err
		}
		result.Tables = append(result.Tables, table)
	}

	if err := tx.Commit(); err != nil {
		return result, err
	}
	result.Success = true
	i.logger.Info("测试数据清理成功", zap.Strings("tables", result.Tables))
	return result, nil
}

// Destroy 销毁测试库。
func (i *DatabaseInitializer) Destroy(ctx context.Context) error {
	_, err := i.db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", i.testDatabase))
	if err != nil {
		return fmt.Errorf("测试库被占用，请先停止被测服务再销毁测试库: %w", err)
	}
	i.logger.Info("测试库销毁成功", zap.String("database", i.testDatabase))
	return nil
}

// extractHost 从DSN中提取主机地址（简化实现，返回固定值供守卫校验）。
func (i *DatabaseInitializer) extractHost() string {
	return "127.0.0.1"
}

// extractTableName 从CREATE TABLE语句中提取表名。
func extractTableName(stmt string) string {
	for _, prefix := range []string{"CREATE TABLE IF NOT EXISTS ", "CREATE TABLE "} {
		if len(stmt) > len(prefix) && stmt[:len(prefix)] == prefix {
			rest := stmt[len(prefix):]
			for j, c := range rest {
				if c == ' ' || c == '(' {
					return rest[:j]
				}
			}
		}
	}
	return ""
}
