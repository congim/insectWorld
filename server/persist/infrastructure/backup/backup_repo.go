// Package backup Persist服务备份仓储实现，实现domain/backup.BackupRepository接口。
//
// infrastructure层技术适配，将备份任务聚合根状态持久化到t_backup_task表。
// 表名引用shared/schema/tables常量（规范2），不硬编码表名字符串。
package backup

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	domainBackup "insectworld/server/persist/domain/backup"
	"insectworld/server/shared/schema/tables"
)

// BackupRepoImpl BackupRepository接口实现。
type BackupRepoImpl struct {
	db     *sql.DB     // MySQL连接池
	logger *zap.Logger // 结构化日志
}

// NewBackupRepoImpl 创建备份仓储实现实例。
func NewBackupRepoImpl(db *sql.DB, logger *zap.Logger) *BackupRepoImpl {
	return &BackupRepoImpl{db: db, logger: logger}
}

// Save 保存备份任务状态，表名引用shared/schema/tables.TBackupTask常量（规范2）。
// backup_path用空字符串占位，实际备份文件路径在备份执行完成后更新。
func (r *BackupRepoImpl) Save(ctx context.Context, task *domainBackup.BackupTask) error {
	if r.db == nil {
		return fmt.Errorf("数据库未初始化，无法保存备份任务")
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO `+tables.TBackupTask+` (id, backup_type, status, backup_path, file_size, create_time) VALUES (?, ?, ?, '', 0, ?)`,
		task.BackupID(), task.BackupType(), task.Status(), task.CreateTime(),
	)
	if err != nil {
		return fmt.Errorf("保存备份任务失败: %w", err)
	}

	r.logger.Info("备份任务保存成功",
		zap.Int64("backup_id", task.BackupID()),
		zap.Int("backup_type", task.BackupType()),
		zap.Int("status", task.Status()),
	)
	return nil
}

// FindRestorePoint 查询指定时间之前的最近恢复点，返回status=3且create_time<=beforeTime的最新记录。
func (r *BackupRepoImpl) FindRestorePoint(ctx context.Context, beforeTime int64) (*domainBackup.BackupTask, error) {
	if r.db == nil {
		return nil, nil
	}

	var backupID int64
	var backupType, status int
	var backupPath string
	var fileSize, createTime int64

	err := r.db.QueryRowContext(ctx,
		`SELECT id, backup_type, status, backup_path, file_size, create_time FROM `+tables.TBackupTask+
			` WHERE status = 3 AND create_time <= ? ORDER BY create_time DESC LIMIT 1`,
		beforeTime,
	).Scan(&backupID, &backupType, &status, &backupPath, &fileSize, &createTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询恢复点失败: %w", err)
	}

	r.logger.Debug("查询恢复点成功",
		zap.Int64("backup_id", backupID),
		zap.Int64("create_time", createTime),
	)
	return domainBackup.RestoreBackupTask(backupID, backupType, status, createTime, createTime), nil
}
