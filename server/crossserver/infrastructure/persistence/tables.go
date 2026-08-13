// Package persistence CrossServer服务持久化层，提供聚合根的仓储实现。
// 本文件通过常量别名引用shared/schema/tables统一表名定义（规范2），消除表名常量散落。
package persistence

import "insectworld/server/shared/schema/tables"

// 数据库表名常量，引用shared/schema/tables统一定义，t_前缀+蛇形+单数（规范2）。
const (
	// TableServerNode 游戏服节点表，存储节点注册与状态信息
	TableServerNode = tables.TServerNode
	// TableCrossServerActivity 跨服活动表，存储跨服活动生命周期数据
	TableCrossServerActivity = tables.TCrossServerActivity
	// TableMergeTask 合服任务表，存储合服迁移进度与状态
	TableMergeTask = tables.TMergeTask
)
