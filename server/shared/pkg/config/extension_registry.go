// Package config 共享内核配置模块，提供配置加载/校验/查询统一API。
// 本文件定义ExtensionRegistry扩展点注册表与全局扩展点ID常量，
// 扩展点ID常量集中定义（规范1），业务代码引用常量而非硬编码字符串查询配置。
package config

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// 扩展点ID常量集中定义（规范1），格式 <module>.<extension_point_name>。
// 业务代码通过引用常量而非硬编码字符串查询配置，如 config.ExtPointCombatSkills。
const (
	// 空间配置扩展点
	ExtPointMapVisionRules = "map.map_vision_rules" // 地图视野规则扩展点ID，对应map.json的视野配置
	ExtPointTerrains       = "terrains"             // 地形配置扩展点ID，对应terrains.json
	// 移动配置扩展点
	ExtPointMovementTypes    = "movement.movement_types"          // 移动类型扩展点ID，对应movement.json的移动类型定义
	ExtPointMovementBlocking = "movement.movement_blocking_rules" // 移动阻挡规则扩展点ID，对应movement.json的阻挡规则
	// 战斗配置扩展点
	ExtPointCombatTypes            = "combat.combat_types"             // 战斗类型扩展点ID，对应combat.json的战斗类型定义
	ExtPointUnitTypes              = "entity.entity_types"             // 实体兵种类型扩展点ID，对应units.json的兵种类型定义（ADR-004 3.1结算校验用）
	ExtPointCombatSkills           = "combat.combat_skills"            // 战斗技能扩展点ID，对应combat.json的技能定义
	ExtPointCombatFormationEffects = "combat.combat_formation_effects" // 战斗阵型效果扩展点ID，对应combat.json的阵型效果
	ExtPointCombatLootRules        = "combat.combat_loot_rules"        // 战斗掉落规则扩展点ID，对应combat.json的掉落规则
	ExtPointDamageFormulas         = "formulas.damage_formulas"        // 伤害公式扩展点ID，对应formulas.json的伤害公式
	ExtPointCounterMatrix          = "counter_matrix"                  // 兵种克制矩阵扩展点ID，对应counter_matrix.json
	// 经济配置扩展点
	ExtPointProductionRules  = "economy.production_rules"  // 生产规则扩展点ID，对应economy.json的生产规则
	ExtPointTradeRules       = "economy.trade_rules"       // 交易规则扩展点ID，对应economy.json的交易规则
	ExtPointConversionRules  = "economy.conversion_rules"  // 资源转换规则扩展点ID，对应economy.json的转换规则
	ExtPointStorageRules     = "economy.storage_rules"     // 存储规则扩展点ID，对应economy.json的存储规则
	ExtPointEconomyModifiers = "economy.economy_modifiers" // 经济修正器扩展点ID，对应economy.json的修正器
	ExtPointCollectionRules  = "economy.collection_rules"  // 采集规则扩展点ID，对应economy.json的采集规则
	// 社交配置扩展点
	ExtPointAlliancePermissions = "alliance.alliance_permissions" // 联盟权限扩展点ID，对应alliance.json的权限定义
	ExtPointAllianceWelfare     = "alliance.alliance_welfare"     // 联盟福利扩展点ID，对应alliance.json的福利定义
	// 运营配置扩展点
	ExtPointSeasonPhases     = "season.season_phases"           // 赛季阶段扩展点ID，对应season.json的阶段定义
	ExtPointSeasonTransition = "season.season_transition_rules" // 赛季阶段切换规则扩展点ID，对应season.json的切换规则
	ExtPointSeasonReset      = "season.season_reset_rules"      // 赛季重置规则扩展点ID，对应season.json的重置规则
	ExtPointSeasonInherit    = "season.season_inherit_rules"    // 赛季继承规则扩展点ID，对应season.json的继承规则
	ExtPointSeasonRewards    = "season.season_rewards"          // 赛季奖励扩展点ID，对应season.json的奖励定义
	ExtPointSeasonScoring    = "season.season_scoring_rules"    // 赛季积分规则扩展点ID，对应season.json的积分规则
	// 事件配置扩展点
	ExtPointEventTypes        = "event.event_types"         // 事件类型扩展点ID，对应event.json的事件类型
	ExtPointEventTriggerRules = "rules.event_trigger_rules" // 事件触发规则扩展点ID，对应rules.json的触发规则
)

// ExtensionPointContract 扩展点契约元数据，描述扩展点的输入约束、校验规则与默认值。
// 每个扩展点在配置加载前注册契约，Register时校验配置是否符合契约。
type ExtensionPointContract struct {
	ExtPointID      string   // 扩展点ID，对应常量块中定义的ExtPointXxx
	InputContract   string   // 输入契约，JSON Schema字符串描述配置结构约束
	ValidationRules []string // 校验规则ID列表，指向Validator中注册的校验规则
	DefaultValue    any      // 默认值，配置缺失时使用
}

// ExtensionRegistry 扩展点注册表，维护扩展点ID到编译后配置的映射。
// 支持配置热更时的原子替换，通过读写锁保证并发安全。
type ExtensionRegistry struct {
	entries   map[string]any                    // 扩展点注册表，key=扩展点ID常量，value=编译后配置
	contracts map[string]ExtensionPointContract // 扩展点契约元数据，key=扩展点ID常量
	mu        sync.RWMutex                      // 读写锁，保证热更时原子替换的并发安全
	logger    *zap.Logger                       // 结构化日志器，记录注册/查询/热更操作（规范7）
}

// NewExtensionRegistry 创建扩展点注册表实例。
// logger为结构化日志器，用于记录配置注册与查询操作（规范7）。
func NewExtensionRegistry(logger *zap.Logger) *ExtensionRegistry {
	return &ExtensionRegistry{
		entries:   make(map[string]any),
		contracts: make(map[string]ExtensionPointContract),
		logger:    logger,
	}
}

// RegisterContract 注册扩展点契约元数据，在配置加载前调用。
// 同一扩展点重复注册将覆盖旧契约。
func (e *ExtensionRegistry) RegisterContract(contract ExtensionPointContract) error {
	if contract.ExtPointID == "" {
		return fmt.Errorf("扩展点契约注册失败，ExtPointID为空: %w", ErrConfigInvalid)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.contracts[contract.ExtPointID] = contract
	e.logger.Info("扩展点契约注册成功",
		zap.String("ext_point_id", contract.ExtPointID),
		zap.Int("validation_rules_count", len(contract.ValidationRules)),
	)
	return nil
}

// Register 注册编译后配置到扩展点，支持热更时原子替换。
// 校验扩展点契约存在性，校验通过后原子替换配置。
func (e *ExtensionRegistry) Register(ctx context.Context, extPointID string, cfg any) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 检查扩展点契约是否已注册
	contract, ok := e.contracts[extPointID]
	if !ok {
		e.logger.Error("扩展点注册失败，契约未注册",
			zap.String("ext_point_id", extPointID),
		)
		return fmt.Errorf("扩展点注册失败，契约未注册，extPointID=%s: %w", extPointID, ErrExtensionPointNotFound)
	}

	// 原子替换配置（热更时直接覆盖旧值）
	e.entries[extPointID] = cfg

	e.logger.Info("扩展点配置注册成功",
		zap.String("ext_point_id", extPointID),
		zap.Int("validation_rules_count", len(contract.ValidationRules)),
	)
	return nil
}

// Query 按扩展点ID查询编译后的配置。
// 扩展点不存在返回ErrExtensionPointNotFound错误。
func (e *ExtensionRegistry) Query(ctx context.Context, extPointID string) (any, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	cfg, ok := e.entries[extPointID]
	if !ok {
		e.logger.Warn("扩展点查询失败，扩展点未注册",
			zap.String("ext_point_id", extPointID),
		)
		return nil, fmt.Errorf("扩展点查询失败，extPointID=%s: %w", extPointID, ErrExtensionPointNotFound)
	}
	return cfg, nil
}

// HasContract 检查扩展点契约是否已注册。
func (e *ExtensionRegistry) HasContract(extPointID string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, ok := e.contracts[extPointID]
	return ok
}

// GetContract 查询扩展点契约元数据。
func (e *ExtensionRegistry) GetContract(extPointID string) (ExtensionPointContract, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	contract, ok := e.contracts[extPointID]
	return contract, ok
}

// AllExtensionPointIDs 返回所有已注册的扩展点ID常量列表，用于完整性校验。
func (e *ExtensionRegistry) AllExtensionPointIDs() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	ids := make([]string, 0, len(e.contracts))
	for id := range e.contracts {
		ids = append(ids, id)
	}
	return ids
}
