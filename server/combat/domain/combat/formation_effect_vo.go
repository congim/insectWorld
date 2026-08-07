// Package combat 战斗聚合根，维护战斗状态与轮次执行。
// 本文件定义FormationEffectVO阵型效果值对象，从config查询阵型效果配置。
package combat

// 阵型效果应用时机常量（规范1）。
const (
	FormationApplyRoundStart = 1 // 轮次开始应用
	FormationApplyRoundEnd   = 2 // 轮次结束应用
)

// FormationEffectVO 阵型效果值对象，封装阵型对战斗的加成效果。
// 阵型效果无独立生命周期，使用值对象而非聚合根（规范4）。
type FormationEffectVO struct {
	FormationID       string  // 阵型ID，对应combat.json的阵型配置
	AttackBonus       float64 // 攻击加成百分比，由配置注入（规范8配置数值例外）
	DefenseBonus      float64 // 防御加成百分比，由配置注入
	ApplyPhase        int     // 应用时机：1=轮次开始 2=轮次结束（规范8用int枚举）
	RequiredUnitCount int     // 所需最低部队数量
}

// NewFormationEffect 创建阵型效果值对象实例。
func NewFormationEffect(formationID string, attackBonus, defenseBonus float64, applyPhase, requiredUnitCount int) *FormationEffectVO {
	return &FormationEffectVO{
		FormationID:       formationID,
		AttackBonus:       attackBonus,
		DefenseBonus:      defenseBonus,
		ApplyPhase:        applyPhase,
		RequiredUnitCount: requiredUnitCount,
	}
}

// Apply 应用阵型效果到攻击/防御值。
// baseAttack/baseDefense为原始值，返回修正后的值。
func (f *FormationEffectVO) Apply(baseAttack, baseDefense int64) (int64, int64) {
	modifiedAttack := int64(float64(baseAttack) * (1 + f.AttackBonus))
	modifiedDefense := int64(float64(baseDefense) * (1 + f.DefenseBonus))
	return modifiedAttack, modifiedDefense
}

// ShouldApply 判断阵型效果是否应在指定时机应用。
func (f *FormationEffectVO) ShouldApply(timing int) bool {
	return f.ApplyPhase == timing
}
