// Package replay 战斗重放校验工具，实现战报记录与逐轮重放比对（E3-S5 / ADR-004 3.5）。
// 战报记录combatID+快照configVersion+战斗级种子+逐轮结果；重放按战报configVersion取公式源码并编译，
// 每轮用DeriveRoundSeed(combatID, configVersion, round)重建Pcg32随机源重算伤害并与战报逐轮比对。
// 本工具是ADR-001（确定性随机）与ADR-004（快照版本绑定）的验收项：configVersion取快照冻结值，
// 热更/回滚发生在战斗中途也不影响重放（ADR-004 3.5重放一致性方案）。
package replay

import (
	"context"
	"fmt"
	"strings"

	"insectworld/server/combat/domain/combat"
	"insectworld/server/shared/pkg/config"
	"insectworld/server/shared/pkg/formula"
)

// 不一致原因分类常量（规范1，就近归属重放校验领域）。
const (
	// MismatchReasonFormulaMissing 公式ID缺失（配置版本不可用）：战报引用的公式在快照版本不存在，
	// 或快照版本已被清理（ADR-004场景A重放侧检测）。
	MismatchReasonFormulaMissing = 1
	// MismatchReasonRandomDrift 随机序列漂移：战报记录的轮次种子与战斗参数派生种子不一致，
	// 随机流无法复现（ADR-004 3.5种子不漂移约束被破坏）。
	MismatchReasonRandomDrift = 2
	// MismatchReasonPropChanged 属性快照变化：确定性公式重放不一致，参与计算的属性快照与实战斗不一致。
	// 快照版本公式不可变（版本化查询隔离），确定性公式不一致只能来自属性快照（spec.md功能10每轮刷新）。
	MismatchReasonPropChanged = 3
	// MismatchReasonValueDeviation 数值偏差：含random公式在种子一致前提下重放结果与战报不一致，
	// 属于战报数值记录错误或被篡改（数值层面的偏差，非随机流/快照问题）。
	MismatchReasonValueDeviation = 4
)

// battleBaseRound 战斗级基准轮次，用于派生战斗级种子（round=0即战斗开始前的基准）。
const battleBaseRound = 0

// CombatReport 战斗战报，用于持久化与重放校验（ADR-004 3.5重放一致性方案）。
// 战报记录快照冻结的配置版本与逐轮结果，重放时按configVersion取公式、按战斗参数派生每轮随机源，
// 与真实战斗完全一致的口径重算，逐轮比对验证回放一致性。
type CombatReport struct {
	CombatID      int64         // 战斗ID，全局唯一，由雪花算法生成（规范8用int64）
	ConfigVersion int64         // 快照冻结的配置版本号，开战时记录，热更/回滚不改变（ADR-004 3.1）
	Seed          uint64        // 战斗级种子，= DeriveRoundSeed(combatID, configVersion, 0)，重放完整性校验基准
	Rounds        []RoundRecord // 逐轮记录，按轮次升序，重放逐轮比对
}

// RoundRecord 战报单轮记录，重放时逐轮重建随机源并比对伤害结果（ADR-004 3.5）。
// 双方属性快照按开战/当轮刷新值记录；本期伤害公式只消费攻击方atk/def（与execute_round.go口径一致），
// 防守方属性预留供克制/反伤/减免公式接入后参与重放。
type RoundRecord struct {
	Round       int    // 轮次序号，从1开始
	FormulaID   string // 本轮伤害公式ID，对应快照版本formulas.json
	Seed        uint64 // 本轮随机种子，= DeriveRoundSeed(combatID, configVersion, round)，重放重建随机源
	AttackerAtk int64  // 本轮攻击方攻击力（int64，AGENTS.md规范8）
	AttackerDef int64  // 本轮攻击方防御力
	DefenderAtk int64  // 本轮防守方攻击力（预留：克制/反伤公式接入后参与重放）
	DefenderDef int64  // 本轮防守方防御力（预留：伤害减免公式接入后参与重放）
	Damage      int64  // 本轮伤害结果（int64，公式引擎结果取整，AGENTS.md规范8）
}

// FormulaEngine 重放用公式引擎最小接口，定义在消费方（规范9接口小化），
// 真实实现为formula.FormulaEngine：Parse编译公式、Eval按变量与随机源求值；测试可用mock替换。
type FormulaEngine interface {
	// Parse 解析并编译公式表达式，语法/函数/深度校验任一失败返回错误。
	Parse(source string) (*formula.Formula, error)
	// Eval 按变量与随机源求值公式，返回float64；同一Vars+Rand必然同结果（确定性）。
	Eval(f *formula.Formula, vars map[string]float64, rand formula.RandSource) (float64, error)
}

// ReplayResult 重放校验结果。
type ReplayResult struct {
	Matched        bool            // 是否全部轮次匹配，无任何不一致轮次时为true
	MismatchRounds []RoundMismatch // 不一致轮次清单，按轮次升序
	TotalRounds    int             // 校验的总轮次数，等于战报轮次记录数
}

// RoundMismatch 单轮不一致记录，供运营定位不一致轮次与原因。
type RoundMismatch struct {
	Round    int   // 轮次序号
	Expected int64 // 战报记录的期望伤害值
	Actual   int64 // 重放计算的实际伤害值；公式ID缺失/随机序列漂移时无法计算为0
	Reason   int   // 不一致原因：1=公式ID缺失 2=随机序列漂移 3=属性快照变化 4=数值偏差
}

// ReplayCombat 战斗重放校验（ADR-004 3.5 / ADR-001验收"战斗回放逐轮一致"）。
// 输入战报+公式引擎+版本化配置查询：按战报configVersion取公式源码并编译，每轮用
// DeriveRoundSeed(combatID, configVersion, round)重建Pcg32随机源，重算伤害并与战报逐轮比对；
// configVersion取快照冻结值，热更/回滚发生在战斗中途也不影响重放（ADR-004 3.5重放一致性方案）。
// 返回ReplayResult（Matched/MismatchRounds/TotalRounds）；配置版本不可用返回错误（含ErrConfigVersionGone）。
func ReplayCombat(ctx context.Context, report *CombatReport, engine FormulaEngine, query config.ConfigQueryAPI) (*ReplayResult, error) {
	if report == nil {
		return nil, fmt.Errorf("重放校验失败，战报不能为空")
	}
	if engine == nil {
		return nil, fmt.Errorf("重放校验失败，公式引擎不能为空")
	}
	if query == nil {
		return nil, fmt.Errorf("重放校验失败，配置查询接口不能为空")
	}
	if len(report.Rounds) == 0 {
		return nil, fmt.Errorf("重放校验失败，战报无轮次记录，combatID=%d", report.CombatID)
	}

	// 战报完整性校验：战斗级种子必须与战斗参数派生一致（ADR-004 3.5随机流可复现）
	baseSeed := combat.DeriveRoundSeed(report.CombatID, report.ConfigVersion, battleBaseRound)
	if report.Seed != baseSeed {
		return nil, fmt.Errorf("重放校验失败，战报战斗级种子与战斗参数不一致，combatID=%d, configVersion=%d", report.CombatID, report.ConfigVersion)
	}

	result := &ReplayResult{TotalRounds: len(report.Rounds)}
	for i := range report.Rounds {
		rec := &report.Rounds[i]

		// 1. 公式ID缺失校验：公式必须存在于快照版本（ADR-004场景A重放侧检测）；
		//    版本不可用（已被清理/从未记录）直接返回错误，由调用方按熔断/告警处理
		exists, err := query.HasWithVersion(ctx, config.ExtPointDamageFormulas, rec.FormulaID, report.ConfigVersion)
		if err != nil {
			return nil, fmt.Errorf("重放校验失败，配置版本不可用，combatID=%d, configVersion=%d, round=%d: %w", report.CombatID, report.ConfigVersion, rec.Round, err)
		}
		if !exists {
			result.MismatchRounds = append(result.MismatchRounds, RoundMismatch{Round: rec.Round, Expected: rec.Damage, Actual: 0, Reason: MismatchReasonFormulaMissing})
			continue
		}

		// 2. 随机序列漂移校验：本轮种子必须与战斗参数派生一致（ADR-004 3.5种子不漂移）
		expectedSeed := combat.DeriveRoundSeed(report.CombatID, report.ConfigVersion, rec.Round)
		if rec.Seed != expectedSeed {
			result.MismatchRounds = append(result.MismatchRounds, RoundMismatch{Round: rec.Round, Expected: rec.Damage, Actual: 0, Reason: MismatchReasonRandomDrift})
			continue
		}

		// 3. 按快照版本取公式源码并编译（版本化查询，与当前热更版本解耦，ADR-004 3.1）
		raw, err := query.GetWithVersion(ctx, config.ExtPointDamageFormulas, rec.FormulaID, report.ConfigVersion)
		if err != nil {
			return nil, fmt.Errorf("重放校验失败，取公式源码失败，formulaID=%s, round=%d: %w", rec.FormulaID, rec.Round, err)
		}
		src, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("重放校验失败，公式源码类型非法，formulaID=%s, round=%d，期望string实际%T", rec.FormulaID, rec.Round, raw)
		}
		f, err := engine.Parse(src)
		if err != nil {
			return nil, fmt.Errorf("重放校验失败，公式编译失败，formulaID=%s, round=%d: %w", rec.FormulaID, rec.Round, err)
		}

		// 4. 重建随机源并重放伤害计算（离散量伤害取整落库，AGENTS.md规范8）
		rand := formula.NewPcg32(expectedSeed)
		vars := buildDamageVars(rec)
		val, err := engine.Eval(f, vars, rand)
		if err != nil {
			return nil, fmt.Errorf("重放校验失败，伤害公式求值失败，formulaID=%s, round=%d: %w", rec.FormulaID, rec.Round, err)
		}
		actual := int64(val)

		// 5. 与战报记录的期望值逐轮比对，不一致按根因分类记录
		if actual != rec.Damage {
			result.MismatchRounds = append(result.MismatchRounds, RoundMismatch{
				Round:    rec.Round,
				Expected: rec.Damage,
				Actual:   actual,
				Reason:   classifyMismatch(src),
			})
		}
	}

	result.Matched = len(result.MismatchRounds) == 0
	return result, nil
}

// classifyMismatch 分类数值不一致的根因（尽力而为的根因提示，非精确定位）。
// 快照版本公式不可变（ADR-004版本化查询）：确定性公式（不含random函数）不一致说明参与计算的
// 属性快照与实战斗不一致（属性快照变化）；含random公式在种子一致前提下不一致说明战报数值
// 记录错误或被篡改（数值偏差）。用"random("函数调用形态判断，避免与含random子串的变量名误判。
func classifyMismatch(formulaSrc string) int {
	if strings.Contains(formulaSrc, "random(") {
		return MismatchReasonValueDeviation
	}
	return MismatchReasonPropChanged
}

// buildDamageVars 构造重放伤害公式求值变量表，与execute_round.go的buildDamageVars口径一致：
// 本期取攻击方属性（atk/def），counter/terrain_mod暂为1（克制矩阵与地形修正后续接入，配置数值可为小数规范8例外）；
// defender_atk/defender_def为预留变量，供克制/反伤/减免公式接入后引用，当前公式不消费。
func buildDamageVars(rec *RoundRecord) map[string]float64 {
	return map[string]float64{
		"atk":          float64(rec.AttackerAtk),
		"def":          float64(rec.AttackerDef),
		"counter":      1,
		"terrain_mod":  1,
		"defender_atk": float64(rec.DefenderAtk),
		"defender_def": float64(rec.DefenderDef),
	}
}
