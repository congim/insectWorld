// Package command Persist服务应用层命令处理，编排聚合根与仓储完成数据治理操作。
//
// command handler注入Repository接口（domain层声明），不直接依赖infrastructure实现，
// 保证application层可独立单测（规范3 DDD可测性）。
package command

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"insectworld/server/persist/domain/archive"
	"insectworld/server/persist/domain/backup"
	"insectworld/server/persist/domain/migration"
	"insectworld/server/persist/domain/snapshot"
	"insectworld/server/persist/domain/vo"
)

// CreateSnapshotHandler 创建快照命令处理器。
type CreateSnapshotHandler struct {
	snapshotRepo snapshot.SnapshotRepository
	logger       *zap.Logger
}

// NewCreateSnapshotHandler 创建快照命令处理器实例。
func NewCreateSnapshotHandler(
	snapshotRepo snapshot.SnapshotRepository,
	logger *zap.Logger,
) *CreateSnapshotHandler {
	return &CreateSnapshotHandler{
		snapshotRepo: snapshotRepo,
		logger:       logger,
	}
}

// CreateSnapshotCommand 创建快照命令参数。
type CreateSnapshotCommand struct {
	SnapshotID int64 // 快照ID
	Scope      int   // 快照类型：1=全量 2=增量
	DataRange  vo.SnapshotRange // 快照数据范围
}

// Handle 执行创建快照命令。
func (h *CreateSnapshotHandler) Handle(ctx context.Context, cmd CreateSnapshotCommand) error {
	now := time.Now().UnixMilli()
	s := snapshot.NewSnapshot(cmd.SnapshotID, cmd.Scope, now, cmd.DataRange)

	if err := s.Start(); err != nil {
		h.logger.Error("快照任务启动失败", zap.Int64("snapshot_id", cmd.SnapshotID), zap.Error(err))
		return fmt.Errorf("快照任务启动失败: %w", err)
	}

	if err := h.snapshotRepo.Save(ctx, s); err != nil {
		h.logger.Error("快照保存失败", zap.Int64("snapshot_id", cmd.SnapshotID), zap.Error(err))
		return fmt.Errorf("快照保存失败: %w", err)
	}

	h.logger.Info("快照任务创建成功",
		zap.Int64("snapshot_id", cmd.SnapshotID),
		zap.Int("scope", cmd.Scope),
	)
	return nil
}

// ExecuteMigrationHandler 执行迁移命令处理器。
type ExecuteMigrationHandler struct {
	migrationRepo migration.MigrationRepository
	logger        *zap.Logger
}

// NewExecuteMigrationHandler 创建迁移命令处理器实例。
func NewExecuteMigrationHandler(
	migrationRepo migration.MigrationRepository,
	logger *zap.Logger,
) *ExecuteMigrationHandler {
	return &ExecuteMigrationHandler{
		migrationRepo: migrationRepo,
		logger:        logger,
	}
}

// ExecuteMigrationCommand 执行迁移命令参数。
type ExecuteMigrationCommand struct {
	Version    int64  // 迁移版本号
	ScriptName string // 迁移脚本文件名
}

// Handle 执行迁移命令。
func (h *ExecuteMigrationHandler) Handle(ctx context.Context, cmd ExecuteMigrationCommand) error {
	record := migration.NewMigrationRecord(cmd.Version, cmd.ScriptName)
	now := time.Now().UnixMilli()

	if err := record.MarkExecuted(now); err != nil {
		h.logger.Error("迁移执行失败", zap.Int64("version", cmd.Version), zap.Error(err))
		return fmt.Errorf("迁移执行失败: %w", err)
	}

	if err := h.migrationRepo.SaveRecord(ctx, record); err != nil {
		h.logger.Error("迁移记录保存失败", zap.Int64("version", cmd.Version), zap.Error(err))
		return fmt.Errorf("迁移记录保存失败: %w", err)
	}

	h.logger.Info("迁移执行成功", zap.Int64("version", cmd.Version), zap.String("script", cmd.ScriptName))
	return nil
}

// ArchiveColdDataHandler 归档冷数据命令处理器。
type ArchiveColdDataHandler struct {
	archiveRepo archive.ArchiveRepository
	logger      *zap.Logger
}

// NewArchiveColdDataHandler 创建归档命令处理器实例。
func NewArchiveColdDataHandler(
	archiveRepo archive.ArchiveRepository,
	logger *zap.Logger,
) *ArchiveColdDataHandler {
	return &ArchiveColdDataHandler{
		archiveRepo: archiveRepo,
		logger:      logger,
	}
}

// ArchiveColdDataCommand 归档冷数据命令参数。
type ArchiveColdDataCommand struct {
	TaskID        int64  // 归档任务ID
	RuleID        string // 归档规则ID
	SourceType    int    // 源数据类型
	TargetStorage int    // 目标存储类型
}

// Handle 执行归档冷数据命令。
func (h *ArchiveColdDataHandler) Handle(ctx context.Context, cmd ArchiveColdDataCommand) error {
	task := archive.NewArchiveTask(cmd.TaskID, cmd.RuleID, cmd.SourceType, cmd.TargetStorage)

	if err := task.Start(); err != nil {
		h.logger.Error("归档任务启动失败", zap.Int64("task_id", cmd.TaskID), zap.Error(err))
		return fmt.Errorf("归档任务启动失败: %w", err)
	}

	if err := h.archiveRepo.Save(ctx, task); err != nil {
		h.logger.Error("归档任务保存失败", zap.Int64("task_id", cmd.TaskID), zap.Error(err))
		return fmt.Errorf("归档任务保存失败: %w", err)
	}

	h.logger.Info("归档任务创建成功",
		zap.Int64("task_id", cmd.TaskID),
		zap.String("rule_id", cmd.RuleID),
		zap.Int("source_type", cmd.SourceType),
	)
	return nil
}

// CreateBackupHandler 创建备份命令处理器。
type CreateBackupHandler struct {
	backupRepo backup.BackupRepository
	logger     *zap.Logger
}

// NewCreateBackupHandler 创建备份命令处理器实例。
func NewCreateBackupHandler(
	backupRepo backup.BackupRepository,
	logger *zap.Logger,
) *CreateBackupHandler {
	return &CreateBackupHandler{
		backupRepo: backupRepo,
		logger:     logger,
	}
}

// CreateBackupCommand 创建备份命令参数。
type CreateBackupCommand struct {
	BackupID   int64 // 备份任务ID
	BackupType int   // 备份类型：1=全量 2=增量 3=日志
}

// Handle 执行创建备份命令。
func (h *CreateBackupHandler) Handle(ctx context.Context, cmd CreateBackupCommand) error {
	now := time.Now().UnixMilli()
	task := backup.NewBackupTask(cmd.BackupID, cmd.BackupType, now)

	if err := task.Start(); err != nil {
		h.logger.Error("备份任务启动失败", zap.Int64("backup_id", cmd.BackupID), zap.Error(err))
		return fmt.Errorf("备份任务启动失败: %w", err)
	}

	if err := h.backupRepo.Save(ctx, task); err != nil {
		h.logger.Error("备份任务保存失败", zap.Int64("backup_id", cmd.BackupID), zap.Error(err))
		return fmt.Errorf("备份任务保存失败: %w", err)
	}

	h.logger.Info("备份任务创建成功",
		zap.Int64("backup_id", cmd.BackupID),
		zap.Int("backup_type", cmd.BackupType),
	)
	return nil
}