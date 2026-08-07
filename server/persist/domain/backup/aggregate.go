// Package backup 备份聚合根，维护数据备份任务的状态与恢复点。
//
// 备份聚合根管理数据备份的生命周期，包括全量备份（mysqldump）、增量备份（binlog）
// 和日志备份，支持时间点恢复（PITR），保证数据安全与可恢复性。
package backup

import (
	"context"
	"fmt"
)

// 备份类型枚举常量。
// 取值映射：1=全量备份 2=增量备份 3=日志备份
const (
	BackupTypeFull    = 1 // 全量备份，mysqldump导出全部数据
	BackupTypeIncremental = 2 // 增量备份，基于binlog的增量数据备份
	BackupTypeLog     = 3 // 日志备份，binlog日志文件备份
)

// 备份任务状态枚举常量。
// 取值映射：1=待执行 2=执行中 3=已完成 4=失败
const (
	BackupStatusPending  = 1 // 待执行状态
	BackupStatusRunning  = 2 // 执行中状态
	BackupStatusCompleted = 3 // 已完成状态
	BackupStatusFailed   = 4 // 失败状态
)

// BackupTask 备份任务，描述单次备份操作的配置与状态。
type BackupTask struct {
	backupID     int64 // 备份任务ID，全局唯一
	backupType   int   // 备份类型：1=全量 2=增量 3=日志
	status       int   // 任务状态：1=待执行 2=执行中 3=已完成 4=失败
	createTime   int64 // 创建时间戳，毫秒级
	restorePoint int64 // 恢复点时间戳，毫秒级，用于PITR时间点恢复
}

// NewBackupTask 创建备份任务实例。
func NewBackupTask(backupID int64, backupType int, createTime int64) *BackupTask {
	return &BackupTask{
		backupID:   backupID,
		backupType: backupType,
		status:     BackupStatusPending,
		createTime: createTime,
	}
}

// BackupID 返回备份任务ID。
func (b *BackupTask) BackupID() int64 { return b.backupID }

// Status 返回任务状态。
func (b *BackupTask) Status() int { return b.status }

// Start 执行备份任务。
func (b *BackupTask) Start() error {
	if b.status != BackupStatusPending {
		return fmt.Errorf("备份任务状态非待执行，当前状态：%d", b.status)
	}
	b.status = BackupStatusRunning
	return nil
}

// Complete 完成备份任务，设置恢复点时间戳。
func (b *BackupTask) Complete(restorePoint int64) error {
	if b.status != BackupStatusRunning {
		return fmt.Errorf("备份任务状态非执行中，当前状态：%d", b.status)
	}
	b.status = BackupStatusCompleted
	b.restorePoint = restorePoint
	return nil
}

// BackupRepository 备份仓储接口，在domain层声明，infrastructure层实现。
type BackupRepository interface {
	// Save 保存备份任务状态。
	Save(ctx context.Context, task *BackupTask) error
	// FindRestorePoint 查询指定时间之前的最近恢复点。
	FindRestorePoint(ctx context.Context, beforeTime int64) (*BackupTask, error)
}