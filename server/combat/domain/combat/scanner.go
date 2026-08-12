// Package combat 战斗聚合根，维护战斗状态与轮次执行。
// 本文件实现战斗实例配置引用扫描器（ADR-004 3.3 ConfigDependencyScanner的Combat实现）：
// 供Config Service热更删除类变更发布前预检汇聚——扫描进行中战斗对配置项的引用，
// 有引用则阻塞删除类热更或标记"待熔断"，把引用断裂风险前置到发布时（风险前置，非结算兜底）。
package combat

import (
	"context"
	"fmt"
	"strconv"

	"insectworld/server/shared/pkg/config"
	"insectworld/server/shared/pkg/configdeps"
)

// CombatRefScanner 战斗实例配置引用扫描器（ADR-004 3.3）。
// 实现configdeps.ConfigDependencyScanner接口；归属domain层纯逻辑（规范3），
// 进行中战斗列表由调用方（infrastructure/application）经loadInProgress注入，domain不依赖外部包。
type CombatRefScanner struct {
	loadInProgress func(ctx context.Context) ([]*Combat, error) // 进行中战斗加载函数，由infrastructure注入；nil时视为无进行中战斗
}

// NewCombatRefScanner 创建战斗实例配置引用扫描器实例。
// loadInProgress为进行中战斗加载函数（如CombatRepository.LoadInProgress），nil表示当前无进行中战斗。
func NewCombatRefScanner(loadInProgress func(ctx context.Context) ([]*Combat, error)) *CombatRefScanner {
	return &CombatRefScanner{loadInProgress: loadInProgress}
}

// ScanCombatRefs 扫描进行中战斗对指定配置项ID的引用（公式/技能/战利品/兵种/阵型/战斗类型，ADR-004 3.3）。
// deletedIDs为本次热更删除的配置项ID清单；返回引用记录，无引用返回空切片。
// 快照类版本归属：每场战斗以快照冻结的ConfigVersion为基准（ADR-004 3.1），
// 删除该版本的引用项即触发预检阻塞；运营强制发布后结算走熔断协议（ADR-004 3.2）。
func (s *CombatRefScanner) ScanCombatRefs(ctx context.Context, deletedIDs []string) ([]configdeps.InstanceRef, error) {
	deleted := make(map[string]struct{}, len(deletedIDs))
	for _, id := range deletedIDs {
		deleted[id] = struct{}{}
	}
	if len(deleted) == 0 {
		return nil, nil
	}
	if s.loadInProgress == nil {
		return nil, nil
	}

	combats, err := s.loadInProgress(ctx)
	if err != nil {
		return nil, fmt.Errorf("加载进行中战斗失败: %w", err)
	}
	var refs []configdeps.InstanceRef
	for _, c := range combats {
		if c == nil {
			continue
		}
		refs = append(refs, scanCombatRefs(c, deleted)...)
	}
	return refs, nil
}

// scanCombatRefs 扫描单场进行中战斗对删除配置项的引用。
// 引用域覆盖战斗类型/伤害公式/战利品规则/技能/兵种/阵型，与ValidateSnapshotConfig口径一致（ADR-004 3.2），
// 保证"预检发现的引用项"与"结算校验的缺失项"同一基准，无漏扫。
func scanCombatRefs(c *Combat, deleted map[string]struct{}) []configdeps.InstanceRef {
	var refs []configdeps.InstanceRef
	snap := c.snapshot
	version := snap.ConfigVersion

	// 战斗类型（combat.json#combat_types）：开战时冻结，删除战斗类型即战斗流程不可判定
	if _, ok := deleted[strconv.Itoa(snap.CombatType)]; ok {
		refs = append(refs, configdeps.InstanceRef{
			InstanceType:  configdeps.InstanceTypeCombat,
			InstanceID:    c.combatID,
			RefExtPoint:   config.ExtPointCombatTypes,
			RefKey:        strconv.Itoa(snap.CombatType),
			ConfigVersion: version,
		})
	}
	// 伤害公式（formulas.json#damage_formulas）：空表示未绑定公式，跳过（旧数据兼容，与结算校验一致）
	if snap.FormulaID != "" {
		if _, ok := deleted[snap.FormulaID]; ok {
			refs = append(refs, configdeps.InstanceRef{
				InstanceType:  configdeps.InstanceTypeCombat,
				InstanceID:    c.combatID,
				RefExtPoint:   config.ExtPointDamageFormulas,
				RefKey:        snap.FormulaID,
				ConfigVersion: version,
			})
		}
	}
	// 战利品规则（combat.json#combat_loot_rules）：空表示无战利品规则，跳过
	if snap.LootRuleID != "" {
		if _, ok := deleted[snap.LootRuleID]; ok {
			refs = append(refs, configdeps.InstanceRef{
				InstanceType:  configdeps.InstanceTypeCombat,
				InstanceID:    c.combatID,
				RefExtPoint:   config.ExtPointCombatLootRules,
				RefKey:        snap.LootRuleID,
				ConfigVersion: version,
			})
		}
	}
	// 技能（combat.json#combat_skills）：开战时冻结的启用技能列表
	for _, skillID := range snap.SkillIDs {
		if _, ok := deleted[skillID]; ok {
			refs = append(refs, configdeps.InstanceRef{
				InstanceType:  configdeps.InstanceTypeCombat,
				InstanceID:    c.combatID,
				RefExtPoint:   config.ExtPointCombatSkills,
				RefKey:        skillID,
				ConfigVersion: version,
			})
		}
	}
	// 参战方兵种（units.json#entity_types）：删除兵种即触发结算熔断（ADR-004场景A），预检前置拦截
	for _, prop := range snap.AttackerProps {
		if _, ok := deleted[strconv.Itoa(prop.UnitType)]; ok {
			refs = append(refs, configdeps.InstanceRef{
				InstanceType:  configdeps.InstanceTypeCombat,
				InstanceID:    c.combatID,
				RefExtPoint:   config.ExtPointUnitTypes,
				RefKey:        strconv.Itoa(prop.UnitType),
				ConfigVersion: version,
			})
		}
	}
	for _, prop := range snap.DefenderProps {
		if _, ok := deleted[strconv.Itoa(prop.UnitType)]; ok {
			refs = append(refs, configdeps.InstanceRef{
				InstanceType:  configdeps.InstanceTypeCombat,
				InstanceID:    c.combatID,
				RefExtPoint:   config.ExtPointUnitTypes,
				RefKey:        strconv.Itoa(prop.UnitType),
				ConfigVersion: version,
			})
		}
	}
	// 阵型（combat.json#combat_formation_effects）：0表示无阵型，跳过
	if c.formationID != 0 {
		key := strconv.Itoa(int(c.formationID))
		if _, ok := deleted[key]; ok {
			refs = append(refs, configdeps.InstanceRef{
				InstanceType:  configdeps.InstanceTypeCombat,
				InstanceID:    c.combatID,
				RefExtPoint:   config.ExtPointCombatFormationEffects,
				RefKey:        key,
				ConfigVersion: version,
			})
		}
	}
	return refs
}

// ScanMovementRefs 行军引用扫描（ADR-004 3.3接口契约）。
// 战斗服务不持有行军实例，返回空（无引用）；行军扫描器归属Social服务，P3移动骨头落地后实现。
func (s *CombatRefScanner) ScanMovementRefs(ctx context.Context, deletedIDs []string) ([]configdeps.InstanceRef, error) {
	return nil, nil
}

// ScanProductionRefs 生产队列引用扫描（ADR-004 3.3接口契约）。
// 战斗服务不持有生产队列，返回空（无引用）；生产扫描器归属Economy服务，P3生产骨头落地后实现。
func (s *CombatRefScanner) ScanProductionRefs(ctx context.Context, deletedIDs []string) ([]configdeps.InstanceRef, error) {
	return nil, nil
}
