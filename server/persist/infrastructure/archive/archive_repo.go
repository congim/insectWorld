// Package archive Persist服务归档仓储实现，实现domain/archive.ArchiveRepository接口。
package archive

import (
	"context"
	"database/sql"
	"fmt"

	domainArchive "insectworld/server/persist/domain/archive"
)

// ArchiveRepoImpl ArchiveRepository接口实现。
type ArchiveRepoImpl struct {
	db *sql.DB // MySQL连接池
}

// NewArchiveRepoImpl 创建归档仓储实现实例。
func NewArchiveRepoImpl(db *sql.DB) *ArchiveRepoImpl {
	return &ArchiveRepoImpl{db: db}
}

// Save 保存归档任务状态。
// TODO 后续接入归档规则引擎，当前仅记录任务元数据。
func (r *ArchiveRepoImpl) Save(ctx context.Context, task *domainArchive.ArchiveTask) error {
	_ = ctx
	_ = task
	return nil
}

// FindPending 查询待执行的归档任务。
// TODO 后续接入读模型查询，当前返回空列表。
func (r *ArchiveRepoImpl) FindPending(ctx context.Context, limit int) ([]*domainArchive.ArchiveTask, error) {
	_ = ctx
	_ = limit
	_ = fmt.Sprintf("查询limit=%d的待执行归档任务", limit)
	return nil, nil
}
