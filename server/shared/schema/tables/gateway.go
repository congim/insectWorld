// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// Gateway服务数据库表名常量（规范2），t_前缀+蛇形+单数。
const (
	// TSession 会话表，存储玩家在线会话信息
	TSession = "t_session"
	// TConnectionRecord 连接记录表，存储玩家连接历史
	TConnectionRecord = "t_connection_record"
	// TRouteTable 路由表，存储请求路由配置
	TRouteTable = "t_route_table"
	// TPlayerAccount 玩家账号档案表，存储注册账号、密码哈希、封禁状态等
	TPlayerAccount = "t_player_account"
	// TAuthAuditLog 认证审计日志表，存储注册/登录/登出/封禁等操作审计记录
	TAuthAuditLog = "t_auth_audit_log"
)
