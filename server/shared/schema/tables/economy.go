// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// Economy服务数据库表名常量（规范2），t_前缀+蛇形+单数。
const (
	// TResourceBalance 资源余额表，存储玩家各资源类型的余额
	TResourceBalance = "t_resource_balance"
	// TProductionLine 生产线表，存储资源产出线的配置与状态
	TProductionLine = "t_production_line"
	// TTradeOrder 交易订单表，存储玩家间交易订单
	TTradeOrder = "t_trade_order"
	// TConversionOrder 转换订单表，存储资源转换订单
	TConversionOrder = "t_conversion_order"
)
