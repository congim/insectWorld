// Package query Economy服务application层读侧查询，CQRS读模型查询handler。
package query

import (
	"context"

	"go.uber.org/zap"
)

// ResourceQueryHandler 资源查询handler。
type ResourceQueryHandler struct {
	logger *zap.Logger // 结构化日志器（规范7）
}

// NewResourceQueryHandler 创建资源查询handler实例。
func NewResourceQueryHandler(logger *zap.Logger) *ResourceQueryHandler {
	return &ResourceQueryHandler{logger: logger}
}

// Handle 处理资源查询。
func (h *ResourceQueryHandler) Handle(ctx context.Context, playerID int64) error {
	h.logger.Debug("查询玩家资源", zap.Int64("player_id", playerID))
	// TODO 后续从读模型查询玩家资源
	return nil
}
