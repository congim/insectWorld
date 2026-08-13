// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// 跨服务共享表名常量（规范2），t_前缀+蛇形+单数。
// 这些表被多个服务共同引用，不属于单一服务。
const (
	// TOutbox Outbox表，存储待投递的领域事件，各服务通过Outbox模式保证事件可靠投递
	TOutbox = "t_outbox"
	// TPlayerArchive 玩家归档表，存储玩家冷数据归档，由Persist服务管理
	TPlayerArchive = "t_player_archive"
)
