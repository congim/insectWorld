// Package grpc Operation服务接口层gRPC handler，实现OperationServiceServer接口。
package grpc

import (
	"context"

	"go.uber.org/zap"

	operationpb "insectworld/server/shared/proto/operation"
)

// OperationHandler Operation服务gRPC handler，实现OperationServiceServer接口。
type OperationHandler struct {
	operationpb.UnimplementedOperationServiceServer             // 嵌入未实现基类，保证前向兼容
	logger                                          *zap.Logger // 结构化日志器（规范7）
}

// NewOperationHandler 创建Operation服务gRPC handler实例。
func NewOperationHandler(logger *zap.Logger) *OperationHandler {
	return &OperationHandler{logger: logger}
}

// GetCurrentSeason 查询当前赛季。
func (h *OperationHandler) GetCurrentSeason(ctx context.Context, req *operationpb.GetSeasonRequest) (*operationpb.SeasonResponse, error) {
	h.logger.Info("查询当前赛季")
	return &operationpb.SeasonResponse{}, nil
}

// GetRanking 查询排行。
func (h *OperationHandler) GetRanking(ctx context.Context, req *operationpb.GetRankingRequest) (*operationpb.RankingResponse, error) {
	h.logger.Info("查询排行")
	return &operationpb.RankingResponse{}, nil
}

// GetSeasonHistory 查询赛季历史。
func (h *OperationHandler) GetSeasonHistory(ctx context.Context, req *operationpb.SeasonHistoryRequest) (*operationpb.SeasonHistoryResponse, error) {
	h.logger.Info("查询赛季历史")
	return &operationpb.SeasonHistoryResponse{}, nil
}

// TriggerGameEvent 触发游戏事件。
func (h *OperationHandler) TriggerGameEvent(ctx context.Context, req *operationpb.TriggerEventRequest) (*operationpb.TriggerEventResponse, error) {
	h.logger.Info("触发游戏事件")
	return &operationpb.TriggerEventResponse{}, nil
}
