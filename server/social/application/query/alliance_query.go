// Package query Social服务application层读侧查询，CQRS读模型查询handler。
package query

import (
	"context"

	"go.uber.org/zap"
)

// AllianceQueryHandler 联盟查询handler。
type AllianceQueryHandler struct {
	logger *zap.Logger // 结构化日志器（规范7）
}

// NewAllianceQueryHandler 创建联盟查询handler实例。
func NewAllianceQueryHandler(logger *zap.Logger) *AllianceQueryHandler {
	return &AllianceQueryHandler{logger: logger}
}

// Handle 处理联盟查询。
func (h *AllianceQueryHandler) Handle(ctx context.Context, allianceID int64) error {
	h.logger.Debug("查询联盟信息", zap.Int64("alliance_id", allianceID))
	// TODO 后续从读模型查询联盟信息
	return nil
}
