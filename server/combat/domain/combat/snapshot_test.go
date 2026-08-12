// Package combat 战斗聚合根，维护战斗状态与轮次执行。
// 本文件定义CombatSnapshot战斗快照的单元测试（ADR-004 3.1）。
package combat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCombatSnapshot_ConfigVersionTransparent 测试开战配置版本透传到快照。
func TestCombatSnapshot_ConfigVersionTransparent(t *testing.T) {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 42, 1000)

	snap := c.Snapshot()
	assert.Equal(t, int64(42), snap.ConfigVersion)
	assert.Equal(t, int64(42), c.ConfigVersion())
	// 克制矩阵版本与配置版本一致（矩阵随配置包整体替换）
	assert.Equal(t, int64(42), snap.CounterMatrixVer)
	// 战斗类型冻结
	assert.Equal(t, 1, snap.CombatType)
}

// TestCombatSnapshot_DefaultUnbound 测试默认configVersion为0（未绑定版本）。
func TestCombatSnapshot_DefaultUnbound(t *testing.T) {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 0, 1000)

	snap := c.Snapshot()
	assert.Equal(t, int64(0), snap.ConfigVersion)
}

// TestCombatSnapshot_BindComplete 测试快照绑定后字段完整。
func TestCombatSnapshot_BindComplete(t *testing.T) {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 100, 1000)

	attackerProps := map[int64]PropEntry{
		101: NewPropEntry(101, 120, 60, 800, 1, map[string]struct{}{"insect": {}}),
	}
	defenderProps := map[int64]PropEntry{
		201: NewPropEntry(201, 90, 100, 1000, 2, map[string]struct{}{"insect": {}}),
	}
	c.BindSnapshot("dmg_001", "loot_001", []string{"skill_1", "skill_2"}, attackerProps, defenderProps)

	snap := c.Snapshot()
	assert.Equal(t, "dmg_001", snap.FormulaID)
	assert.Equal(t, "loot_001", snap.LootRuleID)
	assert.ElementsMatch(t, []string{"skill_1", "skill_2"}, snap.SkillIDs)
	assert.Equal(t, attackerProps, snap.AttackerProps)
	assert.Equal(t, defenderProps, snap.DefenderProps)
	require.Contains(t, snap.AttackerProps, int64(101))
	assert.Equal(t, int64(120), snap.AttackerProps[101].Atk)
	assert.Equal(t, 1, snap.AttackerProps[101].UnitType)
}

// TestCombatSnapshot_ValueCopy 测试Snapshot返回副本，外部修改不影响聚合根。
func TestCombatSnapshot_ValueCopy(t *testing.T) {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 100, 1000)

	// 外部修改快照副本的配置版本，不影响聚合根
	snap := c.Snapshot()
	snap.ConfigVersion = 999
	assert.Equal(t, int64(100), c.ConfigVersion())
}

// TestCombatSnapshot_UpdateProps 测试长时战斗属性快照刷新。
func TestCombatSnapshot_UpdateProps(t *testing.T) {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 100, 1000)
	c.BindSnapshot("dmg_001", "loot_001", nil, map[int64]PropEntry{
		101: NewPropEntry(101, 120, 60, 800, 1, nil),
	}, nil)

	// 属性回写后刷新
	c.UpdateSnapshotProps(map[int64]PropEntry{
		101: NewPropEntry(101, 150, 60, 600, 1, nil),
	}, nil)
	snap := c.Snapshot()
	assert.Equal(t, int64(150), snap.AttackerProps[101].Atk)
	assert.Equal(t, int64(600), snap.AttackerProps[101].HP)
}
