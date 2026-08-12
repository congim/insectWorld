// Package archive Persist服务归档仓储实现，实现domain/archive.ArchiveRepository接口。
//
// infrastructure层技术适配，将归档任务聚合根状态持久化到t_archive_task表。
// 表名引用shared/schema/tables常量（规范2），不硬编码表名字符串。
package archive

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"

	domainArchive "insectworld/server/persist/domain/archive"
	"insectworld/server/shared/schema/tables"
)

// ArchiveRepoImpl ArchiveRepository接口实现。
type ArchiveRepoImpl struct {
	db     *sql.DB     // MySQL连接池
	logger *zap.Logger // 结构化日志
}

// NewArchiveRepoImpl 创建归档仓储实现实例。
func NewArchiveRepoImpl(db *sql.DB, logger *zap.Logger) *ArchiveRepoImpl {
	return &ArchiveRepoImpl{db: db, logger: logger}
}

// Save 保存归档任务状态，表名引用shared/schema/tables.TArchiveTask常量（规范2）。
// 将聚合根字段映射到DDL列：ruleID→source_table，sourceType+targetStorage→archive_condition。
func (r *ArchiveRepoImpl) Save(ctx context.Context, task *domainArchive.ArchiveTask) error {
	if r.db == nil {
		return fmt.Errorf("数据库未初始化，无法保存归档任务")
	}

	archiveCondition := fmt.Sprintf("source_type=%d,target_storage=%d", task.SourceType(), task.TargetStorage())
	now := time.Now().UnixMilli()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO `+tables.TArchiveTask+` (id, source_table, archive_condition, status, archived_count, create_time) VALUES (?, ?, ?, ?, 0, ?)`,
		task.TaskID(), task.RuleID(), archiveCondition, task.Status(), now,
	)
	if err != nil {
		return fmt.Errorf("保存归档任务失败: %w", err)
	}

	r.logger.Info("归档任务保存成功",
		zap.Int64("task_id", task.TaskID()),
		zap.String("rule_id", task.RuleID()),
		zap.Int("status", task.Status()),
	)
	return nil
}

// FindPending 查询待执行的归档任务，返回status=1的记录，按创建时间升序排列。
func (r *ArchiveRepoImpl) FindPending(ctx context.Context, limit int) ([]*domainArchive.ArchiveTask, error) {
	if r.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, source_table, archive_condition, status, create_time FROM `+tables.TArchiveTask+
			` WHERE status = 1 ORDER BY create_time ASC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("查询待执行归档任务失败: %w", err)
	}
	defer rows.Close()

	var tasks []*domainArchive.ArchiveTask
	for rows.Next() {
		var taskID int64
		var ruleID, archiveCondition string
		var status, createTime int64

		if err := rows.Scan(&taskID, &ruleID, &archiveCondition, &status, &createTime); err != nil {
			return nil, fmt.Errorf("读取归档任务记录失败: %w", err)
		}

		tasks = append(tasks, domainArchive.RestoreArchiveTask(taskID, ruleID, 0, 0, int(status), 0))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历归档任务记录失败: %w", err)
	}

	r.logger.Debug("查询待执行归档任务", zap.Int("count", len(tasks)))
	return tasks, nil
}
