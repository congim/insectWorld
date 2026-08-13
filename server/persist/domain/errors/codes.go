// Package errors Persist服务错误码定义，集中管理服务内部错误码。
package errors

// Persist服务错误码常量，使用显式整型赋值，区间划分。
// 错误码区间：30000-30999 Persist服务专用
const (
	// ErrMigrationFailed 迁移执行失败，迁移脚本执行出错
	ErrMigrationFailed = 30001
	// ErrSnapshotFailed 快照执行失败，快照任务执行出错
	ErrSnapshotFailed = 30002
	// ErrArchiveFailed 归档执行失败，归档任务执行出错
	ErrArchiveFailed = 30003
	// ErrBackupFailed 备份执行失败，备份任务执行出错
	ErrBackupFailed = 30004
	// ErrRestoreFailed 恢复执行失败，数据恢复操作出错
	ErrRestoreFailed = 30005
	// ErrDatasourceConnFailed 数据源连接失败
	ErrDatasourceConnFailed = 30006
	// ErrMigrationVersionConflict 迁移版本冲突，重复执行已执行的版本
	ErrMigrationVersionConflict = 30007
)
