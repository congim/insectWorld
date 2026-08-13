// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// Gateway服务数据库表名常量（规范2），t_前缀+蛇形+单数。
const (
	// TPlayerAccount 玩家账号档案表，存储注册账号、密码哈希、封禁状态等
	TPlayerAccount = "t_player_account"
	// TAuthAuditLog 认证审计日志表，存储注册/登录/登出/封禁等操作审计记录
	TAuthAuditLog = "t_auth_audit_log"
)
