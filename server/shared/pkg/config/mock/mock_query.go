// Package mock 配置查询接口的测试mock，供各服务单元测试使用。
// MockConfigQuery实现config.ConfigQueryAPI接口，测试中按需设置返回值。
package mock

import (
	"context"

	"insectworld/server/shared/pkg/config"
)

// MockConfigQuery ConfigQueryAPI的mock实现，测试中按需设置各字段返回值。
type MockConfigQuery struct {
	Terrain               map[string]*config.TerrainConfig         // 地形配置mock
	MovementType          map[string]*config.MovementTypeConfig    // 移动类型配置mock
	CombatType            map[string]*config.CombatTypeConfig      // 战斗类型配置mock
	CombatSkill           map[string]*config.SkillConfig           // 技能配置mock
	FormationEffect       map[string]*config.FormationEffectConfig // 阵型效果配置mock
	DamageFormula         map[string]*config.FormulaConfig         // 伤害公式配置mock
	CounterMatrix         *config.CounterMatrixConfig              // 兵种克制矩阵mock
	ProductionRule        map[string]*config.ProductionRuleConfig  // 生产规则配置mock
	TradeRule             map[string]*config.TradeRuleConfig       // 交易规则配置mock
	ConversionRule        map[string]*config.ConversionRuleConfig  // 资源转换规则mock
	StorageRule           map[string]*config.StorageRuleConfig     // 存储规则配置mock
	EconomyModifiers      *config.EconomyModifiersConfig           // 经济修正器配置mock
	AlliancePermissions   map[string][]string                      // 联盟权限配置mock
	AllianceWelfare       map[string]*config.WelfareConfig         // 联盟福利配置mock
	SeasonPhases          []config.PhaseConfig                     // 赛季阶段配置mock
	SeasonTransitionRules []config.TransitionRuleConfig            // 赛季阶段切换规则mock
	SeasonResetRules      *config.ResetRulesConfig                 // 赛季重置规则mock
	SeasonInheritRules    []config.InheritRuleConfig               // 赛季继承规则mock
	SeasonRewards         []config.RewardConfig                    // 赛季奖励配置mock
	SeasonScoringRules    []config.ScoringRuleConfig               // 赛季积分规则mock
	EventTypes            []config.EventTypeConfig                 // 事件类型配置mock
	MaxRounds             map[string]int                           // 战斗最大轮数mock
}

// NewMockConfigQuery 创建mock实例，初始化所有map。
func NewMockConfigQuery() *MockConfigQuery {
	return &MockConfigQuery{
		Terrain:         make(map[string]*config.TerrainConfig),
		MovementType:    make(map[string]*config.MovementTypeConfig),
		CombatType:      make(map[string]*config.CombatTypeConfig),
		CombatSkill:     make(map[string]*config.SkillConfig),
		FormationEffect: make(map[string]*config.FormationEffectConfig),
		DamageFormula:   make(map[string]*config.FormulaConfig),
		ProductionRule:  make(map[string]*config.ProductionRuleConfig),
		TradeRule:       make(map[string]*config.TradeRuleConfig),
		ConversionRule:  make(map[string]*config.ConversionRuleConfig),
		StorageRule:     make(map[string]*config.StorageRuleConfig),
		AllianceWelfare: make(map[string]*config.WelfareConfig),
		MaxRounds:       make(map[string]int),
	}
}

// QueryByExtensionPoint 按扩展点ID查询的mock实现，返回nil。
func (m *MockConfigQuery) QueryByExtensionPoint(ctx context.Context, extPointID string) (any, error) {
	return nil, nil
}

// GetTerrain 查询地形配置的mock实现。
func (m *MockConfigQuery) GetTerrain(ctx context.Context, terrainID string) *config.TerrainConfig {
	return m.Terrain[terrainID]
}

// GetMovementType 查询移动类型配置的mock实现。
func (m *MockConfigQuery) GetMovementType(ctx context.Context, typeID string) *config.MovementTypeConfig {
	return m.MovementType[typeID]
}

// GetCombatType 查询战斗类型配置的mock实现。
func (m *MockConfigQuery) GetCombatType(ctx context.Context, typeID string) *config.CombatTypeConfig {
	return m.CombatType[typeID]
}

// GetCombatSkill 查询技能配置的mock实现。
func (m *MockConfigQuery) GetCombatSkill(ctx context.Context, skillID string) *config.SkillConfig {
	return m.CombatSkill[skillID]
}

// GetFormationEffect 查询阵型效果配置的mock实现。
func (m *MockConfigQuery) GetFormationEffect(ctx context.Context, formationID string) *config.FormationEffectConfig {
	return m.FormationEffect[formationID]
}

// GetDamageFormula 查询伤害公式配置的mock实现。
func (m *MockConfigQuery) GetDamageFormula(ctx context.Context, formulaID string) *config.FormulaConfig {
	return m.DamageFormula[formulaID]
}

// GetCounterMatrix 查询兵种克制矩阵配置的mock实现。
func (m *MockConfigQuery) GetCounterMatrix(ctx context.Context) *config.CounterMatrixConfig {
	return m.CounterMatrix
}

// GetProductionRule 查询生产规则配置的mock实现。
func (m *MockConfigQuery) GetProductionRule(ctx context.Context, ruleID string) *config.ProductionRuleConfig {
	return m.ProductionRule[ruleID]
}

// GetTradeRule 查询交易规则配置的mock实现。
func (m *MockConfigQuery) GetTradeRule(ctx context.Context, ruleID string) *config.TradeRuleConfig {
	return m.TradeRule[ruleID]
}

// GetConversionRule 查询资源转换规则配置的mock实现。
func (m *MockConfigQuery) GetConversionRule(ctx context.Context, ruleID string) *config.ConversionRuleConfig {
	return m.ConversionRule[ruleID]
}

// GetStorageRule 查询存储规则配置的mock实现。
func (m *MockConfigQuery) GetStorageRule(ctx context.Context, resourceType string) *config.StorageRuleConfig {
	return m.StorageRule[resourceType]
}

// GetEconomyModifiers 查询经济修正器配置的mock实现。
func (m *MockConfigQuery) GetEconomyModifiers(ctx context.Context) *config.EconomyModifiersConfig {
	return m.EconomyModifiers
}

// GetAlliancePermissions 查询联盟权限配置的mock实现。
func (m *MockConfigQuery) GetAlliancePermissions(ctx context.Context) map[string][]string {
	return m.AlliancePermissions
}

// GetAllianceWelfare 查询联盟福利配置的mock实现。
func (m *MockConfigQuery) GetAllianceWelfare(ctx context.Context, welfareID string) *config.WelfareConfig {
	return m.AllianceWelfare[welfareID]
}

// GetSeasonPhases 查询赛季阶段配置列表的mock实现。
func (m *MockConfigQuery) GetSeasonPhases(ctx context.Context) []config.PhaseConfig {
	return m.SeasonPhases
}

// GetSeasonTransitionRules 查询赛季阶段切换规则列表的mock实现。
func (m *MockConfigQuery) GetSeasonTransitionRules(ctx context.Context) []config.TransitionRuleConfig {
	return m.SeasonTransitionRules
}

// GetSeasonResetRules 查询赛季重置规则配置的mock实现。
func (m *MockConfigQuery) GetSeasonResetRules(ctx context.Context) *config.ResetRulesConfig {
	return m.SeasonResetRules
}

// GetSeasonInheritRules 查询赛季继承规则列表的mock实现。
func (m *MockConfigQuery) GetSeasonInheritRules(ctx context.Context) []config.InheritRuleConfig {
	return m.SeasonInheritRules
}

// GetSeasonRewards 查询赛季奖励配置列表的mock实现。
func (m *MockConfigQuery) GetSeasonRewards(ctx context.Context) []config.RewardConfig {
	return m.SeasonRewards
}

// GetSeasonScoringRules 查询赛季积分规则列表的mock实现。
func (m *MockConfigQuery) GetSeasonScoringRules(ctx context.Context) []config.ScoringRuleConfig {
	return m.SeasonScoringRules
}

// GetEventTypes 查询事件类型配置列表的mock实现。
func (m *MockConfigQuery) GetEventTypes(ctx context.Context) []config.EventTypeConfig {
	return m.EventTypes
}

// GetMaxRounds 查询战斗类型最大轮数的mock实现。
func (m *MockConfigQuery) GetMaxRounds(ctx context.Context, combatType string) int {
	return m.MaxRounds[combatType]
}
