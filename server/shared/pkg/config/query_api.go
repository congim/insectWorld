// Package config 共享内核配置模块，提供配置加载/校验/查询统一API。
// 本文件定义ConfigQueryAPI接口，业务服务通过此接口从本地缓存查询配置驱动执行。
// 接口在共享内核domain层声明（规范3），实现在infrastructure层，application层通过接口调用。
package config

import "context"

// ConfigQueryAPI 配置查询统一API，业务服务从本地缓存查询配置驱动执行。
// 配置走本地缓存（不读etcd），保证查询延迟<1ms。
// 接口在domain层声明，实现在infrastructure层通过ExtensionRegistry查询编译后配置。
type ConfigQueryAPI interface {
	// QueryByExtensionPoint 按扩展点ID查询编译后的配置，统一查询入口。
	// extPointID应使用ExtPointXxx常量而非硬编码字符串（规范1）。
	// 扩展点不存在返回ErrExtensionPointNotFound。
	QueryByExtensionPoint(ctx context.Context, extPointID string) (any, error)

	// GetTerrain 查询地形配置，terrainID为地形类型ID，不存在返回nil。
	GetTerrain(ctx context.Context, terrainID string) *TerrainConfig

	// GetMovementType 查询移动类型配置，typeID为移动类型ID，不存在返回nil。
	GetMovementType(ctx context.Context, typeID string) *MovementTypeConfig

	// GetCombatType 查询战斗类型配置，typeID为战斗类型ID，不存在返回nil。
	GetCombatType(ctx context.Context, typeID string) *CombatTypeConfig

	// GetCombatSkill 查询技能配置，skillID为技能ID，不存在返回nil。
	GetCombatSkill(ctx context.Context, skillID string) *SkillConfig

	// GetFormationEffect 查询阵型效果配置，formationID为阵型ID，不存在返回nil。
	GetFormationEffect(ctx context.Context, formationID string) *FormationEffectConfig

	// GetDamageFormula 查询伤害公式配置，formulaID为公式ID，不存在返回nil。
	GetDamageFormula(ctx context.Context, formulaID string) *FormulaConfig

	// GetCounterMatrix 查询兵种克制矩阵配置，不存在返回nil。
	GetCounterMatrix(ctx context.Context) *CounterMatrixConfig

	// GetProductionRule 查询生产规则配置，ruleID为规则ID，不存在返回nil。
	GetProductionRule(ctx context.Context, ruleID string) *ProductionRuleConfig

	// GetTradeRule 查询交易规则配置，ruleID为规则ID，不存在返回nil。
	GetTradeRule(ctx context.Context, ruleID string) *TradeRuleConfig

	// GetConversionRule 查询资源转换规则配置，ruleID为规则ID，不存在返回nil。
	GetConversionRule(ctx context.Context, ruleID string) *ConversionRuleConfig

	// GetStorageRule 查询存储规则配置，resourceType为资源类型，不存在返回nil。
	GetStorageRule(ctx context.Context, resourceType string) *StorageRuleConfig

	// GetEconomyModifiers 查询经济修正器配置，不存在返回nil。
	GetEconomyModifiers(ctx context.Context) *EconomyModifiersConfig

	// GetAlliancePermissions 查询联盟权限配置，返回职位→权限列表映射，不存在返回nil。
	GetAlliancePermissions(ctx context.Context) map[string][]string

	// GetAllianceWelfare 查询联盟福利配置，welfareID为福利ID，不存在返回nil。
	GetAllianceWelfare(ctx context.Context, welfareID string) *WelfareConfig

	// GetSeasonPhases 查询赛季阶段配置列表，不存在返回nil。
	GetSeasonPhases(ctx context.Context) []PhaseConfig

	// GetSeasonTransitionRules 查询赛季阶段切换规则列表，不存在返回nil。
	GetSeasonTransitionRules(ctx context.Context) []TransitionRuleConfig

	// GetSeasonResetRules 查询赛季重置规则配置，不存在返回nil。
	GetSeasonResetRules(ctx context.Context) *ResetRulesConfig

	// GetSeasonInheritRules 查询赛季继承规则列表，不存在返回nil。
	GetSeasonInheritRules(ctx context.Context) []InheritRuleConfig

	// GetSeasonRewards 查询赛季奖励配置列表，不存在返回nil。
	GetSeasonRewards(ctx context.Context) []RewardConfig

	// GetSeasonScoringRules 查询赛季积分规则列表，不存在返回nil。
	GetSeasonScoringRules(ctx context.Context) []ScoringRuleConfig

	// GetEventTypes 查询事件类型配置列表，不存在返回nil。
	GetEventTypes(ctx context.Context) []EventTypeConfig

	// GetMaxRounds 查询战斗类型的最大轮数，combatType为战斗类型ID，不存在返回0。
	GetMaxRounds(ctx context.Context, combatType string) int

	// GetWithVersion 按指定配置版本查询配置项，供快照类业务以冻结版本读取配置（ADR-004 3.1）。
	// 战斗/生产队列等快照类业务全程用开战/开始时的configVersion查询，与当前热更版本解耦。
	// 版本不可用返回ErrConfigVersionGone；版本存在但配置项缺失返回nil（存在性用HasWithVersion判断）。
	GetWithVersion(ctx context.Context, extPointID string, key string, configVersion int64) (any, error)

	// HasWithVersion 判断指定配置版本中配置项是否存在，供结算校验使用（ADR-004 3.2）。
	// 版本不可用返回ErrConfigVersionGone；配置项存在返回true，否则false。
	HasWithVersion(ctx context.Context, extPointID string, key string, configVersion int64) (bool, error)

	// PinVersion 锁定配置版本，引用计数+1；快照类业务创建时调用（ADR-004 3.1版本保留机制）。
	// 进行中实例引用的版本不被历史滚动清理，保证回滚后版本仍在。
	PinVersion(ctx context.Context, configVersion int64) error

	// UnpinVersion 释放配置版本引用，引用计数-1；业务结束时调用（ADR-004 3.4双向回滚语义）。
	// 归零且超出保留上限后允许清理。
	UnpinVersion(ctx context.Context, configVersion int64) error
}
