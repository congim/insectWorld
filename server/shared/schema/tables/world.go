// Package tables 统一表名常量定义，全服务端表名单一真相源。
//
// 本包按服务分文件定义全服务端数据库表名常量，所有表名遵循t_前缀+蛇形+单数规范（规范2）。
// 各服务infrastructure/persistence/tables.go通过常量别名引用本包，消除表名常量散落定义。
// 表名常量集中管理确保表名单一真相源，DDL变更与代码引用同步。
package tables

// World服务数据库表名常量（规范2），t_前缀+蛇形+单数。
const (
	// TMapCell 地图格子表，存储每个格子的地形与实体信息
	TMapCell = "t_map_cell"
	// TMovementOrder 移动订单表，存储移动订单状态与路径
	TMovementOrder = "t_movement_order"
	// TRegion 区域表，存储区域定义与格子范围
	TRegion = "t_region"
	// TTeleportRecord 传送记录表，存储传送历史与冷却记录
	TTeleportRecord = "t_teleport_record"
)