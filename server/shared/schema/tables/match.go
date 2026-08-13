// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// Match服务数据库表名常量（规范2），t_前缀+蛇形+单数。
const (
	// TMatchTicket 匹配票表，存储匹配队列与等待状态
	TMatchTicket = "t_match_ticket"
	// TBattlefield 战场表，存储限时战场状态与参与方
	TBattlefield = "t_battlefield"
	// TRankTier 排行榜表，存储跨服排行榜排名数据
	TRankTier = "t_rank_tier"
)
