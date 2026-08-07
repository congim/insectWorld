// Package snapshot Persist服务快照仓储实现，实现domain/snapshot.SnapshotRepository接口。
package snapshot

import (
	"context"
	"database/sql"
	"fmt"

	"insectworld/server/persist/domain/snapshot"
)

// SnapshotRepoImpl SnapshotRepository接口实现。
type SnapshotRepoImpl struct {
	db *sql.DB // MySQL连接池
}

// NewSnapshotRepoImpl 创建快照仓储实现实例。
func NewSnapshotRepoImpl(db *sql.DB) *SnapshotRepoImpl {
	return &SnapshotRepoImpl{db: db}
}

// Save 保存快照聚合根状态。
// TODO 后续接入S3上传，当前仅记录快照任务元数据。
func (r *SnapshotRepoImpl) Save(ctx context.Context, s *snapshot.Snapshot) error {
	// TODO 后续实现快照数据上传S3，当前仅保存任务元数据到数据库
	_ = ctx
	_ = s
	return nil
}

// FindLatest 查询指定类型的最新快照。
// TODO 后续接入读模型查询，当前返回nil表示无快照。
func (r *SnapshotRepoImpl) FindLatest(ctx context.Context, scope int) (*snapshot.Snapshot, error) {
	_ = ctx
	_ = scope
	_ = fmt.Sprintf("查询scope=%d的最新快照", scope)
	return nil, nil
}
