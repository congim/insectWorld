// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// Operation服务数据库表名常量（规范2），t_前缀+蛇形+单数。
const (
	// TSeason 赛季表，存储赛季基本信息与时间范围
	TSeason = "t_season"
	// TSeasonPhase 赛季阶段表，存储赛季各阶段的配置与状态
	TSeasonPhase = "t_season_phase"
	// TScoreBoard 排行榜表，存储赛季排行榜数据
	TScoreBoard = "t_score_board"
	// TGameEvent 游戏事件表，存储赛季中的游戏事件记录
	TGameEvent = "t_game_event"
	// TSeasonSnapshot 赛季快照表，存储赛季结束时的数据快照
	TSeasonSnapshot = "t_season_snapshot"
)
