// Package snapshot 快照聚合根，维护快照任务状态与范围。
//
// 快照聚合根管理数据快照的生命周期，包括全量快照与增量快照。
// 全量快照保存当前所有数据，增量快照保存指定版本范围内的变更数据。
package snapshot

import (
	"context"
	"fmt"

	"insectworld/server/persist/domain/vo"
)

// 快照类型枚举常量。
// 取值映射：1=全量快照 2=增量快照
const (
	SnapshotTypeFull        = 1 // 全量快照，保存当前所有数据
	SnapshotTypeIncremental = 2 // 增量快照，保存指定版本范围内的变更数据
)

// 快照状态枚举常量。
// 取值映射：1=待执行 2=执行中 3=完成 4=失败
const (
	SnapshotStatusPending   = 1 // 待执行状态，快照任务已创建尚未开始
	SnapshotStatusRunning   = 2 // 执行中状态，快照任务正在执行
	SnapshotStatusCompleted = 3 // 完成状态，快照任务执行成功
	SnapshotStatusFailed    = 4 // 失败状态，快照任务执行失败
)

// Snapshot 快照聚合根，维护快照任务的状态与数据范围。
type Snapshot struct {
	snapshotID  int64            // 快照ID，全局唯一，由雪花算法生成
	scope       int              // 快照类型：1=全量 2=增量
	status      int              // 快照状态：1=待执行 2=执行中 3=完成 4=失败
	createTime  int64            // 创建时间戳，毫秒级
	executeTime int64            // 执行完成时间戳，毫秒级
	dataRange   vo.SnapshotRange // 快照数据范围，增量快照的版本范围与表列表
}

// NewSnapshot 创建快照聚合根实例。
func NewSnapshot(snapshotID int64, scope int, createTime int64, dataRange vo.SnapshotRange) *Snapshot {
	return &Snapshot{
		snapshotID: snapshotID,
		scope:      scope,
		status:     SnapshotStatusPending,
		createTime: createTime,
		dataRange:  dataRange,
	}
}

// SnapshotID 返回快照ID。
func (s *Snapshot) SnapshotID() int64 { return s.snapshotID }

// Scope 返回快照类型。
func (s *Snapshot) Scope() int { return s.scope }

// Status 返回快照状态。
func (s *Snapshot) Status() int { return s.status }

// CreateTime 返回创建时间戳。
func (s *Snapshot) CreateTime() int64 { return s.createTime }

// ExecuteTime 返回执行完成时间戳。
func (s *Snapshot) ExecuteTime() int64 { return s.executeTime }

// RestoreSnapshot 从持久化恢复快照聚合根，用于Repository Find方法重建聚合根状态。
func RestoreSnapshot(snapshotID int64, scope, status int, createTime, executeTime int64, dataRange vo.SnapshotRange) *Snapshot {
	return &Snapshot{
		snapshotID:  snapshotID,
		scope:       scope,
		status:      status,
		createTime:  createTime,
		executeTime: executeTime,
		dataRange:   dataRange,
	}
}

// Start 执行快照任务，状态从待执行变为执行中。
func (s *Snapshot) Start() error {
	if s.status != SnapshotStatusPending {
		return fmt.Errorf("快照任务状态非待执行，当前状态：%d", s.status)
	}
	s.status = SnapshotStatusRunning
	return nil
}

// Complete 完成快照任务，状态从执行中变为完成。
func (s *Snapshot) Complete(executeTime int64) error {
	if s.status != SnapshotStatusRunning {
		return fmt.Errorf("快照任务状态非执行中，当前状态：%d", s.status)
	}
	s.status = SnapshotStatusCompleted
	s.executeTime = executeTime
	return nil
}

// Fail 快照任务失败，状态从执行中变为失败。
func (s *Snapshot) Fail(executeTime int64) {
	s.status = SnapshotStatusFailed
	s.executeTime = executeTime
}

// SnapshotRepository 快照仓储接口，在domain层声明，infrastructure层实现。
type SnapshotRepository interface {
	// Save 保存快照聚合根状态。
	Save(ctx context.Context, s *Snapshot) error
	// FindLatest 查询指定类型的最新快照。
	FindLatest(ctx context.Context, scope int) (*Snapshot, error)
}
