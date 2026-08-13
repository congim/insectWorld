// Package datasource Persist服务数据源管理，提供多数据源连接池管理能力。
//
// infrastructure层技术适配，管理MySQL/MongoDB/Redis多数据源连接，
// 供snapshot/migration/archive/backup子目录使用。依赖方向infrastructure → domain（规范3）。
package datasource

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

// Manager 数据源管理器，管理多数据源连接池。
type Manager struct {
	mysqlDB *sql.DB      // MySQL连接池，热库数据源
	coldDB  *sql.DB      // 冷库MySQL连接池，归档冷数据存储
	logger  *zap.Logger  // 结构化日志
	mu      sync.RWMutex // 读写锁，保护连接池并发访问
}

// NewManager 创建数据源管理器实例。
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		logger: logger,
	}
}

// InitMySQL 初始化MySQL热库连接池。
func (m *Manager) InitMySQL(ctx context.Context, dsn string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("MySQL连接池初始化失败: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("MySQL连接测试失败: %w", err)
	}
	m.mysqlDB = db
	m.logger.Info("MySQL热库连接池初始化成功")
	return nil
}

// InitColdMySQL 初始化冷库MySQL连接池。
func (m *Manager) InitColdMySQL(ctx context.Context, dsn string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("冷库MySQL连接池初始化失败: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("冷库MySQL连接测试失败: %w", err)
	}
	m.coldDB = db
	m.logger.Info("冷库MySQL连接池初始化成功")
	return nil
}

// MySQL 返回MySQL热库连接池。
func (m *Manager) MySQL() *sql.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mysqlDB
}

// ColdMySQL 返回冷库MySQL连接池。
func (m *Manager) ColdMySQL() *sql.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.coldDB
}

// Close 关闭所有数据源连接。
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	if m.mysqlDB != nil {
		if err := m.mysqlDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if m.coldDB != nil {
		if err := m.coldDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("关闭数据源连接失败: %v", errs)
	}
	return nil
}
