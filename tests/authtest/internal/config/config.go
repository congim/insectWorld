// Package config 测试端环境配置，集中定义本地MySQL/Redis/Gateway连接参数。
//
// MySQL root/123456为本地测试默认凭据，仅限本地测试，禁止用于生产环境（spec 4.3 安全性3）。
package config

import (
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/zap"
)

// TestConfig 测试端环境配置，含本地MySQL/Redis/Gateway/Web全部连接参数。
//
// 端口/数量用int整型（规范8），密码字段标注警示。
type TestConfig struct {
	MySQLHost       string `json:"mysqlHost"`       // MySQL主机地址，默认127.0.0.1
	MySQLPort       int    `json:"mysqlPort"`       // MySQL端口，默认3306
	MySQLUser       string `json:"mysqlUser"`       // MySQL用户名，默认root
	MySQLPassword   string `json:"mysqlPassword"`   // MySQL密码，默认123456，仅限本地测试，禁止用于生产环境
	TestDatabase    string `json:"testDatabase"`    // 测试数据库名，默认insectworld_test
	RedisHost       string `json:"redisHost"`       // Redis主机地址，默认127.0.0.1
	RedisPort       int    `json:"redisPort"`       // Redis端口，默认6379
	RedisPassword   string `json:"redisPassword"`   // Redis密码，默认空
	GatewayPort     string `json:"gatewayPort"`     // Gateway服务gRPC端口，默认50056
	GatewayWSURL    string `json:"gatewayWSURL"`    // Gateway服务WebSocket URL，默认ws://127.0.0.1:50057/auth
	TokenSigningKey string `json:"tokenSigningKey"` // 令牌签名密钥，默认test-signing-key-for-local-only
	WebListenAddr   string `json:"webListenAddr"`   // 测试端Web监听地址，默认:18080
	DDLScriptPath   string `json:"ddlScriptPath"`   // DDL脚本路径，默认../../shared/schema/ddl/gateway.sql
	GatewayDir      string `json:"gatewayDir"`      // Gateway服务目录，默认../gateway
}

// DefaultConfig 返回全默认值配置。
func DefaultConfig() *TestConfig {
	return &TestConfig{
		MySQLHost:       "127.0.0.1",
		MySQLPort:       3306,
		MySQLUser:       "root",
		MySQLPassword:   "123456",
		TestDatabase:    "insectworld_test",
		RedisHost:       "127.0.0.1",
		RedisPort:       6379,
		RedisPassword:   "",
		GatewayPort:     "50056",
		GatewayWSURL:    "ws://127.0.0.1:50057/auth",
		TokenSigningKey: "test-signing-key-for-local-only",
		WebListenAddr:   ":18080",
		DDLScriptPath:   "../shared/schema/ddl/gateway.sql",
		GatewayDir:      "../gateway",
	}
}

// LoadConfig 从JSON文件加载配置，缺失文件时使用全默认值并记录Warn日志。
//
// 缺失字段用默认值填充，JSON解析失败返回错误。
func LoadConfig(path string, logger *zap.Logger) (*TestConfig, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if logger != nil {
			logger.Warn("配置文件加载失败，使用默认配置", zap.String("path", path), zap.Error(err))
		}
		return cfg, nil
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("配置文件解析失败: %w", err)
	}
	return cfg, nil
}

// MySQLDSN 拼接MySQL DSN字符串，供环境变量注入使用。
func (c *TestConfig) MySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true",
		c.MySQLUser, c.MySQLPassword, c.MySQLHost, c.MySQLPort, c.TestDatabase)
}

// MySQLDSNNoDB 拼接不指定库的MySQL DSN，用于建库建表。
func (c *TestConfig) MySQLDSNNoDB() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=true",
		c.MySQLUser, c.MySQLPassword, c.MySQLHost, c.MySQLPort)
}

// RedisAddr 返回Redis地址 host:port。
func (c *TestConfig) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}
