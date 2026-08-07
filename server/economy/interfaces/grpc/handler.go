// Package grpc Economy服务接口层gRPC handler，实现EconomyServiceServer接口。
package grpc

import (
	"context"

	"go.uber.org/zap"

	economypb "insectworld/server/shared/proto/economy"
)

// EconomyHandler Economy服务gRPC handler，实现EconomyServiceServer接口。
type EconomyHandler struct {
	economypb.UnimplementedEconomyServiceServer             // 嵌入未实现基类，保证前向兼容
	logger                                      *zap.Logger // 结构化日志器（规范7）
}

// NewEconomyHandler 创建Economy服务gRPC handler实例。
func NewEconomyHandler(logger *zap.Logger) *EconomyHandler {
	return &EconomyHandler{logger: logger}
}

// GetPlayerResources 查询玩家资源。
func (h *EconomyHandler) GetPlayerResources(ctx context.Context, req *economypb.GetResourcesRequest) (*economypb.ResourcesResponse, error) {
	h.logger.Info("查询玩家资源", zap.Int64("player_id", req.GetPlayerId()))
	return &economypb.ResourcesResponse{}, nil
}

// CheckSufficient 校验资源是否充足。
func (h *EconomyHandler) CheckSufficient(ctx context.Context, req *economypb.CheckSufficientRequest) (*economypb.CheckSufficientResponse, error) {
	h.logger.Info("校验资源充足")
	return &economypb.CheckSufficientResponse{}, nil
}

// GetAllianceBonus 查询联盟加成。
func (h *EconomyHandler) GetAllianceBonus(ctx context.Context, req *economypb.GetAllianceBonusRequest) (*economypb.AllianceBonusResponse, error) {
	h.logger.Info("查询联盟加成")
	return &economypb.AllianceBonusResponse{}, nil
}
