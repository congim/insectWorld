// Package command Combat服务application层命令，编排domain层聚合根与技能释放。
// 本文件定义SettleHandler战斗结算命令的单元测试（ADR-004 3.2结算熔断）。
package command

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"insectworld/server/combat/domain/combat"
	"insectworld/server/shared/pkg/config"
	"insectworld/server/shared/pkg/config/mock"
)

// newSettleScenario 创建结算测试场景：带版本化配置查询的mock + 绑定快照的战斗。
// versioned为true时预置版本100的完整配置（正常结算路径）；为false时仅预置战斗类型（公式缺失触发熔断）。
func newSettleScenario(t *testing.T, combatID int64, versioned bool) (*mock.MockConfigQuery, *mockCombatRepository, *mockOutbox, *SettleHandler) {
	t.Helper()
	cfg := mock.NewMockConfigQuery()
	cfg.Versioned = config.NewVersionedConfigStore()
	store := cfg.Versioned
	store.PutEntry(100, config.ExtPointCombatTypes, "1", map[string]any{"type": "field"})
	if versioned {
		store.PutEntry(100, config.ExtPointDamageFormulas, "dmg_001", "atk*2")
		store.PutEntry(100, config.ExtPointUnitTypes, "1", map[string]any{"unit": "mantis"})
		store.PutEntry(100, config.ExtPointUnitTypes, "2", map[string]any{"unit": "ant"})
		store.PutEntry(100, config.ExtPointCombatLootRules, "loot_001", "gold:100")
	}

	c := combat.NewCombat(combatID, 1, 10, []int64{101}, []int64{201}, 100, 1000)
	c.BindSnapshot("dmg_001", "loot_001", []string{"skill_1"}, map[int64]combat.PropEntry{
		101: combat.NewPropEntry(101, 120, 60, 800, 1, nil),
	}, map[int64]combat.PropEntry{
		201: combat.NewPropEntry(201, 90, 100, 1000, 2, nil),
	})
	// 执行一轮使战斗有轮次记录
	_, err := c.ExecuteRound()
	require.NoError(t, err)

	repo := &mockCombatRepository{combat: c}
	outbox := &mockOutbox{}
	logger := zap.NewNop()
	handler := NewSettleHandler(repo, cfg, outbox, logger)
	return cfg, repo, outbox, handler
}

// TestSettleHandler_NormalSettle 测试配置齐全正常结算（无熔断事件，结果=攻击方胜）。
func TestSettleHandler_NormalSettle(t *testing.T) {
	_, repo, outbox, handler := newSettleScenario(t, 1, true)

	err := handler.Handle(context.Background(), SettleCommand{CombatID: 1})
	require.NoError(t, err)

	assert.Equal(t, combat.StatusEnded, repo.combat.Status())
	// 正常结算只发布战斗结束事件，无熔断事件
	require.Len(t, outbox.events, 1)
	endEvent, ok := outbox.events[0].(*combat.CombatEndedEvent)
	require.True(t, ok)
	assert.Equal(t, combat.ResultAttackerWin, endEvent.Result)
}

// TestSettleHandler_MissingFormula_CircuitBreak 测试公式缺失触发熔断强制平局并发布熔断事件。
func TestSettleHandler_MissingFormula_CircuitBreak(t *testing.T) {
	_, repo, outbox, handler := newSettleScenario(t, 1, false)

	err := handler.Handle(context.Background(), SettleCommand{CombatID: 1})
	require.NoError(t, err)

	assert.Equal(t, combat.StatusEnded, repo.combat.Status())
	// 熔断路径发布2个事件：熔断事件 + 战斗结束事件
	require.Len(t, outbox.events, 2)

	// 战斗结束事件结果为平局（ForceDraw，不发放战利品）
	endEvent, ok := outbox.events[1].(*combat.CombatEndedEvent)
	require.True(t, ok)
	assert.Equal(t, combat.ResultDraw, endEvent.Result)

	// 熔断事件携带缺失项清单与策略
	circuitEvent, ok := outbox.events[0].(*combat.CircuitBrokenEvent)
	require.True(t, ok)
	assert.Equal(t, int64(1), circuitEvent.CombatID)
	assert.Equal(t, combat.CircuitBreakForceDraw, circuitEvent.Strategy)
	assert.Equal(t, combat.ResultDraw, circuitEvent.Result)
	require.NotEmpty(t, circuitEvent.MissingConfigs)
	// 公式缺失是熔断触发项之一
	foundFormula := false
	for _, ref := range circuitEvent.MissingConfigs {
		if ref.ExtPointID == config.ExtPointDamageFormulas {
			foundFormula = true
		}
	}
	assert.True(t, foundFormula, "缺失项清单应包含伤害公式")
}

// TestSettleHandler_CombatNotFound 测试战斗不存在时结算失败。
func TestSettleHandler_CombatNotFound(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	repo := &mockCombatRepository{combat: nil}
	outbox := &mockOutbox{}

	handler := NewSettleHandler(repo, cfg, outbox, logger)
	err := handler.Handle(context.Background(), SettleCommand{CombatID: 999})
	assert.Error(t, err)
}

// TestSettleHandler_LoadError 测试加载战斗失败。
func TestSettleHandler_LoadError(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	repo := &mockCombatRepository{loadErr: assert.AnError}
	outbox := &mockOutbox{}

	handler := NewSettleHandler(repo, cfg, outbox, logger)
	err := handler.Handle(context.Background(), SettleCommand{CombatID: 1})
	assert.Error(t, err)
}

// TestSettleHandler_SetCircuitBreakPolicy 测试可注入自定义熔断策略。
func TestSettleHandler_SetCircuitBreakPolicy(t *testing.T) {
	_, _, outbox, handler := newSettleScenario(t, 1, false)

	// 注入兜底结算策略（未预置兜底公式 → 回落强制平局）
	handler.SetCircuitBreakPolicy(combat.CircuitBreakPolicy{
		Strategy:          combat.CircuitBreakFallbackSettle,
		FallbackFormulaID: "",
	})

	err := handler.Handle(context.Background(), SettleCommand{CombatID: 1})
	require.NoError(t, err)
	require.Len(t, outbox.events, 2)
	endEvent, ok := outbox.events[1].(*combat.CombatEndedEvent)
	require.True(t, ok)
	assert.Equal(t, combat.ResultDraw, endEvent.Result)
}
