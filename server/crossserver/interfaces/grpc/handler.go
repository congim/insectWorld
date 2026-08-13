// Package grpc CrossServer服务接口层gRPC handler。
// 对接proto/crossserver/crossserver.proto定义的gRPC契约，编排application层command与query。
// TODO 后续protoc生成crossserver.pb.go后接入，当前提供handler骨架。
package grpc

import (
	"go.uber.org/zap"

	"insectworld/server/crossserver/application/query"
)

// CrossServerHandler CrossServer服务gRPC handler。
// TODO 后续嵌入crossserverpb.UnimplementedCrossServerServiceServer，待proto生成后补充。
type CrossServerHandler struct {
	nodeListQueryHandler *query.NodeListQueryHandler // 节点列表查询handler
	logger               *zap.Logger                 // 结构化日志器（规范7）
}

// NewCrossServerHandler 创建CrossServer服务gRPC handler实例。
func NewCrossServerHandler(
	nodeListQueryHandler *query.NodeListQueryHandler,
	logger *zap.Logger,
) *CrossServerHandler {
	return &CrossServerHandler{
		nodeListQueryHandler: nodeListQueryHandler,
		logger:               logger,
	}
}
