// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// Inventory服务数据库表名常量（规范2），t_前缀+蛇形+单数。
const (
	// TInventory 背包表，存储玩家背包容量与状态
	TInventory = "t_inventory"
	// TItem 道具实例表，存储玩家持有的道具实例
	TItem = "t_item"
	// TItemUsage 道具使用订单表，存储道具使用流程记录
	TItemUsage = "t_item_usage"
	// TInventoryRead 背包读模型表，CQRS读侧投影
	TInventoryRead = "t_inventory_read"
)
