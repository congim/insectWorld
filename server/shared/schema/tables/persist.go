// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// Persist服务数据库表名常量（规范2），t_前缀+蛇形+单数。
// Persist服务负责数据治理、快照归档、迁移脚本执行，以下表存储其管理元数据。
const (
	// TSchemaMigration Schema迁移记录表，存储DDL迁移脚本的执行版本与状态
	TSchemaMigration = "t_schema_migration"
	// TSnapshotTask 快照任务表，存储数据快照任务的配置与执行记录
	TSnapshotTask = "t_snapshot_task"
	// TArchiveTask 归档任务表，存储冷数据归档任务的配置与执行记录
	TArchiveTask = "t_archive_task"
	// TBackupTask 备份任务表，存储数据备份任务的配置与执行记录
	TBackupTask = "t_backup_task"
)