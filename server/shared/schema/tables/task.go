// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// Task服务数据库表名常量（规范2），t_前缀+蛇形+单数。
const (
	// TTaskProgress 任务进度表，存储玩家任务进度与领取状态
	TTaskProgress = "t_task_progress"
	// TAchievement 成就表，存储玩家成就达成与领取状态
	TAchievement = "t_achievement"
	// TTaskProgressRead 任务进度读模型表，CQRS读侧投影
	TTaskProgressRead = "t_task_progress_read"
)
