// Package combat 战斗聚合根，维护战斗状态与轮次执行。
// 本文件定义结算校验与熔断协议的单元测试（ADR-004 3.2）。
package combat

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"insectworld/server/shared/pkg/config"
	"insectworld/server/shared/pkg/config/mock"
)

// newVersionedQuery 创建带版本化存储的mock配置查询，并预置版本100的常用配置项。
// combatType按快照语义以字符串形式存储（strconv.Itoa后的战斗类型ID）。
func newVersionedQuery() *mock.MockConfigQuery {
	cfg := mock.NewMockConfigQuery()
	cfg.Versioned = config.NewVersionedConfigStore()
	store := cfg.Versioned
	store.PutEntry(100, config.ExtPointCombatTypes, "1", map[string]any{"type": "field"})
	store.PutEntry(100, config.ExtPointDamageFormulas, "dmg_001", "atk*2")
	store.PutEntry(100, config.ExtPointUnitTypes, "1", map[string]any{"unit": "mantis"})
	store.PutEntry(100, config.ExtPointUnitTypes, "2", map[string]any{"unit": "ant"})
	store.PutEntry(100, config.ExtPointCombatLootRules, "loot_001", "gold:100")
	return cfg
}

// newBoundCombat 创建已绑定快照的战斗（configVersion=100，含公式/兵种/战利品引用）。
func newBoundCombat() *Combat {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 100, 1000)
	c.BindSnapshot("dmg_001", "loot_001", []string{"skill_1"}, map[int64]PropEntry{
		101: NewPropEntry(101, 120, 60, 800, 1, nil),
	}, map[int64]PropEntry{
		201: NewPropEntry(201, 90, 100, 1000, 2, nil),
	})
	return c
}

// TestValidateSnapshotConfig_AllPresent 测试配置齐全返回空缺失清单（正常结算路径）。
func TestValidateSnapshotConfig_AllPresent(t *testing.T) {
	cfg := newVersionedQuery()
	snap := newBoundCombat().Snapshot()

	missing, err := ValidateSnapshotConfig(context.Background(), &snap, cfg)
	require.NoError(t, err)
	assert.Empty(t, missing)
}

// TestValidateSnapshotConfig_MissingFormula 测试伤害公式缺失触发缺项记录（ADR-004场景A）。
func TestValidateSnapshotConfig_MissingFormula(t *testing.T) {
	cfg := newVersionedQuery()
	// 快照引用公式dmg_999在版本100不存在
	c := newBoundCombat()
	c.BindSnapshot("dmg_999", "loot_001", []string{"skill_1"}, map[int64]PropEntry{
		101: NewPropEntry(101, 120, 60, 800, 1, nil),
	}, map[int64]PropEntry{
		201: NewPropEntry(201, 90, 100, 1000, 2, nil),
	})
	snap := c.Snapshot()

	missing, err := ValidateSnapshotConfig(context.Background(), &snap, cfg)
	require.NoError(t, err)
	require.Len(t, missing, 1)
	assert.Equal(t, config.ExtPointDamageFormulas, missing[0].ExtPointID)
	assert.Equal(t, "dmg_999", missing[0].RefKey)
}

// TestValidateSnapshotConfig_MissingUnitType 测试删除兵种触发缺项记录（ADR-004场景A核心）。
func TestValidateSnapshotConfig_MissingUnitType(t *testing.T) {
	cfg := newVersionedQuery()
	// 攻击方兵种类型99在版本100不存在（运营已删除该兵种）
	c := newBoundCombat()
	c.BindSnapshot("dmg_001", "loot_001", []string{"skill_1"}, map[int64]PropEntry{
		101: NewPropEntry(101, 120, 60, 800, 99, nil),
	}, map[int64]PropEntry{
		201: NewPropEntry(201, 90, 100, 1000, 2, nil),
	})
	snap := c.Snapshot()

	missing, err := ValidateSnapshotConfig(context.Background(), &snap, cfg)
	require.NoError(t, err)
	require.Len(t, missing, 1)
	assert.Equal(t, config.ExtPointUnitTypes, missing[0].ExtPointID)
	assert.Equal(t, "99", missing[0].RefKey)
	assert.Equal(t, int64(101), missing[0].InstanceID)
}

// TestValidateSnapshotConfig_MissingLootRule 测试战利品规则缺失触发缺项记录。
func TestValidateSnapshotConfig_MissingLootRule(t *testing.T) {
	cfg := newVersionedQuery()
	c := newBoundCombat()
	c.BindSnapshot("dmg_001", "loot_999", []string{"skill_1"}, map[int64]PropEntry{
		101: NewPropEntry(101, 120, 60, 800, 1, nil),
	}, map[int64]PropEntry{
		201: NewPropEntry(201, 90, 100, 1000, 2, nil),
	})
	snap := c.Snapshot()

	missing, err := ValidateSnapshotConfig(context.Background(), &snap, cfg)
	require.NoError(t, err)
	require.Len(t, missing, 1)
	assert.Equal(t, config.ExtPointCombatLootRules, missing[0].ExtPointID)
	assert.Equal(t, "loot_999", missing[0].RefKey)
}

// TestValidateSnapshotConfig_NilArgs 测试空快照/空查询返回错误。
func TestValidateSnapshotConfig_NilArgs(t *testing.T) {
	cfg := newVersionedQuery()
	_, err := ValidateSnapshotConfig(context.Background(), nil, cfg)
	assert.Error(t, err)

	snap := newBoundCombat().Snapshot()
	_, err = ValidateSnapshotConfig(context.Background(), &snap, nil)
	assert.Error(t, err)
}

// TestResolveCircuitBreak_ForceDraw 测试默认强制平局策略：缺失配置 → ResultDraw。
func TestResolveCircuitBreak_ForceDraw(t *testing.T) {
	cfg := newVersionedQuery()
	missing := []MissingConfigRef{{ExtPointID: config.ExtPointDamageFormulas, RefKey: "dmg_999"}}

	policy := DefaultCircuitBreakPolicy()
	assert.Equal(t, CircuitBreakForceDraw, policy.Strategy)

	result, err := ResolveCircuitBreak(context.Background(), policy, missing, cfg, 100)
	require.NoError(t, err)
	assert.Equal(t, ResultDraw, result)
}

// TestResolveCircuitBreak_NoMissing 测试无缺失配置时调用熔断决策返回错误。
func TestResolveCircuitBreak_NoMissing(t *testing.T) {
	cfg := newVersionedQuery()
	_, err := ResolveCircuitBreak(context.Background(), DefaultCircuitBreakPolicy(), nil, cfg, 100)
	assert.Error(t, err)
}

// TestResolveCircuitBreak_UnknownStrategy 测试未知熔断策略返回错误。
func TestResolveCircuitBreak_UnknownStrategy(t *testing.T) {
	cfg := newVersionedQuery()
	missing := []MissingConfigRef{{ExtPointID: config.ExtPointDamageFormulas, RefKey: "dmg_999"}}

	policy := CircuitBreakPolicy{Strategy: 99}
	_, err := ResolveCircuitBreak(context.Background(), policy, missing, cfg, 100)
	assert.Error(t, err)
}

// TestResolveCircuitBreak_FallbackSettle_FormulaPresent 测试兜底结算且兜底公式存在 → 可结算。
func TestResolveCircuitBreak_FallbackSettle_FormulaPresent(t *testing.T) {
	cfg := newVersionedQuery()
	cfg.Versioned.PutEntry(100, config.ExtPointDamageFormulas, "fallback_dmg", "atk")
	missing := []MissingConfigRef{{ExtPointID: config.ExtPointDamageFormulas, RefKey: "dmg_999"}}

	policy := CircuitBreakPolicy{Strategy: CircuitBreakFallbackSettle, FallbackFormulaID: "fallback_dmg"}
	result, err := ResolveCircuitBreak(context.Background(), policy, missing, cfg, 100)
	require.NoError(t, err)
	assert.Equal(t, ResultAttackerWin, result)
}

// TestResolveCircuitBreak_FallbackSettle_NoFormula 测试兜底公式未预置/缺失 → 回落强制平局。
func TestResolveCircuitBreak_FallbackSettle_NoFormula(t *testing.T) {
	cfg := newVersionedQuery()
	missing := []MissingConfigRef{{ExtPointID: config.ExtPointDamageFormulas, RefKey: "dmg_999"}}

	// 未预置兜底公式 → 回落平局
	policy := CircuitBreakPolicy{Strategy: CircuitBreakFallbackSettle}
	result, err := ResolveCircuitBreak(context.Background(), policy, missing, cfg, 100)
	require.NoError(t, err)
	assert.Equal(t, ResultDraw, result)

	// 兜底公式在快照版本缺失 → 回落平局
	policy = CircuitBreakPolicy{Strategy: CircuitBreakFallbackSettle, FallbackFormulaID: "missing_fallback"}
	result, err = ResolveCircuitBreak(context.Background(), policy, missing, cfg, 100)
	require.NoError(t, err)
	assert.Equal(t, ResultDraw, result)
}

// TestCircuitBrokenEvent 测试熔断事件字段完整。
func TestCircuitBrokenEvent(t *testing.T) {
	event := &CircuitBrokenEvent{
		CombatID:       1,
		MissingConfigs: []MissingConfigRef{{ExtPointID: config.ExtPointDamageFormulas, RefKey: "dmg_999"}},
		Strategy:       CircuitBreakForceDraw,
		Result:         ResultDraw,
	}
	assert.Equal(t, int64(1), event.CombatID)
	assert.Len(t, event.MissingConfigs, 1)
	assert.Equal(t, CircuitBreakForceDraw, event.Strategy)
	assert.Equal(t, ResultDraw, event.Result)
}
