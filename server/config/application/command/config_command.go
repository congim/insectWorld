// Package command Config服务application层命令，编排配置提交/回滚操作。
package command

import (
	"go.uber.org/zap"
)

// ConfigCommandHandler 配置命令handler，编排配置提交/回滚/热更操作。
type ConfigCommandHandler struct {
	logger *zap.Logger // 结构化日志器（规范7）
}

// NewConfigCommandHandler 创建配置命令handler实例。
func NewConfigCommandHandler(logger *zap.Logger) *ConfigCommandHandler {
	return &ConfigCommandHandler{logger: logger}
}
