// Package training 定义单位训练任务聚合及训练状态机。
package training

import (
	"context"
	"fmt"

	gameerr "insectworld/server/game/domain/errors"
)

// Status 是训练任务状态枚举。
type Status int32

// 训练任务状态常量。
const (
	StatusTraining Status = 1 // 训练中
	StatusComplete Status = 2 // 已完成
)

// Task 是单位训练任务聚合根。
type Task struct {
	id         int64  // 训练任务ID，全局唯一
	playerID   int64  // 所属玩家ID
	buildingID int64  // 执行训练的建筑实例ID
	unitTypeID string // 单位类型稳定ID，来源于游戏包
	count      int64  // 训练数量，必须大于0
	status     Status // 当前状态：1=训练中，2=已完成
	startedAt  int64  // 训练开始时间戳，Unix毫秒
	completeAt int64  // 可完成时间戳，Unix毫秒
	commandID  string // 开始训练命令幂等键
}

// NewTask 创建训练中的任务聚合。
func NewTask(id int64, playerID int64, buildingID int64, unitTypeID string, count int64, startedAt int64, completeAt int64, commandID string) (*Task, error) {
	if id <= 0 || playerID <= 0 || buildingID <= 0 || unitTypeID == "" || count <= 0 || startedAt <= 0 || completeAt <= startedAt || commandID == "" {
		return nil, fmt.Errorf("训练任务参数非法，taskID=%d，playerID=%d: %w", id, playerID, gameerr.ErrInvalidCommand)
	}
	return &Task{id: id, playerID: playerID, buildingID: buildingID, unitTypeID: unitTypeID, count: count, status: StatusTraining, startedAt: startedAt, completeAt: completeAt, commandID: commandID}, nil
}

// Complete 在达到完成时间后结束训练；重复完成保持幂等。
func (t *Task) Complete(nowMs int64) error {
	if t.status == StatusComplete {
		return nil
	}
	if nowMs < t.completeAt {
		return fmt.Errorf("训练尚未达到完成时间，taskID=%d，completeAt=%d: %w", t.id, t.completeAt, gameerr.ErrTaskNotReady)
	}
	t.status = StatusComplete
	return nil
}

// ID 返回训练任务ID。
func (t *Task) ID() int64 { return t.id }

// PlayerID 返回所属玩家ID。
func (t *Task) PlayerID() int64 { return t.playerID }

// BuildingID 返回执行训练的建筑实例ID。
func (t *Task) BuildingID() int64 { return t.buildingID }

// UnitTypeID 返回训练单位类型稳定ID。
func (t *Task) UnitTypeID() string { return t.unitTypeID }

// Count 返回训练数量。
func (t *Task) Count() int64 { return t.count }

// Status 返回训练任务当前状态。
func (t *Task) Status() Status { return t.status }

// StartedAt 返回训练开始时间戳，单位毫秒。
func (t *Task) StartedAt() int64 { return t.startedAt }

// CompleteAt 返回最早完成时间戳，单位毫秒。
func (t *Task) CompleteAt() int64 { return t.completeAt }

// CommandID 返回开始训练命令幂等键。
func (t *Task) CommandID() string { return t.commandID }

// Clone 返回与仓储内部状态隔离的训练任务副本。
func (t *Task) Clone() *Task {
	copyValue := *t
	return &copyValue
}

// Repository 是训练任务仓储，由Growth上下文拥有写权限。
type Repository interface {
	FindByID(ctx context.Context, taskID int64) (*Task, error)
	FindByCommandID(ctx context.Context, commandID string) (*Task, error)
	SaveIfAbsent(ctx context.Context, task *Task) (*Task, bool, error)
	Save(ctx context.Context, task *Task) error
}

// Roster 是玩家已训练单位名册端口，由Growth拥有写权限，Combat只能消费投影。
type Roster interface {
	Grant(ctx context.Context, playerID int64, unitTypeID string, count int64, operationID string) error
	Count(ctx context.Context, playerID int64, unitTypeID string) (int64, error)
}
