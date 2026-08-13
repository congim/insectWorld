// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// CrossServer服务数据库表名常量（规范2），t_前缀+蛇形+单数。
const (
	// TServerNode 游戏服节点表，存储节点注册与状态信息
	TServerNode = "t_server_node"
	// TCrossServerActivity 跨服活动表，存储跨服活动生命周期数据
	TCrossServerActivity = "t_cross_server_activity"
	// TMergeTask 合服任务表，存储合服迁移进度与状态
	TMergeTask = "t_merge_task"
)
