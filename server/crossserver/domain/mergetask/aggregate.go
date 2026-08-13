// Package mergetask 合服任务聚合根，维护合服迁移的进度与状态。
// MergeTask聚合根管理源区数据迁移到目标区的完整流程。
// 对应spec.md 5.1.7.5节CrossServer上下文功能3"合服"。
package mergetask

import (
	"context"
	"fmt"

	cserr "insectworld/server/crossserver/domain/errors"
)

// 合服任务状态常量（规范1）。
const (
	StatusPending    = 1 // 待执行
	StatusMigrating  = 2 // 迁移中
	StatusVerifying  = 3 // 校验中
	StatusCompleted  = 4 // 已完成
	StatusFailed     = 5 // 失败
	StatusRolledBack = 6 // 已回滚
)

// 迁移阶段常量（规范1），标识当前迁移的数据类型。
const (
	PhasePlayer   = 1 // 玩家数据迁移
	PhaseAlliance = 2 // 联盟数据迁移
	PhaseWorld    = 3 // 世界数据迁移
	PhaseVerify   = 4 // 数据校验
)

// MergeTask 合服任务聚合根。
type MergeTask struct {
	taskID       int64   // 合服任务ID，全局唯一
	sourceZones  []int64 // 源区ID列表
	targetZone   int64   // 目标区ID
	status       int     // 任务状态：1=待执行 2=迁移中 3=校验中 4=已完成 5=失败 6=已回滚
	currentPhase int     // 当前迁移阶段：1=玩家 2=联盟 3=世界 4=校验
	progress     int64   // 迁移进度（已完成记录数）
	totalRecords int64   // 总记录数
	createTime   int64   // 创建时间戳（毫秒）
	finishTime   int64   // 完成时间戳（毫秒），0表示未完成
	errorMsg     string  // 失败原因，status=5时有效
}

// NewMergeTask 创建合服任务聚合根实例。
func NewMergeTask(taskID int64, sourceZones []int64, targetZone int64, totalRecords int64, createTime int64) *MergeTask {
	return &MergeTask{
		taskID:       taskID,
		sourceZones:  sourceZones,
		targetZone:   targetZone,
		status:       StatusPending,
		currentPhase: PhasePlayer,
		totalRecords: totalRecords,
		createTime:   createTime,
	}
}

// TaskID 返回合服任务ID。
func (m *MergeTask) TaskID() int64 { return m.taskID }

// Status 返回任务状态。
func (m *MergeTask) Status() int { return m.status }

// Progress 返回迁移进度。
func (m *MergeTask) Progress() int64 { return m.progress }

// Start 开始迁移，状态从待执行转为迁移中。
func (m *MergeTask) Start() error {
	if m.status != StatusPending {
		return fmt.Errorf("合服任务非待执行状态，taskID=%d，当前状态=%d: %w",
			m.taskID, m.status, cserr.ErrMergeTaskRunning)
	}
	m.status = StatusMigrating
	return nil
}

// AdvanceProgress 推进迁移进度，delta为本次迁移的记录数。
func (m *MergeTask) AdvanceProgress(delta int64, phase int) {
	m.progress += delta
	m.currentPhase = phase
}

// EnterVerify 进入校验阶段，状态从迁移中转为校验中。
func (m *MergeTask) EnterVerify() {
	m.status = StatusVerifying
	m.currentPhase = PhaseVerify
}

// Complete 完成合服，状态从校验中转为已完成。
func (m *MergeTask) Complete(finishTime int64) (*MergeCompletedEvent, error) {
	if m.status != StatusVerifying {
		return nil, fmt.Errorf("合服任务非校验中状态，taskID=%d: %w", m.taskID, cserr.ErrMergeTaskRunning)
	}
	m.status = StatusCompleted
	m.finishTime = finishTime
	return &MergeCompletedEvent{
		TaskID:     m.taskID,
		TargetZone: m.targetZone,
	}, nil
}

// Fail 标记合服失败。
func (m *MergeTask) Fail(errMsg string, finishTime int64) *MergeFailedEvent {
	m.status = StatusFailed
	m.errorMsg = errMsg
	m.finishTime = finishTime
	return &MergeFailedEvent{
		TaskID:   m.taskID,
		ErrorMsg: errMsg,
	}
}

// MergeCompletedEvent 合服完成事件。
type MergeCompletedEvent struct {
	TaskID     int64 // 合服任务ID
	TargetZone int64 // 目标区ID
}

// MergeFailedEvent 合服失败事件。
type MergeFailedEvent struct {
	TaskID   int64  // 合服任务ID
	ErrorMsg string // 失败原因
}

// MergeTaskRepository 合服任务仓储接口，在domain层声明（规范3）。
type MergeTaskRepository interface {
	// LoadMergeTask 加载合服任务聚合根
	LoadMergeTask(ctx context.Context, taskID int64) (*MergeTask, error)
	// SaveMergeTask 保存合服任务聚合根
	SaveMergeTask(ctx context.Context, m *MergeTask) error
}
