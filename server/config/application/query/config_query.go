// Package query Config服务application层读侧查询，CQRS读模型查询handler。
package query

import (
	"context"

	"go.uber.org/zap"
)

// ConfigVersionQueryHandler 配置版本查询handler。
type ConfigVersionQueryHandler struct {
	logger *zap.Logger // 结构化日志器（规范7）
}

// NewConfigVersionQueryHandler 创建配置版本查询handler实例。
func NewConfigVersionQueryHandler(logger *zap.Logger) *ConfigVersionQueryHandler {
	return &ConfigVersionQueryHandler{logger: logger}
}

// Handle 处理配置版本查询。
func (h *ConfigVersionQueryHandler) Handle(ctx context.Context) error {
	h.logger.Debug("查询配置版本")
	// TODO 后续从读模型查询配置版本
	return nil
}
