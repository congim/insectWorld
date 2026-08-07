// Package persistence Operation服务持久化层。
// 本文件通过常量别名引用shared/schema/tables统一表名定义（规范2），消除表名常量散落。
package persistence

import "insectworld/server/shared/schema/tables"

const (
	TableSeason         = tables.TSeason         // 赛季表
	TableSeasonPhase    = tables.TSeasonPhase    // 赛季阶段表
	TableScoreBoard     = tables.TScoreBoard     // 排行榜表
	TableGameEvent      = tables.TGameEvent      // 游戏事件表
	TableSeasonSnapshot = tables.TSeasonSnapshot // 赛季快照表
)
