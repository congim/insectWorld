// Package query Operation服务application层读侧查询，CQRS读模型查询handler。
package query

import (
	"context"

	"go.uber.org/zap"
)

// SeasonQueryHandler 赛季查询handler。
type SeasonQueryHandler struct {
	logger *zap.Logger // 结构化日志器（规范7）
}

// NewSeasonQueryHandler 创建赛季查询handler实例。
func NewSeasonQueryHandler(logger *zap.Logger) *SeasonQueryHandler {
	return &SeasonQueryHandler{logger: logger}
}

// Handle 处理赛季查询。
func (h *SeasonQueryHandler) Handle(ctx context.Context, seasonID int64) error {
	h.logger.Debug("查询赛季信息", zap.Int64("season_id", seasonID))
	// TODO 后续从读模型查询赛季信息
	return nil
}
