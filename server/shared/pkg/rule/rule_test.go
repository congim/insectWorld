// Package rule 通用规则引擎共享内核，提供规则执行契约与注册表。
package rule

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockRuleExecutor 测试用规则执行器mock。
type mockRuleExecutor struct {
	result RuleResult
}

func (m *mockRuleExecutor) Execute(ctx RuleContext) RuleResult {
	return m.result
}

// TestRuleRegistry_Register 测试规则执行器注册。
func TestRuleRegistry_Register(t *testing.T) {
	registry := NewRuleRegistry()
	executor := &mockRuleExecutor{result: RuleResult{Success: true}}

	t.Run("正常注册", func(t *testing.T) {
		err := registry.Register("combat.damage_formulas", executor)
		assert.NoError(t, err)
	})

	t.Run("空扩展点ID", func(t *testing.T) {
		err := registry.Register("", executor)
		assert.Error(t, err)
	})

	t.Run("nil执行器", func(t *testing.T) {
		err := registry.Register("test.point", nil)
		assert.Error(t, err)
	})

	t.Run("重复注册", func(t *testing.T) {
		err := registry.Register("combat.damage_formulas", executor)
		assert.Error(t, err)
	})
}

// TestRuleRegistry_Get 测试规则执行器查询。
func TestRuleRegistry_Get(t *testing.T) {
	registry := NewRuleRegistry()
	executor := &mockRuleExecutor{result: RuleResult{Success: true, Output: map[string]any{"damage": 100}}}
	_ = registry.Register("combat.damage_formulas", executor)

	t.Run("已注册的扩展点", func(t *testing.T) {
		got, err := registry.Get("combat.damage_formulas")
		assert.NoError(t, err)
		assert.NotNil(t, got)

		result := got.Execute(RuleContext{ExtensionPointID: "combat.damage_formulas"})
		assert.True(t, result.Success)
		assert.Equal(t, 100, result.Output["damage"])
	})

	t.Run("未注册的扩展点", func(t *testing.T) {
		got, err := registry.Get("unknown.point")
		assert.Error(t, err)
		assert.Nil(t, got)
	})
}

// TestRuleContext 测试规则上下文构造。
func TestRuleContext(t *testing.T) {
	ctx := RuleContext{
		ExtensionPointID: "economy.production_rules",
		Input:            map[string]any{"level": 5, "baseRate": 100},
		Config:           "config data",
	}
	assert.Equal(t, "economy.production_rules", ctx.ExtensionPointID)
	assert.Equal(t, 5, ctx.Input["level"])
	assert.Equal(t, 100, ctx.Input["baseRate"])
	assert.Equal(t, "config data", ctx.Config)
}
