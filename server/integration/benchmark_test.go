// Package integration 跨服务集成测试，验证服务间通过领域事件的协作。
// 本文件定义关键路径的性能基准测试。
package integration

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"insectworld/server/combat/domain/combat"
	"insectworld/server/economy/domain/wallet"
	"insectworld/server/shared/pkg/config"
	"insectworld/server/shared/pkg/config/mock"
)

// BenchmarkConfigQuery_GetTerrain 配置查询性能基准（要求<1ms）。
func BenchmarkConfigQuery_GetTerrain(b *testing.B) {
	cfg := mock.NewMockConfigQuery()
	cfg.Terrain["plain"] = &config.TerrainConfig{
		TerrainID: "plain", MoveCost: 1, DefenseBonus: 10, IsBlock: false, IsBuildable: true,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.GetTerrain(ctx, "plain")
	}
}

// BenchmarkConfigQuery_GetCombatSkill 技能配置查询性能基准。
func BenchmarkConfigQuery_GetCombatSkill(b *testing.B) {
	cfg := mock.NewMockConfigQuery()
	cfg.CombatSkill["1"] = &config.SkillConfig{
		SkillID: "1", CooldownRounds: 3, EffectType: 1, EffectValue: 500,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.GetCombatSkill(ctx, "1")
	}
}

// BenchmarkCombat_ExecuteRound 战斗轮次执行性能基准。
func BenchmarkCombat_ExecuteRound(b *testing.B) {
	c := combat.NewCombat(1, 1, 100, []int64{101, 102}, []int64{201, 202}, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.ExecuteRound()
		if c.CheckMaxRounds() {
			_, _ = c.End(combat.ResultDraw)
			c = combat.NewCombat(1, 1, 100, []int64{101, 102}, []int64{201, 202}, 1000)
		}
	}
}

// BenchmarkWallet_Produce 钱包资源产出性能基准。
func BenchmarkWallet_Produce(b *testing.B) {
	w := wallet.NewPlayerWallet(1)
	ctx := context.Background()
	modifiers := []float64{0.2, 0.1, 0.05}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = w.Produce(ctx, 100, 500, modifiers, wallet.OverflowDiscard, 1000000)
	}
}

// BenchmarkWallet_Consume 钱包资源消耗性能基准。
func BenchmarkWallet_Consume(b *testing.B) {
	w := wallet.NewPlayerWallet(1)
	w.AddBalance(100, int64(b.N)*100)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.Consume(ctx, 100, 100)
	}
}

// BenchmarkExtensionRegistry_Query 扩展点注册表查询性能基准。
func BenchmarkExtensionRegistry_Query(b *testing.B) {
	logger := zap.NewNop()
	registry := config.NewExtensionRegistry(logger)
	ctx := context.Background()

	_ = registry.RegisterContract(config.ExtensionPointContract{ExtPointID: config.ExtPointTerrains})
	_ = registry.Register(ctx, config.ExtPointTerrains, map[string]any{"plain": "grass"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.Query(ctx, config.ExtPointTerrains)
	}
}
