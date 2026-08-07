// Package combat 战斗聚合根，维护战斗状态与轮次执行。
// 本文件定义ResultModifierVO战果修正器值对象，从config查询战果修正规则。
package combat

// 战果修正器类型常量（规范1）。
const (
	ModifierTypeFirstWin  = 1 // 首胜加成
	ModifierTypeWinStreak = 2 // 连胜加成
)

// ResultModifierVO 战果修正器值对象，封装战果修正规则。
// 修正器无独立生命周期，使用值对象而非聚合根（规范4）。
type ResultModifierVO struct {
	ModifierType   int     // 修正器类型：1=首胜加成 2=连胜加成（规范8用int枚举）
	ModifierValue  float64 // 修正值，由配置注入（规范8配置数值例外，float64）
	WinStreakCount int     // 连胜次数，ModifierType=2时生效
}

// NewResultModifier 创建战果修正器值对象实例。
func NewResultModifier(modifierType int, modifierValue float64, winStreakCount int) *ResultModifierVO {
	return &ResultModifierVO{
		ModifierType:   modifierType,
		ModifierValue:  modifierValue,
		WinStreakCount: winStreakCount,
	}
}

// Apply 应用修正器到战利品数值。
// baseLoot为原始战利品值，返回修正后的值。
func (m *ResultModifierVO) Apply(baseLoot int64) int64 {
	modified := float64(baseLoot) * (1 + m.ModifierValue)
	return int64(modified)
}

// ChainApply 链式应用多个修正器。
// modifiers为修正器列表，按顺序链式应用。
func ChainApply(baseLoot int64, modifiers ...*ResultModifierVO) int64 {
	result := baseLoot
	for _, m := range modifiers {
		result = m.Apply(result)
	}
	return result
}
