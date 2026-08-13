// Package tables 统一表名常量定义，全服务端表名单一真相源。
package tables

// Combat服务数据库表名常量（规范2），t_前缀+蛇形+单数。
const (
	// TCombat 战斗表，存储战斗状态与参战方信息
	TCombat = "t_combat"
	// TCombatRound 战斗轮次表，存储每轮战斗结果
	TCombatRound = "t_combat_round"
	// TCombatReport 战报表，存储战报详情
	TCombatReport = "t_combat_report"
	// TSkillCooldown 技能冷却表，存储技能冷却记录
	TSkillCooldown = "t_skill_cooldown"
)
