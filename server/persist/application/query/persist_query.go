// Package query Persist服务应用层查询处理，走CQRS读模型提供数据治理查询能力。
package query

import (
	"context"

	"go.uber.org/zap"

	"insectworld/server/persist/domain/migration"
	"insectworld/server/persist/domain/snapshot"
)

// SnapshotQueryHandler 快照查询处理器。
type SnapshotQueryHandler struct {
	snapshotRepo snapshot.SnapshotRepository // 快照仓储，提供最新快照读模型
	logger       *zap.Logger                 // 结构化日志器，记录快照查询异常
}

// NewSnapshotQueryHandler 创建快照查询处理器实例。
func NewSnapshotQueryHandler(
	snapshotRepo snapshot.SnapshotRepository,
	logger *zap.Logger,
) *SnapshotQueryHandler {
	return &SnapshotQueryHandler{
		snapshotRepo: snapshotRepo,
		logger:       logger,
	}
}

// Query 查询指定类型的最新快照。
func (h *SnapshotQueryHandler) Query(ctx context.Context, scope int) (*snapshot.Snapshot, error) {
	s, err := h.snapshotRepo.FindLatest(ctx, scope)
	if err != nil {
		h.logger.Error("查询最新快照失败", zap.Int("scope", scope), zap.Error(err))
		return nil, err
	}
	return s, nil
}

// MigrationStatusQueryHandler 迁移状态查询处理器。
type MigrationStatusQueryHandler struct {
	migrationRepo migration.MigrationRepository // 迁移仓储，提供已执行版本读模型
	logger        *zap.Logger                   // 结构化日志器，记录迁移查询异常
}

// NewMigrationStatusQueryHandler 创建迁移状态查询处理器实例。
func NewMigrationStatusQueryHandler(
	migrationRepo migration.MigrationRepository,
	logger *zap.Logger,
) *MigrationStatusQueryHandler {
	return &MigrationStatusQueryHandler{
		migrationRepo: migrationRepo,
		logger:        logger,
	}
}

// Query 查询已执行的迁移版本号列表。
func (h *MigrationStatusQueryHandler) Query(ctx context.Context) ([]int64, error) {
	versions, err := h.migrationRepo.FindExecutedVersions(ctx)
	if err != nil {
		h.logger.Error("查询已执行迁移版本失败", zap.Error(err))
		return nil, err
	}
	return versions, nil
}
