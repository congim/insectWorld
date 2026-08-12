// Package combat 战斗聚合根，维护战斗状态与轮次执行。
// 本文件定义战斗快照值对象（ADR-004 3.1）：冻结开战时参战方属性与配置引用版本。
// 战斗一经开始，其配置引用域被冻结到开战版本——所有战斗期配置读取走版本化查询GetWithVersion(snapshot.ConfigVersion)。
package combat

// CombatSnapshot 战斗快照，冻结开战时参战方属性与配置版本（ADR-004 3.1）。
// 配置引用域（公式/技能/克制矩阵/战利品/战斗类型）以ConfigVersion为唯一基准，
// 战斗期配置查询一律走版本化查询，与当前热更版本解耦；热更/回滚不改变进行中战斗的配置基准。
type CombatSnapshot struct {
	ConfigVersion    int64               // 开战时配置版本号，战斗全程引用该版本，不因热更/回滚改变（默认0=未绑定）
	CombatType       int                 // 战斗类型：1=野战 2=攻城 3=城战，开战时冻结，对应快照版本combat.json
	FormulaID        string              // 本场伤害公式ID，开战时冻结，对应快照版本formulas.json；空表示未绑定公式
	CounterMatrixVer int64               // 克制矩阵版本，与ConfigVersion一致（克制矩阵随配置包整体替换，无独立版本号）
	LootRuleID       string              // 战利品规则ID，开战时冻结，对应快照版本combat_loot_rules；空表示无战利品规则
	SkillIDs         []string            // 本场启用技能ID列表，开战时冻结，对应快照版本combat_skills
	AttackerProps    map[int64]PropEntry // 攻击方实体属性快照，key=实体ID；长时战斗按spec.md功能10每轮刷新
	DefenderProps    map[int64]PropEntry // 防守方实体属性快照，key=实体ID
}

// PropEntry 参战方单实体属性快照项（ADR-004 3.1）。
// 兵种类型对应快照版本units.json；运营强制发布删除类热更删除该兵种后结算即触发熔断。
type PropEntry struct {
	EntityID int64               // 实体ID，对应World中的实体
	Atk      int64               // 攻击力（int64，AGENTS.md规范8）
	Def      int64               // 防御力
	HP       int64               // 当前血量
	UnitType int                 // 兵种类型ID，对应快照版本units.json
	Tags     map[string]struct{} // 实体标签（克制矩阵查询用），开战时冻结
}

// NewPropEntry 创建单实体属性快照项。
// entityID为实体ID，atk/def/hp为攻击/防御/血量，unitType为兵种类型ID，tags为实体标签。
func NewPropEntry(entityID, atk, def, hp int64, unitType int, tags map[string]struct{}) PropEntry {
	return PropEntry{
		EntityID: entityID,
		Atk:      atk,
		Def:      def,
		HP:       hp,
		UnitType: unitType,
		Tags:     tags,
	}
}
