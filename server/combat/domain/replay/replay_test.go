// Package replay 战斗重放校验工具单元测试（E3-S5 / ADR-004 3.5重放一致性验收项）。
// 覆盖：正向全轮匹配、同战报重放两次结果一致（种子确定性）、篡改伤害值检测、
// config_version缺失报错、公式ID缺失/随机序列漂移/属性快照变化的原因分类。
package replay

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"insectworld/server/combat/domain/combat"
	"insectworld/server/shared/pkg/config"
	"insectworld/server/shared/pkg/config/mock"
	"insectworld/server/shared/pkg/formula"
)

// testConfigVersion 测试用配置版本号，对应版本化存储中预置的版本。
const testConfigVersion = 100

// testFormulaRandom 含随机波动的伤害公式源码（ADR-001验证集#1形态）：
// 伤害 = 攻击×(1-防御/100)×random(0.9,1.1)，验证重放随机序列一致。
const testFormulaRandom = "atk * (1 - def/100) * random(0.9, 1.1)"

// testFormulaFixed 确定性伤害公式源码（无random函数），用于属性快照变化分类测试。
const testFormulaFixed = "atk * 2"

// newReplayConfig 创建带版本化存储的mock配置查询，将公式源码预置到版本testConfigVersion。
// 与circuit_break_test.go的装配方式一致：GetWithVersion/HasWithVersion委托VersionedConfigStore。
func newReplayConfig(formulas map[string]string) *mock.MockConfigQuery {
	cfg := mock.NewMockConfigQuery()
	cfg.Versioned = config.NewVersionedConfigStore()
	for id, src := range formulas {
		cfg.Versioned.PutEntry(testConfigVersion, config.ExtPointDamageFormulas, id, src)
	}
	return cfg
}

// runBattle 模拟一场战斗并生成战报，与真实战斗同一种子派生/公式求值口径
// （execute_round.go calculateDamage：DeriveRoundSeed+NewPcg32+Eval+int64取整）。
// 每轮从formulaSrc解析公式并求值，结果写入RoundRecord作为重放校验的期望值；
// 是"先跑一遍战斗记录战报，再重放"正向测试的数据源。
func runBattle(t *testing.T, combatID, configVersion int64, rounds int, formulaID, formulaSrc string, atk, def int64) *CombatReport {
	t.Helper()
	engine := formula.NewFormulaEngine()
	report := &CombatReport{
		CombatID:      combatID,
		ConfigVersion: configVersion,
		Seed:          combat.DeriveRoundSeed(combatID, configVersion, battleBaseRound),
	}
	for r := 1; r <= rounds; r++ {
		seed := combat.DeriveRoundSeed(combatID, configVersion, r)
		f, err := engine.Parse(formulaSrc)
		require.NoError(t, err, "解析公式 %q 应成功", formulaSrc)
		vars := map[string]float64{
			"atk":         float64(atk),
			"def":         float64(def),
			"counter":     1,
			"terrain_mod": 1,
		}
		val, err := engine.Eval(f, vars, formula.NewPcg32(seed))
		require.NoError(t, err, "求值公式 %q 应成功", formulaSrc)
		report.Rounds = append(report.Rounds, RoundRecord{
			Round:       r,
			FormulaID:   formulaID,
			Seed:        seed,
			AttackerAtk: atk,
			AttackerDef: def,
			DefenderAtk: def,
			DefenderDef: atk,
			Damage:      int64(val),
		})
	}
	return report
}

// TestReplayCombat_Matched 正向：先跑一遍战斗记录战报，再重放，断言全轮匹配（ADR-004 3.5重放一致性）。
func TestReplayCombat_Matched(t *testing.T) {
	const combatID = int64(9001)
	cfg := newReplayConfig(map[string]string{"dmg_rand": testFormulaRandom})
	report := runBattle(t, combatID, testConfigVersion, 5, "dmg_rand", testFormulaRandom, 120, 20)

	result, err := ReplayCombat(context.Background(), report, formula.NewFormulaEngine(), cfg)
	require.NoError(t, err)
	assert.True(t, result.Matched, "重放应全轮匹配")
	assert.Equal(t, 5, result.TotalRounds)
	assert.Empty(t, result.MismatchRounds)
}

// TestReplayCombat_Deterministic 种子确定性：同战报重放两次结果一致（ADR-001验收"同种子同结果"）。
func TestReplayCombat_Deterministic(t *testing.T) {
	const combatID = int64(9002)
	cfg := newReplayConfig(map[string]string{"dmg_rand": testFormulaRandom})
	report := runBattle(t, combatID, testConfigVersion, 3, "dmg_rand", testFormulaRandom, 120, 20)

	r1, err := ReplayCombat(context.Background(), report, formula.NewFormulaEngine(), cfg)
	require.NoError(t, err)
	r2, err := ReplayCombat(context.Background(), report, formula.NewFormulaEngine(), cfg)
	require.NoError(t, err)
	assert.Equal(t, r1.Matched, r2.Matched, "两次重放Matched应一致")
	assert.Equal(t, r1.TotalRounds, r2.TotalRounds, "两次重放TotalRounds应一致")
	assert.Equal(t, r1.MismatchRounds, r2.MismatchRounds, "两次重放不一致清单应一致")
}

// TestReplayCombat_DamageTampered 反例：篡改某轮伤害值，重放检测到不一致（数值偏差）。
func TestReplayCombat_DamageTampered(t *testing.T) {
	const combatID = int64(9003)
	cfg := newReplayConfig(map[string]string{"dmg_rand": testFormulaRandom})
	report := runBattle(t, combatID, testConfigVersion, 5, "dmg_rand", testFormulaRandom, 120, 20)

	// 篡改第3轮伤害值（+100），重放按正确种子/快照重算应与原值一致，检测到数值偏差
	report.Rounds[2].Damage += 100

	result, err := ReplayCombat(context.Background(), report, formula.NewFormulaEngine(), cfg)
	require.NoError(t, err)
	assert.False(t, result.Matched, "篡改伤害值后重放应检测到不一致")
	require.Len(t, result.MismatchRounds, 1, "应恰好检测到1轮不一致")
	assert.Equal(t, 3, result.MismatchRounds[0].Round)
	assert.Equal(t, MismatchReasonValueDeviation, result.MismatchRounds[0].Reason)
	assert.NotEqual(t, result.MismatchRounds[0].Expected, result.MismatchRounds[0].Actual)
}

// TestReplayCombat_ConfigVersionGone 反例：config_version缺失（版本已清理/从未记录），重放报错。
func TestReplayCombat_ConfigVersionGone(t *testing.T) {
	const combatID = int64(9004)
	cfg := newReplayConfig(map[string]string{"dmg_rand": testFormulaRandom})
	// 战报引用版本999，版本化存储中不存在（热更清理或从未记录）
	report := runBattle(t, combatID, 999, 3, "dmg_rand", testFormulaRandom, 120, 20)

	_, err := ReplayCombat(context.Background(), report, formula.NewFormulaEngine(), cfg)
	require.Error(t, err)
	assert.True(t, errors.Is(err, config.ErrConfigVersionGone), "错误应可识别为配置版本不可用，实际: %v", err)
}

// TestReplayCombat_FormulaMissing 反例：战报引用的公式在快照版本不存在，重放检测到公式ID缺失。
func TestReplayCombat_FormulaMissing(t *testing.T) {
	const combatID = int64(9005)
	cfg := newReplayConfig(map[string]string{"dmg_rand": testFormulaRandom})
	// 战报公式ID为dmg_missing，快照版本formulas.json中不存在（ADR-004场景A重放侧检测）
	report := runBattle(t, combatID, testConfigVersion, 3, "dmg_missing", testFormulaRandom, 120, 20)

	result, err := ReplayCombat(context.Background(), report, formula.NewFormulaEngine(), cfg)
	require.NoError(t, err)
	assert.False(t, result.Matched, "公式缺失后重放应检测到不一致")
	require.Len(t, result.MismatchRounds, 3, "每轮公式都缺失应逐一记录")
	assert.Equal(t, MismatchReasonFormulaMissing, result.MismatchRounds[0].Reason)
}

// TestReplayCombat_SeedDrift 反例：篡改某轮种子，重放检测到随机序列漂移。
func TestReplayCombat_SeedDrift(t *testing.T) {
	const combatID = int64(9006)
	cfg := newReplayConfig(map[string]string{"dmg_rand": testFormulaRandom})
	report := runBattle(t, combatID, testConfigVersion, 3, "dmg_rand", testFormulaRandom, 120, 20)

	// 篡改第2轮种子为任意值，与DeriveRoundSeed派生值不一致 → 随机流无法复现
	report.Rounds[1].Seed = 42

	result, err := ReplayCombat(context.Background(), report, formula.NewFormulaEngine(), cfg)
	require.NoError(t, err)
	assert.False(t, result.Matched, "种子漂移后重放应检测到不一致")
	require.Len(t, result.MismatchRounds, 1, "应恰好检测到1轮不一致")
	assert.Equal(t, 2, result.MismatchRounds[0].Round)
	assert.Equal(t, MismatchReasonRandomDrift, result.MismatchRounds[0].Reason)
}

// TestReplayCombat_PropChanged 反例：篡改属性快照（确定性公式），重放检测到属性快照变化。
// 确定性公式（atk*2）在种子一致前提下重放不一致，根因只可能是属性快照（spec.md功能10每轮刷新）。
func TestReplayCombat_PropChanged(t *testing.T) {
	const combatID = int64(9007)
	cfg := newReplayConfig(map[string]string{"dmg_fixed": testFormulaFixed})
	report := runBattle(t, combatID, testConfigVersion, 3, "dmg_fixed", testFormulaFixed, 100, 20)

	// 篡改第2轮攻击方攻击力：100→999，确定性公式重放结果应变化
	report.Rounds[1].AttackerAtk = 999

	result, err := ReplayCombat(context.Background(), report, formula.NewFormulaEngine(), cfg)
	require.NoError(t, err)
	assert.False(t, result.Matched, "属性快照变化后重放应检测到不一致")
	require.Len(t, result.MismatchRounds, 1, "应恰好检测到1轮不一致")
	assert.Equal(t, 2, result.MismatchRounds[0].Round)
	assert.Equal(t, MismatchReasonPropChanged, result.MismatchRounds[0].Reason)
}

// TestReplayCombat_NilArgs 边界：空战报/空引擎/空查询返回错误（规范9错误处理）。
func TestReplayCombat_NilArgs(t *testing.T) {
	cfg := newReplayConfig(map[string]string{"dmg_rand": testFormulaRandom})
	report := runBattle(t, 1, testConfigVersion, 1, "dmg_rand", testFormulaRandom, 100, 10)

	_, err := ReplayCombat(context.Background(), nil, formula.NewFormulaEngine(), cfg)
	assert.Error(t, err, "空战报应报错")
	_, err = ReplayCombat(context.Background(), report, nil, cfg)
	assert.Error(t, err, "空公式引擎应报错")
	_, err = ReplayCombat(context.Background(), report, formula.NewFormulaEngine(), nil)
	assert.Error(t, err, "空配置查询应报错")
}

// TestReplayCombat_EmptyRounds 边界：战报无轮次记录返回错误（战报完整性校验）。
func TestReplayCombat_EmptyRounds(t *testing.T) {
	cfg := newReplayConfig(map[string]string{"dmg_rand": testFormulaRandom})
	report := &CombatReport{
		CombatID:      1,
		ConfigVersion: testConfigVersion,
		Seed:          combat.DeriveRoundSeed(1, testConfigVersion, battleBaseRound),
	}

	_, err := ReplayCombat(context.Background(), report, formula.NewFormulaEngine(), cfg)
	assert.Error(t, err, "空轮次战报应报错")
}

// TestReplayCombat_SeedIntegrity 边界：战斗级种子与战斗参数不一致，战报完整性校验报错。
func TestReplayCombat_SeedIntegrity(t *testing.T) {
	cfg := newReplayConfig(map[string]string{"dmg_rand": testFormulaRandom})
	report := runBattle(t, 1, testConfigVersion, 2, "dmg_rand", testFormulaRandom, 100, 10)
	report.Seed = 42 // 篡改战斗级种子，与DeriveRoundSeed(combatID, configVersion, 0)不一致

	_, err := ReplayCombat(context.Background(), report, formula.NewFormulaEngine(), cfg)
	assert.Error(t, err, "战斗级种子不一致应报错")
}
