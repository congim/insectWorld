// Package taskprogress 任务进度聚合根，维护玩家任务进度与领取状态。
// TaskProgress聚合根订阅全服业务事件推进任务进度，是任务系统的核心聚合根。
// 对应spec.md 5.1.7.3节Task上下文，按玩家ID分片。
package taskprogress

import (
	"context"
	"fmt"

	taskerr "insectworld/server/task/domain/errors"
)

// 任务状态常量（规范1）。
const (
	StatusInProgress = 1 // 进行中
	StatusCompleted  = 2 // 已完成（可领取）
	StatusClaimed    = 3 // 已领取奖励
)

// 任务周期类型常量（规范1），对应tasks.json配置。
const (
	CycleOnce    = 1 // 一次性任务
	CycleDaily   = 2 // 日常
	CycleWeekly  = 3 // 周常
	CycleMonthly = 4 // 月常
)

// TaskProgress 任务进度聚合根，维护玩家单个任务的进度与领取状态。
type TaskProgress struct {
	taskID     int64 // 任务ID，对应tasks.json配置
	playerID   int64 // 玩家ID
	defID      int64 // 任务定义ID，对应tasks.json的任务定义
	current    int64 // 当前进度
	target     int64 // 目标进度，从配置查询
	status     int   // 任务状态：1=进行中 2=已完成 3=已领取
	cycle      int   // 周期类型：1=一次性 2=日常 3=周常 4=月常
	lastReset  int64 // 上次重置时间戳（毫秒），周期任务有效
	updateTime int64 // 进度最后更新时间戳（毫秒）
}

// NewTaskProgress 创建任务进度聚合根实例。
// target为目标进度，cycle为周期类型，从tasks.json配置查询注入。
func NewTaskProgress(taskID int64, playerID int64, defID int64, target int64, cycle int) *TaskProgress {
	return &TaskProgress{
		taskID:   taskID,
		playerID: playerID,
		defID:    defID,
		target:   target,
		status:   StatusInProgress,
		cycle:    cycle,
	}
}

// TaskID 返回任务ID。
func (t *TaskProgress) TaskID() int64 { return t.taskID }

// PlayerID 返回玩家ID。
func (t *TaskProgress) PlayerID() int64 { return t.playerID }

// Current 返回当前进度。
func (t *TaskProgress) Current() int64 { return t.current }

// Status 返回任务状态。
func (t *TaskProgress) Status() int { return t.status }

// Advance 推进任务进度，当进度达到目标时自动标记为已完成。
// delta为增量进度，now为当前时间戳（毫秒）。
func (t *TaskProgress) Advance(ctx context.Context, delta int64, now int64) (*ProgressChangedEvent, error) {
	if t.status == StatusClaimed {
		return nil, fmt.Errorf("任务奖励已领取，taskID=%d: %w", t.taskID, taskerr.ErrTaskAlreadyClaimed)
	}

	t.current += delta
	t.updateTime = now

	event := &ProgressChangedEvent{
		TaskID:   t.taskID,
		PlayerID: t.playerID,
		Current:  t.current,
		Target:   t.target,
	}

	// 进度达到目标自动完成
	if t.current >= t.target && t.status == StatusInProgress {
		t.status = StatusCompleted
		event.Completed = true
	}

	return event, nil
}

// ClaimReward 领取任务奖励，校验任务已完成且未重复领取。
func (t *TaskProgress) ClaimReward(now int64) (*RewardClaimedEvent, error) {
	if t.status != StatusCompleted {
		return nil, fmt.Errorf("任务未完成，taskID=%d，当前状态=%d: %w",
			t.taskID, t.status, taskerr.ErrTaskNotCompleted)
	}

	t.status = StatusClaimed
	t.updateTime = now

	return &RewardClaimedEvent{
		TaskID:   t.taskID,
		PlayerID: t.playerID,
		DefID:    t.defID,
	}, nil
}

// Reset 重置周期任务进度，用于日常/周常/月常重置。
func (t *TaskProgress) Reset(now int64) *ProgressChangedEvent {
	t.current = 0
	t.status = StatusInProgress
	t.lastReset = now
	t.updateTime = now
	return &ProgressChangedEvent{
		TaskID:   t.taskID,
		PlayerID: t.playerID,
		Current:  0,
		Target:   t.target,
	}
}

// ProgressChangedEvent 任务进度变更事件。
type ProgressChangedEvent struct {
	TaskID    int64 // 任务ID
	PlayerID  int64 // 玩家ID
	Current   int64 // 当前进度
	Target    int64 // 目标进度
	Completed bool  // 本次推进是否触发完成
}

// RewardClaimedEvent 任务奖励领取事件。
type RewardClaimedEvent struct {
	TaskID   int64 // 任务ID
	PlayerID int64 // 玩家ID
	DefID    int64 // 任务定义ID，用于查询奖励配置
}

// TaskProgressRepository 任务进度聚合根仓储接口，在domain层声明（规范3）。
type TaskProgressRepository interface {
	// LoadTaskProgress 加载任务进度聚合根
	LoadTaskProgress(ctx context.Context, taskID int64, playerID int64) (*TaskProgress, error)
	// SaveTaskProgress 保存任务进度聚合根
	SaveTaskProgress(ctx context.Context, tp *TaskProgress) error
}
