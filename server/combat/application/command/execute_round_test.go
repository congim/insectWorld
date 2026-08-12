// Package command Combat服务application层命令，编排domain层聚合根与技能释放。
// 本文件定义ExecuteRoundHandler的单元测试。
package command

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"insectworld/server/combat/domain/combat"
	"insectworld/server/combat/domain/skill"

	"insectworld/server/shared/pkg/config/mock"
	"insectworld/server/shared/pkg/formula"
)

// mockFormulaEngine FormulaEvaluator的mock实现，按公式ID返回固定结果（E3-S4）。
type mockFormulaEngine struct {
	results map[string]float64 // 公式ID→固定结果
	err     error              // 注入求值错误
}

// EvalByID 按公式ID返回固定结果；未注册公式返回错误。
func (m *mockFormulaEngine) EvalByID(formulaID string, vars map[string]float64, rand formula.RandSource) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	if v, ok := m.results[formulaID]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("公式 %s 未注册", formulaID)
}

// newRealFormulaEngine 创建真实公式引擎并注册测试伤害公式。
// formulaID注册公式"atk * counter + def * 0.5"（确定性公式，便于精确断言伤害）。
func newRealFormulaEngine(t *testing.T, formulaID string) *formula.FormulaEngine {
	t.Helper()
	engine := formula.NewFormulaEngine()
	f, err := engine.Parse("atk * counter + def * 0.5")
	require.NoError(t, err)
	require.NoError(t, engine.Register(formulaID, f))
	return engine
}

// newBoundCombatForRound 创建绑定伤害公式的战斗（configVersion=100，攻击方atk=100/def=50）。
func newBoundCombatForRound(combatID int64, formulaID string) *combat.Combat {
	c := combat.NewCombat(combatID, 1, 10, []int64{101}, []int64{201}, 100, 1000)
	c.BindSnapshot(formulaID, "loot_001", nil, map[int64]combat.PropEntry{
		101: combat.NewPropEntry(101, 100, 50, 800, 1, nil),
	}, map[int64]combat.PropEntry{
		201: combat.NewPropEntry(201, 90, 100, 1000, 2, nil),
	})
	return c
}

// mockCombatRepository Combat仓储mock实现。
type mockCombatRepository struct {
	mu      sync.Mutex
	combat  *combat.Combat
	loadErr error
	saveErr error
}

// LoadCombat 加载战斗聚合根的mock实现。
func (r *mockCombatRepository) LoadCombat(ctx context.Context, combatID int64) (*combat.Combat, error) {
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	return r.combat, nil
}

// SaveCombat 保存战斗聚合根的mock实现。
func (r *mockCombatRepository) SaveCombat(ctx context.Context, c *combat.Combat) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.combat = c
	if r.saveErr != nil {
		return r.saveErr
	}
	return nil
}

// mockOutbox Outbox mock实现。
type mockOutbox struct {
	events []any
	err    error
}

// Append 写Outbox的mock实现。
func (o *mockOutbox) Append(ctx context.Context, event any) error {
	if o.err != nil {
		return o.err
	}
	o.events = append(o.events, event)
	return nil
}

// TestExecuteRoundHandler_Success 测试战斗轮次执行成功。
func TestExecuteRoundHandler_Success(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	c := combat.NewCombat(1, 1, 10, []int64{101}, []int64{201}, 100, 1000)
	repo := &mockCombatRepository{combat: c}
	outbox := &mockOutbox{}

	handler := NewExecuteRoundHandler(repo, cfg, formula.NewFormulaEngine(), skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.combat.CurrentRound())
	assert.Len(t, outbox.events, 1)
}

// TestExecuteRoundHandler_CombatNotFound 测试战斗不存在时执行失败。
func TestExecuteRoundHandler_CombatNotFound(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	repo := &mockCombatRepository{combat: nil}
	outbox := &mockOutbox{}

	handler := NewExecuteRoundHandler(repo, cfg, formula.NewFormulaEngine(), skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 999})
	assert.Error(t, err)
}

// TestExecuteRoundHandler_LoadError 测试加载战斗失败。
func TestExecuteRoundHandler_LoadError(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	repo := &mockCombatRepository{loadErr: assert.AnError}
	outbox := &mockOutbox{}

	handler := NewExecuteRoundHandler(repo, cfg, formula.NewFormulaEngine(), skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
	assert.Error(t, err)
}

// TestExecuteRoundHandler_MaxRoundsForceDraw 测试轮数超限强制平局。
func TestExecuteRoundHandler_MaxRoundsForceDraw(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	// maxRounds=1，执行1轮后超限强制平局
	c := combat.NewCombat(1, 1, 1, []int64{101}, []int64{201}, 100, 1000)
	repo := &mockCombatRepository{combat: c}
	outbox := &mockOutbox{}

	handler := NewExecuteRoundHandler(repo, cfg, formula.NewFormulaEngine(), skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
	require.NoError(t, err)
	assert.Equal(t, combat.StatusEnded, repo.combat.Status())
	// 超限平局事件 + 轮次完成事件 = 2个事件
	assert.Len(t, outbox.events, 2)
}

// TestExecuteRoundHandler_SaveError 测试保存战斗失败。
func TestExecuteRoundHandler_SaveError(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	c := combat.NewCombat(1, 1, 10, []int64{101}, []int64{201}, 100, 1000)
	repo := &mockCombatRepository{combat: c, saveErr: assert.AnError}
	outbox := &mockOutbox{}

	handler := NewExecuteRoundHandler(repo, cfg, formula.NewFormulaEngine(), skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
	assert.Error(t, err)
}

// TestExecuteRoundHandler_OutboxError 测试写Outbox失败。
func TestExecuteRoundHandler_OutboxError(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	c := combat.NewCombat(1, 1, 10, []int64{101}, []int64{201}, 100, 1000)
	repo := &mockCombatRepository{combat: c}
	outbox := &mockOutbox{err: assert.AnError}

	handler := NewExecuteRoundHandler(repo, cfg, formula.NewFormulaEngine(), skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
	assert.Error(t, err)
}

// TestExecuteRoundHandler_MultipleRounds 测试多轮次执行。
func TestExecuteRoundHandler_MultipleRounds(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	c := combat.NewCombat(1, 1, 5, []int64{101}, []int64{201}, 100, 1000)
	repo := &mockCombatRepository{combat: c}
	outbox := &mockOutbox{}

	handler := NewExecuteRoundHandler(repo, cfg, formula.NewFormulaEngine(), skillSvc, outbox, logger)

	// 执行3轮
	for i := 0; i < 3; i++ {
		err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
		require.NoError(t, err)
	}

	assert.Equal(t, 3, repo.combat.CurrentRound())
	assert.True(t, repo.combat.IsInProgress())
	// 3轮，每轮1个轮次完成事件
	assert.Len(t, outbox.events, 3)
}

// TestExecuteRoundHandler_DamageCalculation_MockEngine 测试mock公式引擎计算伤害写入轮次事件。
func TestExecuteRoundHandler_DamageCalculation_MockEngine(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	c := newBoundCombatForRound(1, "dmg_001")
	repo := &mockCombatRepository{combat: c}
	outbox := &mockOutbox{}

	engine := &mockFormulaEngine{results: map[string]float64{"dmg_001": 250.5}}
	handler := NewExecuteRoundHandler(repo, cfg, engine, skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
	require.NoError(t, err)

	// 轮次完成事件携带伤害（int64取整）
	require.Len(t, outbox.events, 1)
	roundEvent, ok := outbox.events[0].(*combat.RoundCompletedEvent)
	require.True(t, ok)
	assert.Equal(t, int64(250), roundEvent.Damage)
}

// TestExecuteRoundHandler_DamageCalculation_RealEngine 测试真实公式引擎计算伤害与种子确定性。
// 公式"atk * counter + def * 0.5"，atk=100/def=50 → 基础伤害125；同战斗重放伤害一致。
func TestExecuteRoundHandler_DamageCalculation_RealEngine(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)
	engine := newRealFormulaEngine(t, "dmg_001")

	// 同combatID执行两轮，种子派生一致 → 每轮伤害确定
	c := newBoundCombatForRound(1, "dmg_001")
	repo := &mockCombatRepository{combat: c}
	outbox := &mockOutbox{}
	handler := NewExecuteRoundHandler(repo, cfg, engine, skillSvc, outbox, logger)

	for i := 0; i < 2; i++ {
		err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
		require.NoError(t, err)
	}
	require.Len(t, outbox.events, 2)
	first, ok1 := outbox.events[0].(*combat.RoundCompletedEvent)
	second, ok2 := outbox.events[1].(*combat.RoundCompletedEvent)
	require.True(t, ok1)
	require.True(t, ok2)
	assert.Equal(t, int64(125), first.Damage)
	assert.Equal(t, int64(125), second.Damage)
}

// TestExecuteRoundHandler_DamageCalculation_NoFormula 测试未绑定公式的战斗不计算伤害（旧数据兼容）。
func TestExecuteRoundHandler_DamageCalculation_NoFormula(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	// 不绑定公式
	c := combat.NewCombat(1, 1, 10, []int64{101}, []int64{201}, 100, 1000)
	repo := &mockCombatRepository{combat: c}
	outbox := &mockOutbox{}

	handler := NewExecuteRoundHandler(repo, cfg, formula.NewFormulaEngine(), skillSvc, outbox, logger)
	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
	require.NoError(t, err)

	require.Len(t, outbox.events, 1)
	roundEvent, ok := outbox.events[0].(*combat.RoundCompletedEvent)
	require.True(t, ok)
	assert.Equal(t, int64(0), roundEvent.Damage)
}

// TestExecuteRoundHandler_FormulaEvalError 测试公式求值失败时轮次执行返回错误。
func TestExecuteRoundHandler_FormulaEvalError(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	logger := zap.NewNop()
	skillSvc := skill.NewSkillService(cfg, logger)

	c := newBoundCombatForRound(1, "dmg_001")
	repo := &mockCombatRepository{combat: c}
	outbox := &mockOutbox{}

	engine := &mockFormulaEngine{err: fmt.Errorf("公式求值失败")}
	handler := NewExecuteRoundHandler(repo, cfg, engine, skillSvc, outbox, logger)

	err := handler.Handle(context.Background(), ExecuteRoundCommand{CombatID: 1})
	assert.Error(t, err)
}
