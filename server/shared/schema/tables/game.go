// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// Game模块Growth上下文表名常量。
const (
	// TPlayerProfile 玩家成长档案表
	TPlayerProfile = "t_player_profile"
	// TPlayerBuilding 玩家建筑表
	TPlayerBuilding = "t_player_building"
	// TTrainingTask 单位训练任务表
	TTrainingTask = "t_training_task"
	// TUnitRoster 玩家已训练单位名册表
	TUnitRoster = "t_unit_roster"
	// TUnitGrantOperation 单位入账幂等操作表
	TUnitGrantOperation = "t_unit_grant_operation"
)
