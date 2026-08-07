// Package combat 战斗聚合根，维护战斗状态与轮次执行。
// 本文件定义ResultModifierVO战果修正器值对象的单元测试。
package combat

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestResultModifier_Apply 测试战果修正器应用。
func TestResultModifier_Apply(t *testing.T) {
	m := NewResultModifier(ModifierTypeFirstWin, 0.5, 0)

	result := m.Apply(1000)
	assert.Equal(t, int64(1500), result)
}

// TestResultModifier_Apply_ZeroModifier 测试无修正时值不变。
func TestResultModifier_Apply_ZeroModifier(t *testing.T) {
	m := NewResultModifier(ModifierTypeFirstWin, 0, 0)

	result := m.Apply(1000)
	assert.Equal(t, int64(1000), result)
}

// TestResultModifier_Apply_WinStreak 测试连胜加成修正。
func TestResultModifier_Apply_WinStreak(t *testing.T) {
	m := NewResultModifier(ModifierTypeWinStreak, 0.1, 3)

	result := m.Apply(1000)
	assert.Equal(t, int64(1100), result)
}

// TestChainApply 测试链式应用多个修正器。
func TestChainApply(t *testing.T) {
	m1 := NewResultModifier(ModifierTypeFirstWin, 0.5, 0)
	m2 := NewResultModifier(ModifierTypeWinStreak, 0.1, 3)

	result := ChainApply(1000, m1, m2)
	// 1000 * 1.5 = 1500, 1500 * 1.1 = 1650
	assert.Equal(t, int64(1650), result)
}

// TestChainApply_Empty 测试无修正器时值不变。
func TestChainApply_Empty(t *testing.T) {
	result := ChainApply(1000)
	assert.Equal(t, int64(1000), result)
}

// TestChainApply_Single 测试单个修正器链式应用。
func TestChainApply_Single(t *testing.T) {
	m := NewResultModifier(ModifierTypeFirstWin, 0.2, 0)

	result := ChainApply(1000, m)
	assert.Equal(t, int64(1200), result)
}
