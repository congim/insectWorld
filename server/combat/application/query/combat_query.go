// Package query Combat服务application层读侧查询，CQRS读模型查询handler。
package query

import (
	"context"

	"go.uber.org/zap"
)

// CombatStateQueryHandler 战斗状态查询handler。
type CombatStateQueryHandler struct {
	logger *zap.Logger // 结构化日志器（规范7）
}

// NewCombatStateQueryHandler 创建战斗状态查询handler实例。
func NewCombatStateQueryHandler(logger *zap.Logger) *CombatStateQueryHandler {
	return &CombatStateQueryHandler{logger: logger}
}

// Handle 处理战斗状态查询。
func (h *CombatStateQueryHandler) Handle(ctx context.Context, combatID int64) error {
	h.logger.Debug("查询战斗状态", zap.Int64("combat_id", combatID))
	// TODO 后续从读模型查询战斗状态
	return nil
}
