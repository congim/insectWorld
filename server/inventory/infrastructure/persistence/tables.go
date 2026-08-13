// Package persistence Inventory服务持久化层，提供聚合根的仓储实现。
// 本文件通过常量别名引用shared/schema/tables统一表名定义（规范2），消除表名常量散落。
package persistence

import "insectworld/server/shared/schema/tables"

// 数据库表名常量，引用shared/schema/tables统一定义，t_前缀+蛇形+单数（规范2）。
const (
	// TableInventory 背包表，存储玩家背包容量与状态
	TableInventory = tables.TInventory
	// TableItem 道具实例表，存储玩家持有的道具实例
	TableItem = tables.TItem
	// TableItemUsage 道具使用订单表，存储道具使用流程记录
	TableItemUsage = tables.TItemUsage
	// TableInventoryRead 背包读模型表，CQRS读侧投影
	TableInventoryRead = tables.TInventoryRead
)
