// Package skill 技能释放domain service，负责技能触发条件判定与效果执行。
// 本文件定义SkillService的单元测试。
package skill

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"insectworld/server/shared/pkg/config"
	"insectworld/server/shared/pkg/config/mock"
)

// TestSkillService_CheckTrigger_SkillNotFound 测试技能不存在时触发条件不满足。
func TestSkillService_CheckTrigger_SkillNotFound(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	svc := NewSkillService(cfg, zap.NewNop())

	result := svc.CheckTrigger(context.Background(), 999, 1, 100)
	assert.False(t, result)
}

// TestSkillService_CheckTrigger_Success 测试技能触发条件满足。
func TestSkillService_CheckTrigger_Success(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.CombatSkill["1"] = &config.SkillConfig{
		SkillID:        "1",
		CooldownRounds: 2,
		TriggerPhase:   TriggerPhaseRoundStart,
		EffectType:     EffectTypeDamage,
		EffectValue:    500,
	}
	svc := NewSkillService(cfg, zap.NewNop())

	result := svc.CheckTrigger(context.Background(), 1, 1, 100)
	assert.True(t, result)
}

// TestSkillService_CheckTrigger_OnCooldown 测试技能冷却中不满足触发条件。
func TestSkillService_CheckTrigger_OnCooldown(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.CombatSkill["1"] = &config.SkillConfig{
		SkillID:        "1",
		CooldownRounds: 3,
		TriggerPhase:   TriggerPhaseRoundStart,
		EffectType:     EffectTypeDamage,
		EffectValue:    500,
	}
	svc := NewSkillService(cfg, zap.NewNop())

	// 第1轮释放技能，设置冷却
	_, err := svc.Execute(context.Background(), 1, 100, 1)
	require.NoError(t, err)

	// 第2轮，冷却中（冷却结束轮次=1+3=4），不应触发
	result := svc.CheckTrigger(context.Background(), 1, 2, 100)
	assert.False(t, result)

	// 第4轮，冷却结束，应触发
	result = svc.CheckTrigger(context.Background(), 1, 4, 100)
	assert.True(t, result)
}

// TestSkillService_Execute_Success 测试技能执行成功。
func TestSkillService_Execute_Success(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.CombatSkill["1"] = &config.SkillConfig{
		SkillID:        "1",
		CooldownRounds: 2,
		TriggerPhase:   TriggerPhaseRoundStart,
		EffectType:     EffectTypeDamage,
		EffectValue:    500,
	}
	svc := NewSkillService(cfg, zap.NewNop())

	trigger, err := svc.Execute(context.Background(), 1, 100, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), trigger.SkillID)
	assert.Equal(t, 1, trigger.TriggerRound)
	assert.Equal(t, EffectTypeDamage, trigger.EffectType)
	assert.Equal(t, int64(500), trigger.EffectValue)
}

// TestSkillService_Execute_SkillNotFound 测试技能不存在时执行失败。
func TestSkillService_Execute_SkillNotFound(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	svc := NewSkillService(cfg, zap.NewNop())

	_, err := svc.Execute(context.Background(), 999, 100, 1)
	assert.Error(t, err)
}

// TestSkillService_Execute_OnCooldown 测试技能冷却中执行失败。
func TestSkillService_Execute_OnCooldown(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.CombatSkill["1"] = &config.SkillConfig{
		SkillID:        "1",
		CooldownRounds: 3,
		TriggerPhase:   TriggerPhaseRoundStart,
		EffectType:     EffectTypeDamage,
		EffectValue:    500,
	}
	svc := NewSkillService(cfg, zap.NewNop())

	// 第1轮释放
	_, err := svc.Execute(context.Background(), 1, 100, 1)
	require.NoError(t, err)

	// 第2轮冷却中，执行失败
	_, err = svc.Execute(context.Background(), 1, 100, 2)
	assert.Error(t, err)
}

// TestSkillService_IsOnCooldown 测试技能冷却判定。
func TestSkillService_IsOnCooldown(t *testing.T) {
	cfg := mock.NewMockConfigQuery()
	cfg.CombatSkill["1"] = &config.SkillConfig{
		SkillID:        "1",
		CooldownRounds: 3,
		TriggerPhase:   TriggerPhaseRoundStart,
		EffectType:     EffectTypeDamage,
		EffectValue:    500,
	}
	svc := NewSkillService(cfg, zap.NewNop())

	// 未释放过，不在冷却中
	assert.False(t, svc.IsOnCooldown(1, 1))

	// 第1轮释放
	_, err := svc.Execute(context.Background(), 1, 100, 1)
	require.NoError(t, err)

	// 第2轮在冷却中（冷却结束轮次=4）
	assert.True(t, svc.IsOnCooldown(1, 2))

	// 第4轮冷却结束
	assert.False(t, svc.IsOnCooldown(1, 4))
}
