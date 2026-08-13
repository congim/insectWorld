// Package grpc Persist服务gRPC管理面，提供备份/迁移/恢复/快照查询的gRPC接口。
//
// interfaces层依赖application层，不直接依赖infrastructure（规范3）。
package grpc

import (
	"context"

	"go.uber.org/zap"

	persistCmd "insectworld/server/persist/application/command"
)

// Handler Persist服务gRPC管理面handler，注入application层command/query。
type Handler struct {
	createSnapshotHandler   *persistCmd.CreateSnapshotHandler   // 创建快照用例处理器
	executeMigrationHandler *persistCmd.ExecuteMigrationHandler // 执行数据迁移用例处理器
	archiveColdDataHandler  *persistCmd.ArchiveColdDataHandler  // 归档冷数据用例处理器
	createBackupHandler     *persistCmd.CreateBackupHandler     // 创建备份用例处理器
	logger                  *zap.Logger                         // 结构化日志器，记录接口层请求结果
}

// NewHandler 创建gRPC管理面handler实例。
func NewHandler(
	createSnapshotHandler *persistCmd.CreateSnapshotHandler,
	executeMigrationHandler *persistCmd.ExecuteMigrationHandler,
	archiveColdDataHandler *persistCmd.ArchiveColdDataHandler,
	createBackupHandler *persistCmd.CreateBackupHandler,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		createSnapshotHandler:   createSnapshotHandler,
		executeMigrationHandler: executeMigrationHandler,
		archiveColdDataHandler:  archiveColdDataHandler,
		createBackupHandler:     createBackupHandler,
		logger:                  logger,
	}
}

// HandleCreateSnapshot 处理创建快照gRPC请求。
func (h *Handler) HandleCreateSnapshot(ctx context.Context, snapshotID int64, scope int) error {
	cmd := persistCmd.CreateSnapshotCommand{
		SnapshotID: snapshotID,
		Scope:      scope,
	}
	return h.createSnapshotHandler.Handle(ctx, cmd)
}

// HandleExecuteMigration 处理执行迁移gRPC请求。
func (h *Handler) HandleExecuteMigration(ctx context.Context, version int64, scriptName string) error {
	cmd := persistCmd.ExecuteMigrationCommand{
		Version:    version,
		ScriptName: scriptName,
	}
	return h.executeMigrationHandler.Handle(ctx, cmd)
}

// HandleArchiveColdData 处理归档冷数据gRPC请求。
func (h *Handler) HandleArchiveColdData(ctx context.Context, taskID int64, ruleID string, sourceType, targetStorage int) error {
	cmd := persistCmd.ArchiveColdDataCommand{
		TaskID:        taskID,
		RuleID:        ruleID,
		SourceType:    sourceType,
		TargetStorage: targetStorage,
	}
	return h.archiveColdDataHandler.Handle(ctx, cmd)
}

// HandleCreateBackup 处理创建备份gRPC请求。
func (h *Handler) HandleCreateBackup(ctx context.Context, backupID int64, backupType int) error {
	cmd := persistCmd.CreateBackupCommand{
		BackupID:   backupID,
		BackupType: backupType,
	}
	return h.createBackupHandler.Handle(ctx, cmd)
}
