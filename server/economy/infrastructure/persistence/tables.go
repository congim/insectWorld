// Package persistence Economy服务持久化层。
// 本文件通过常量别名引用shared/schema/tables统一表名定义（规范2），消除表名常量散落。
package persistence

import "insectworld/server/shared/schema/tables"

// 数据库表名常量，引用shared/schema/tables统一定义，t_前缀+蛇形+单数（规范2）。
const (
	TableResourceBalance = tables.TResourceBalance // 资源余额表
	TableProductionLine  = tables.TProductionLine  // 生产线表
	TableTradeOrder      = tables.TTradeOrder      // 交易订单表
	TableConversionOrder = tables.TConversionOrder // 转换订单表
)
