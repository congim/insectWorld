// Package grpc Combat服务接口层gRPC handler，实现CombatServiceServer接口。
package grpc

import (
	"context"

	"go.uber.org/zap"

	combatpb "insectworld/server/shared/proto/combat"
)

// CombatHandler Combat服务gRPC handler，实现CombatServiceServer接口。
type CombatHandler struct {
	combatpb.UnimplementedCombatServiceServer             // 嵌入未实现基类，保证前向兼容
	logger                                    *zap.Logger // 结构化日志器（规范7）
}

// NewCombatHandler 创建Combat服务gRPC handler实例。
func NewCombatHandler(logger *zap.Logger) *CombatHandler {
	return &CombatHandler{logger: logger}
}

// GetEntityAttributes 查询实体属性。
func (h *CombatHandler) GetEntityAttributes(ctx context.Context, req *combatpb.GetAttributesRequest) (*combatpb.AttributesResponse, error) {
	h.logger.Info("查询实体属性")
	return &combatpb.AttributesResponse{}, nil
}

// GetCombatState 查询战斗状态。
func (h *CombatHandler) GetCombatState(ctx context.Context, req *combatpb.GetCombatStateRequest) (*combatpb.CombatStateResponse, error) {
	h.logger.Info("查询战斗状态")
	return &combatpb.CombatStateResponse{}, nil
}

// GetCombatReport 查询战斗报告。
func (h *CombatHandler) GetCombatReport(ctx context.Context, req *combatpb.ReportRequest) (*combatpb.ReportResponse, error) {
	h.logger.Info("查询战斗报告")
	return &combatpb.ReportResponse{}, nil
}
