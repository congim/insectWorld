// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// Config服务数据库表名常量（规范2），t_前缀+蛇形+单数。
const (
	// TConfigVersion 配置版本表，存储配置版本历史与发布记录
	TConfigVersion = "t_config_version"
	// TConfigSnapshot 配置快照表，存储配置版本快照内容
	TConfigSnapshot = "t_config_snapshot"
	// TConfigAuditLog 配置审计日志表，存储配置变更的操作审计记录
	TConfigAuditLog = "t_config_audit_log"
)
