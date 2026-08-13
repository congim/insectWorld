// Package persistence Match服务持久化层，提供聚合根的仓储实现。
// 本文件通过常量别名引用shared/schema/tables统一表名定义（规范2），消除表名常量散落。
package persistence

import "insectworld/server/shared/schema/tables"

// 数据库表名常量，引用shared/schema/tables统一定义，t_前缀+蛇形+单数（规范2）。
const (
	// TableMatchTicket 匹配票表，存储匹配队列与等待状态
	TableMatchTicket = tables.TMatchTicket
	// TableBattlefield 战场表，存储限时战场状态与参与方
	TableBattlefield = tables.TBattlefield
	// TableRankTier 排行榜表，存储跨服排行榜排名数据
	TableRankTier = tables.TRankTier
)
