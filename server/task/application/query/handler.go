// Package query Task服务application层读侧查询，CQRS读模型查询handler。
package query

import (
	"context"

	"go.uber.org/zap"
)

// TaskListQuery 任务列表查询DTO。
type TaskListQuery struct {
	PlayerID int64 // 玩家ID
}

// TaskListResult 任务列表查询结果DTO。
type TaskListResult struct {
	Tasks []TaskItem // 任务列表
}

// TaskItem 任务项DTO。
type TaskItem struct {
	TaskID  int64 // 任务ID
	DefID   int64 // 任务定义ID
	Current int64 // 当前进度
	Target  int64 // 目标进度
	Status  int   // 任务状态：1=进行中 2=已完成 3=已领取
}

// TaskReadModel 任务读模型查询接口，在domain层声明，infrastructure层实现。
// CQRS读侧通过读模型表t_task_progress_read查询，不经过聚合根。
type TaskReadModel interface {
	// QueryTaskList 查询玩家任务列表
	QueryTaskList(ctx context.Context, playerID int64) ([]TaskItem, error)
}

// TaskListQueryHandler 任务列表查询handler，CQRS读侧。
type TaskListQueryHandler struct {
	taskReadModel TaskReadModel // 任务读模型查询接口，infrastructure层注入
	logger        *zap.Logger   // 结构化日志器（规范7）
}

// NewTaskListQueryHandler 创建任务列表查询handler实例。
// taskReadModel由infrastructure层实现，cmd/main.go组装时注入。
func NewTaskListQueryHandler(taskReadModel TaskReadModel, logger *zap.Logger) *TaskListQueryHandler {
	return &TaskListQueryHandler{taskReadModel: taskReadModel, logger: logger}
}

// Handle 处理任务列表查询。
func (h *TaskListQueryHandler) Handle(ctx context.Context, q TaskListQuery) (*TaskListResult, error) {
	tasks, err := h.taskReadModel.QueryTaskList(ctx, q.PlayerID)
	if err != nil {
		return nil, err
	}

	h.logger.Debug("查询任务列表",
		zap.Int64("player_id", q.PlayerID),
		zap.Int("count", len(tasks)),
	)
	return &TaskListResult{Tasks: tasks}, nil
}
