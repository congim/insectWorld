// Package persistence Combat服务持久化层，提供聚合根的仓储实现。
// 本文件通过常量别名引用shared/schema/tables统一表名定义（规范2），消除表名常量散落。
package persistence

import "insectworld/server/shared/schema/tables"

// 数据库表名常量，引用shared/schema/tables统一定义，t_前缀+蛇形+单数（规范2）。
const (
	// TableCombat 战斗表，存储战斗状态与参战方信息
	TableCombat = tables.TCombat
	// TableCombatRound 战斗轮次表，存储每轮战斗结果
	TableCombatRound = tables.TCombatRound
	// TableCombatReport 战报表，存储战报详情
	TableCombatReport = tables.TCombatReport
	// TableSkillCooldown 技能冷却表，存储技能冷却记录
	TableSkillCooldown = tables.TSkillCooldown
)
