// Package persistence World服务持久化层，提供聚合根的仓储实现。
// 本文件通过常量别名引用shared/schema/tables统一表名定义（规范2），消除表名常量散落。
package persistence

import "insectworld/server/shared/schema/tables"

// 数据库表名常量，引用shared/schema/tables统一定义，t_前缀+蛇形+单数（规范2）。
const (
	// TableMapCell 地图格子表，存储每个格子的地形与实体信息
	TableMapCell = tables.TMapCell
	// TableMovementOrder 移动订单表，存储移动订单状态与路径
	TableMovementOrder = tables.TMovementOrder
	// TableRegion 区域表，存储区域定义与格子范围
	TableRegion = tables.TRegion
	// TableTeleportRecord 传送记录表，存储传送历史与冷却记录
	TableTeleportRecord = tables.TTeleportRecord
)
