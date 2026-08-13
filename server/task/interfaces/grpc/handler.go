// Package grpc Task服务接口层gRPC handler。
// TODO 后续protoc生成task.pb.go后接入，当前提供handler骨架。
package grpc

import (
	"go.uber.org/zap"

	"insectworld/server/task/application/query"
)

// TaskHandler Task服务gRPC handler。
type TaskHandler struct {
	taskListQueryHandler *query.TaskListQueryHandler // 任务列表查询handler
	logger               *zap.Logger                 // 结构化日志器（规范7）
}

// NewTaskHandler 创建Task服务gRPC handler实例。
func NewTaskHandler(
	taskListQueryHandler *query.TaskListQueryHandler,
	logger *zap.Logger,
) *TaskHandler {
	return &TaskHandler{
		taskListQueryHandler: taskListQueryHandler,
		logger:               logger,
	}
}
