// Package backup Persist服务备份仓储实现，实现domain/backup.BackupRepository接口。
package backup

import (
	"context"
	"database/sql"
	"fmt"

	domainBackup "insectworld/server/persist/domain/backup"
)

// BackupRepoImpl BackupRepository接口实现。
type BackupRepoImpl struct {
	db *sql.DB // MySQL连接池
}

// NewBackupRepoImpl 创建备份仓储实现实例。
func NewBackupRepoImpl(db *sql.DB) *BackupRepoImpl {
	return &BackupRepoImpl{db: db}
}

// Save 保存备份任务状态。
// TODO 后续接入mysqldump/binlog备份执行，当前仅记录任务元数据。
func (r *BackupRepoImpl) Save(ctx context.Context, task *domainBackup.BackupTask) error {
	_ = ctx
	_ = task
	return nil
}

// FindRestorePoint 查询指定时间之前的最近恢复点。
// TODO 后续接入读模型查询，当前返回nil表示无恢复点。
func (r *BackupRepoImpl) FindRestorePoint(ctx context.Context, beforeTime int64) (*domainBackup.BackupTask, error) {
	_ = ctx
	_ = beforeTime
	_ = fmt.Sprintf("查询beforeTime=%d的最近恢复点", beforeTime)
	return nil, nil
}
