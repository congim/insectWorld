// Package combat 战斗聚合根，维护战斗状态与轮次执行。
// 本文件定义轮次随机种子派生的单元测试（ADR-001 3.3 / ADR-004 3.5）。
package combat

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"insectworld/server/shared/pkg/formula"
)

// TestDeriveRoundSeed_Deterministic 测试同参数派生种子必然一致（重放一致性基础）。
func TestDeriveRoundSeed_Deterministic(t *testing.T) {
	assert.Equal(t, DeriveRoundSeed(100, 5, 3), DeriveRoundSeed(100, 5, 3))
	assert.Equal(t, DeriveRoundSeed(0, 0, 1), DeriveRoundSeed(0, 0, 1))
}

// TestDeriveRoundSeed_ConfigVersionAware 测试config_version参与种子派生（热更/回滚版本不同种子不同）。
func TestDeriveRoundSeed_ConfigVersionAware(t *testing.T) {
	// 同一战斗同一轮次，配置版本不同 → 种子不同，跨版本随机流不串（ADR-004 3.5）
	assert.NotEqual(t, DeriveRoundSeed(100, 5, 3), DeriveRoundSeed(100, 6, 3))
}

// TestDeriveRoundSeed_RoundAware 测试轮次参与种子派生（每轮随机序列独立）。
func TestDeriveRoundSeed_RoundAware(t *testing.T) {
	assert.NotEqual(t, DeriveRoundSeed(100, 5, 1), DeriveRoundSeed(100, 5, 2))
	assert.NotEqual(t, DeriveRoundSeed(100, 5, 2), DeriveRoundSeed(100, 5, 3))
}

// TestDeriveRoundSeed_CombatIDAware 测试combatID参与种子派生（不同战斗序列独立）。
func TestDeriveRoundSeed_CombatIDAware(t *testing.T) {
	assert.NotEqual(t, DeriveRoundSeed(100, 5, 3), DeriveRoundSeed(101, 5, 3))
}

// TestDeriveRoundSeed_Pcg32Sequence 测试同种子构建Pcg32产生同一随机序列（回放逐轮一致）。
func TestDeriveRoundSeed_Pcg32Sequence(t *testing.T) {
	seed1 := DeriveRoundSeed(100, 5, 3)
	seed2 := DeriveRoundSeed(100, 5, 3)
	rand1 := formula.NewPcg32(seed1)
	rand2 := formula.NewPcg32(seed2)

	for i := 0; i < 10; i++ {
		assert.Equal(t, rand1.Float64(), rand2.Float64())
	}
}
