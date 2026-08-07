// Package sutmgr 测试端被测服务管理器，以子进程方式启动/停止Gateway服务并注入环境变量。
package sutmgr

import (
	"fmt"
	"os"

	"insectworld/server/authtest/internal/config"
)

// EnvironmentBuilder 环境变量构建器，组装Gateway所需环境变量。
type EnvironmentBuilder struct{}

// NewEnvironmentBuilder 创建环境变量构建器实例。
func NewEnvironmentBuilder() *EnvironmentBuilder {
	return &EnvironmentBuilder{}
}

// Build 组装Gateway所需环境变量切片，继承父进程环境并追加。
func (b *EnvironmentBuilder) Build(cfg *config.TestConfig) []string {
	env := os.Environ()
	env = append(env, fmt.Sprintf("GATEWAY_MYSQL_DSN=%s", cfg.MySQLDSN()))
	env = append(env, fmt.Sprintf("GATEWAY_REDIS_ADDR=%s", cfg.RedisAddr()))
	env = append(env, fmt.Sprintf("GATEWAY_REDIS_PASSWORD=%s", cfg.RedisPassword))
	env = append(env, fmt.Sprintf("GATEWAY_TOKEN_SIGNING_KEY=%s", cfg.TokenSigningKey))
	return env
}
