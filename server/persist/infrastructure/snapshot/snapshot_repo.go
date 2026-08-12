// Package snapshot Persist服务快照仓储实现，实现domain/snapshot.SnapshotRepository接口。
//
// infrastructure层技术适配，将快照聚合根状态持久化到t_snapshot_task表。
// 表名引用shared/schema/tables常量（规范2），不硬编码表名字符串。
package snapshot

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/persist/domain/snapshot"
	"insectworld/server/persist/domain/vo"
	"insectworld/server/shared/schema/tables"
)

// SnapshotRepoImpl SnapshotRepository接口实现。
type SnapshotRepoImpl struct {
	db     *sql.DB     // MySQL连接池
	logger *zap.Logger // 结构化日志
}

// NewSnapshotRepoImpl 创建快照仓储实现实例。
func NewSnapshotRepoImpl(db *sql.DB, logger *zap.Logger) *SnapshotRepoImpl {
	return &SnapshotRepoImpl{db: db, logger: logger}
}

// Save 保存快照聚合根状态，表名引用shared/schema/tables.TSnapshotTask常量（规范2）。
// target_table用空字符串占位，实际目标表在快照执行时确定。
func (r *SnapshotRepoImpl) Save(ctx context.Context, s *snapshot.Snapshot) error {
	if r.db == nil {
		return fmt.Errorf("数据库未初始化，无法保存快照任务")
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO `+tables.TSnapshotTask+` (id, task_type, target_table, status, start_time, end_time, create_time) VALUES (?, ?, '', ?, ?, ?, ?)`,
		s.SnapshotID(), s.Scope(), s.Status(), s.CreateTime(), s.ExecuteTime(), s.CreateTime(),
	)
	if err != nil {
		return fmt.Errorf("保存快照任务失败: %w", err)
	}

	r.logger.Info("快照任务保存成功",
		zap.Int64("snapshot_id", s.SnapshotID()),
		zap.Int("scope", s.Scope()),
		zap.Int("status", s.Status()),
	)
	return nil
}

// FindLatest 查询指定类型的最新快照，按创建时间降序返回第一条。
func (r *SnapshotRepoImpl) FindLatest(ctx context.Context, scope int) (*snapshot.Snapshot, error) {
	if r.db == nil {
		return nil, nil
	}

	var snapshotID int64
	var taskType, status int
	var targetTable string
	var startTime, endTime, createTime int64

	err := r.db.QueryRowContext(ctx,
		`SELECT id, task_type, target_table, status, start_time, end_time, create_time FROM `+tables.TSnapshotTask+
			` WHERE task_type = ? ORDER BY create_time DESC LIMIT 1`,
		scope,
	).Scan(&snapshotID, &taskType, &status, &targetTable, &startTime, &endTime, &createTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询最新快照失败: %w", err)
	}

	r.logger.Debug("查询最新快照成功",
		zap.Int64("snapshot_id", snapshotID),
		zap.Int("scope", scope),
	)
	return snapshot.RestoreSnapshot(snapshotID, taskType, status, createTime, endTime, vo.SnapshotRange{}), nil
}
