// Package persistence Task服务持久化层，提供聚合根的仓储实现。
package persistence

import "insectworld/server/shared/schema/tables"

// 数据库表名常量，引用shared/schema/tables统一定义，t_前缀+蛇形+单数（规范2）。
const (
	// TableTaskProgress 任务进度表，存储玩家任务进度与领取状态
	TableTaskProgress = tables.TTaskProgress
	// TableAchievement 成就表，存储玩家成就达成与领取状态
	TableAchievement = tables.TAchievement
	// TableTaskProgressRead 任务进度读模型表，CQRS读侧投影
	TableTaskProgressRead = tables.TTaskProgressRead
)
