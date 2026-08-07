// Package combat 战斗聚合根，维护战斗状态与轮次执行。
// 本文件定义FormationEffectVO阵型效果值对象的单元测试。
package combat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFormationEffect_Apply 测试阵型效果应用。
func TestFormationEffect_Apply(t *testing.T) {
	f := NewFormationEffect("offensive", 0.2, 0.1, FormationApplyRoundStart, 3)

	modifiedAttack, modifiedDefense := f.Apply(1000, 800)
	assert.Equal(t, int64(1200), modifiedAttack)
	assert.Equal(t, int64(880), modifiedDefense)
}

// TestFormationEffect_Apply_ZeroBonus 测试无加成时效果不变。
func TestFormationEffect_Apply_ZeroBonus(t *testing.T) {
	f := NewFormationEffect("standard", 0, 0, FormationApplyRoundStart, 1)

	modifiedAttack, modifiedDefense := f.Apply(1000, 800)
	assert.Equal(t, int64(1000), modifiedAttack)
	assert.Equal(t, int64(800), modifiedDefense)
}

// TestFormationEffect_ShouldApply 测试阵型效果应用时机判定。
func TestFormationEffect_ShouldApply(t *testing.T) {
	f := NewFormationEffect("offensive", 0.2, 0.1, FormationApplyRoundStart, 3)

	assert.True(t, f.ShouldApply(FormationApplyRoundStart))
	assert.False(t, f.ShouldApply(FormationApplyRoundEnd))
}

// TestFormationEffect_ShouldApply_RoundEnd 测试轮次结束时机应用。
func TestFormationEffect_ShouldApply_RoundEnd(t *testing.T) {
	f := NewFormationEffect("defensive", 0.1, 0.3, FormationApplyRoundEnd, 3)

	assert.False(t, f.ShouldApply(FormationApplyRoundStart))
	assert.True(t, f.ShouldApply(FormationApplyRoundEnd))
}
