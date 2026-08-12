// Package combat 战斗聚合根，维护战斗状态与轮次执行。
// 本文件实现战斗结算熔断协议（ADR-004 3.2）：结算前校验快照引用的配置项在快照版本仍存在，
// 缺失则按熔断策略处理（默认强制平局），发布combat.circuit_broken事件供运营告警。
// 经济安全性优先于战斗结果完整性：宁可平局损失一场战斗体验，不可错发战利品造成对账失败。
package combat

import (
	"context"
	"fmt"
	"strconv"

	"insectworld/server/shared/pkg/config"
)

// 熔断策略常量（规范1，就近归属战斗领域）。
const (
	// CircuitBreakFallbackSettle 兜底配置结算：按combat.json预置的兜底公式结算，战报标注"熔断-兜底"。
	CircuitBreakFallbackSettle = 1
	// CircuitBreakForceDraw 强制平局：按平局结束，不发放战利品、不产生赛季积分（默认）。
	CircuitBreakForceDraw = 2
)

// DefaultCircuitBreakStrategy 默认熔断策略，经济安全性优先（ADR-004 3.2决策依据）。
const DefaultCircuitBreakStrategy = CircuitBreakForceDraw

// CircuitBreakPolicy 熔断策略，由combat.json配置注入（config.circuit_breaker扩展点）。
// 策略=FallbackSettle时必须预置兜底公式且存在于快照版本formulas.json，否则回落强制平局。
type CircuitBreakPolicy struct {
	Strategy           int    // 熔断策略：1=兜底配置结算 2=强制平局（默认2）
	FallbackFormulaID  string // 兜底结算公式ID，Strategy=1时必须预置且存在于快照版本formulas.json
	FallbackLootRuleID string // 兜底战利品规则ID，Strategy=1时使用；缺失时仅结算战斗结果不发放战利品
}

// DefaultCircuitBreakPolicy 返回默认熔断策略（强制平局）。
func DefaultCircuitBreakPolicy() CircuitBreakPolicy {
	return CircuitBreakPolicy{Strategy: CircuitBreakForceDraw}
}

// ValidateSnapshotConfig 结算前校验快照引用的配置项在快照版本仍存在（ADR-004 3.2）。
// 返回缺失配置项清单；空清单表示可正常结算。快照版本由引用计数保护，正常路径不应缺失，
// 缺失仅发生在运营强制发布删除类热更且未等待实例结束的场景（ADR-004 3.3强制发布）。
func ValidateSnapshotConfig(ctx context.Context, s *CombatSnapshot, query config.ConfigQueryAPI) ([]MissingConfigRef, error) {
	if s == nil {
		return nil, fmt.Errorf("战斗快照不能为空")
	}
	if query == nil {
		return nil, fmt.Errorf("配置查询接口不能为空")
	}
	var missing []MissingConfigRef
	// 战斗类型缺失：无法判定阶段流程，必须熔断
	exists, err := query.HasWithVersion(ctx, config.ExtPointCombatTypes, strconv.Itoa(s.CombatType), s.ConfigVersion)
	if err != nil {
		return nil, fmt.Errorf("校验战斗类型失败: %w", err)
	}
	if !exists {
		missing = append(missing, MissingConfigRef{ExtPointID: config.ExtPointCombatTypes, RefKey: strconv.Itoa(s.CombatType)})
	}
	// 伤害公式缺失：无法计算伤害，必须熔断（未绑定公式的战斗跳过校验，旧数据兼容）
	if s.FormulaID != "" {
		exists, err = query.HasWithVersion(ctx, config.ExtPointDamageFormulas, s.FormulaID, s.ConfigVersion)
		if err != nil {
			return nil, fmt.Errorf("校验伤害公式失败: %w", err)
		}
		if !exists {
			missing = append(missing, MissingConfigRef{ExtPointID: config.ExtPointDamageFormulas, RefKey: s.FormulaID})
		}
	}
	// 参战方兵种缺失：实体定义不成立，必须熔断（删除兵种即触发结算熔断，ADR-004场景A）
	for entityID, prop := range s.AttackerProps {
		exists, err = query.HasWithVersion(ctx, config.ExtPointUnitTypes, strconv.Itoa(prop.UnitType), s.ConfigVersion)
		if err != nil {
			return nil, fmt.Errorf("校验攻击方兵种失败: %w", err)
		}
		if !exists {
			missing = append(missing, MissingConfigRef{ExtPointID: config.ExtPointUnitTypes, RefKey: strconv.Itoa(prop.UnitType), InstanceID: entityID})
		}
	}
	for entityID, prop := range s.DefenderProps {
		exists, err = query.HasWithVersion(ctx, config.ExtPointUnitTypes, strconv.Itoa(prop.UnitType), s.ConfigVersion)
		if err != nil {
			return nil, fmt.Errorf("校验防守方兵种失败: %w", err)
		}
		if !exists {
			missing = append(missing, MissingConfigRef{ExtPointID: config.ExtPointUnitTypes, RefKey: strconv.Itoa(prop.UnitType), InstanceID: entityID})
		}
	}
	// 战利品规则缺失：影响经济，必须熔断（未绑定战利品规则的战斗跳过校验）
	if s.LootRuleID != "" {
		exists, err = query.HasWithVersion(ctx, config.ExtPointCombatLootRules, s.LootRuleID, s.ConfigVersion)
		if err != nil {
			return nil, fmt.Errorf("校验战利品规则失败: %w", err)
		}
		if !exists {
			missing = append(missing, MissingConfigRef{ExtPointID: config.ExtPointCombatLootRules, RefKey: s.LootRuleID})
		}
	}
	return missing, nil
}

// MissingConfigRef 结算校验缺失配置项记录。
type MissingConfigRef struct {
	ExtPointID string // 扩展点ID，如"combat.damage_formulas"
	RefKey     string // 配置项ID，如被删除的兵种ID
	InstanceID int64  // 引用实例ID（实体ID），非实体引用时为0
}

// ResolveCircuitBreak 按熔断策略决策缺失配置的结算路径，返回最终战斗结果（ADR-004 3.2）。
// 默认策略为强制平局（经济安全性优先）；兜底配置结算仅当策略显式配置且兜底项存在时使用。
// 调用方仅在ValidateSnapshotConfig返回非空缺失清单时调用。
func ResolveCircuitBreak(ctx context.Context, policy CircuitBreakPolicy, missing []MissingConfigRef, query config.ConfigQueryAPI, configVersion int64) (int, error) {
	// 无缺失 → 正常结算路径由调用方处理，本函数仅处理熔断决策
	if len(missing) == 0 {
		return 0, fmt.Errorf("无缺失配置项，无需熔断决策")
	}
	switch policy.Strategy {
	case CircuitBreakFallbackSettle:
		// 兜底结算：校验兜底公式在快照版本存在，否则回落强制平局
		if policy.FallbackFormulaID == "" {
			return ResultDraw, nil // 未预置兜底公式 → 回落强制平局，保证经济安全
		}
		hasFormula, err := query.HasWithVersion(ctx, config.ExtPointDamageFormulas, policy.FallbackFormulaID, configVersion)
		if err != nil {
			return 0, fmt.Errorf("校验兜底公式失败: %w", err)
		}
		if !hasFormula {
			return ResultDraw, nil // 兜底公式在快照版本缺失 → 回落强制平局，保证经济安全
		}
		return ResultAttackerWin, nil // 兜底结算按快照版本执行（调用方按战报标注"熔断-兜底"）
	case CircuitBreakForceDraw:
		return ResultDraw, nil // 强制平局，不发放战利品
	default:
		return 0, fmt.Errorf("未知熔断策略: %d", policy.Strategy)
	}
}

// CircuitBrokenEvent 结算熔断领域事件（combat.circuit_broken），供运营告警与E4验证报告统计。
// 熔断次数计入可观测性指标combat_circuit_break_total（ADR-004 3.2）。
type CircuitBrokenEvent struct {
	CombatID       int64              // 战斗ID
	MissingConfigs []MissingConfigRef // 缺失配置项清单
	Strategy       int                // 采用熔断策略：1=兜底配置结算 2=强制平局
	Result         int                // 最终战斗结果：1=攻击方胜 2=防守方胜 3=平局
}
