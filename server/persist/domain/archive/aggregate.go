// Package archive 归档聚合根，维护冷数据归档任务的状态与规则。
//
// 归档聚合根管理冷数据归档的生命周期，将热库中的冷数据（如过期战报、历史赛季数据）
// 迁移到冷存储（如MongoDB、冷库MySQL），归档后从热库删除，释放热库空间。
package archive

import (
	"context"
	"fmt"
)

// 归档任务状态枚举常量。
// 取值映射：1=待执行 2=执行中 3=已完成 4=失败
const (
	ArchiveStatusPending  = 1 // 待执行状态，归档任务已创建尚未开始
	ArchiveStatusRunning  = 2 // 执行中状态，归档任务正在执行
	ArchiveStatusCompleted = 3 // 已完成状态，归档任务执行成功
	ArchiveStatusFailed   = 4 // 失败状态，归档任务执行失败
)

// 源数据类型枚举常量，指定归档的数据来源。
// 取值映射：1=战报 2=赛季数据 3=玩家冷数据 4=日志数据
const (
	SourceTypeCombatReport  = 1 // 战报数据，从t_combat_report归档到MongoDB
	SourceTypeSeasonData    = 2 // 赛季数据，从t_season_snapshot归档到冷库
	SourceTypePlayerCold    = 3 // 玩家冷数据，从t_player归档到t_player_archive
	SourceTypeLogData       = 4 // 日志数据，归档到冷存储
)

// 目标存储类型枚举常量，指定归档数据的存储目标。
// 取值映射：1=MongoDB 2=冷库MySQL 3=S3对象存储
const (
	TargetStorageMongoDB   = 1 // MongoDB文档存储，适用于战报等非结构化数据
	TargetStorageColdMySQL = 2 // 冷库MySQL，适用于结构化冷数据
	TargetStorageS3        = 3 // S3对象存储，适用于大文件冷存储
)

// ArchiveTask 归档任务，描述单次归档操作的配置与状态。
type ArchiveTask struct {
	taskID       int64  // 归档任务ID，全局唯一
	ruleID       string // 归档规则ID，标识归档规则配置
	sourceType   int    // 源数据类型：1=战报 2=赛季数据 3=玩家冷数据 4=日志数据
	targetStorage int   // 目标存储：1=MongoDB 2=冷库MySQL 3=S3
	status       int    // 任务状态：1=待执行 2=执行中 3=已完成 4=失败
	archiveTime  int64  // 归档执行时间戳，毫秒级
}

// NewArchiveTask 创建归档任务实例。
func NewArchiveTask(taskID int64, ruleID string, sourceType, targetStorage int) *ArchiveTask {
	return &ArchiveTask{
		taskID:        taskID,
		ruleID:        ruleID,
		sourceType:    sourceType,
		targetStorage: targetStorage,
		status:        ArchiveStatusPending,
	}
}

// TaskID 返回归档任务ID。
func (a *ArchiveTask) TaskID() int64 { return a.taskID }

// Status 返回任务状态。
func (a *ArchiveTask) Status() int { return a.status }

// Start 执行归档任务，状态从待执行变为执行中。
func (a *ArchiveTask) Start() error {
	if a.status != ArchiveStatusPending {
		return fmt.Errorf("归档任务状态非待执行，当前状态：%d", a.status)
	}
	a.status = ArchiveStatusRunning
	return nil
}

// Complete 完成归档任务。
func (a *ArchiveTask) Complete(archiveTime int64) error {
	if a.status != ArchiveStatusRunning {
		return fmt.Errorf("归档任务状态非执行中，当前状态：%d", a.status)
	}
	a.status = ArchiveStatusCompleted
	a.archiveTime = archiveTime
	return nil
}

// ArchiveRepository 归档仓储接口，在domain层声明，infrastructure层实现。
type ArchiveRepository interface {
	// Save 保存归档任务状态。
	Save(ctx context.Context, task *ArchiveTask) error
	// FindPending 查询待执行的归档任务。
	FindPending(ctx context.Context, limit int) ([]*ArchiveTask, error)
}