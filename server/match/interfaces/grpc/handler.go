// Package grpc Match服务接口层gRPC handler。
// 对接proto/match/match.proto定义的gRPC契约，编排application层command与query。
// TODO 后续protoc生成match.pb.go后接入，当前提供handler骨架。
package grpc

import (
	"go.uber.org/zap"

	"insectworld/server/match/application/query"
)

// MatchHandler Match服务gRPC handler。
// TODO 后续嵌入matchpb.UnimplementedMatchServiceServer，待proto生成后补充。
type MatchHandler struct {
	rankListQueryHandler *query.RankListQueryHandler // 排行榜查询handler
	logger               *zap.Logger                 // 结构化日志器（规范7）
}

// NewMatchHandler 创建Match服务gRPC handler实例。
func NewMatchHandler(
	rankListQueryHandler *query.RankListQueryHandler,
	logger *zap.Logger,
) *MatchHandler {
	return &MatchHandler{
		rankListQueryHandler: rankListQueryHandler,
		logger:               logger,
	}
}
