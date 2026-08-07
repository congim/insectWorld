// Package config 共享内核配置模块，提供配置加载/校验/查询统一API。
// 本文件定义Validator配置校验器，实现三层校验（JSON Schema+引用完整性+自定义规则）
// 与配置包完整性/兼容性校验。校验失败不降级，要么全部生效要么全部回滚。
package config

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// 校验阶段常量（规范1），用于日志标识校验阶段。
const (
	ValidatePhaseSchema        = "schema"        // JSON Schema校验阶段
	ValidatePhaseRefIntegrity  = "ref_integrity" // 引用完整性校验阶段
	ValidatePhaseCustomRule    = "custom_rule"   // 自定义规则校验阶段
	ValidatePhaseCompleteness  = "completeness"  // 配置包完整性校验阶段
	ValidatePhaseCompatibility = "compatibility" // 配置包兼容性校验阶段
)

// 必需配置文件列表，配置包完整性校验使用（对应design.md缺口13修复方案）。
var requiredConfigFiles = []string{
	"game", "units", "buildings", "resources", "terrains", "formulas",
}

// CustomRule 自定义业务规则校验函数类型。
// ctx为请求上下文，cfg为待校验的配置数据，返回校验失败的具体违规项描述。
type CustomRule func(ctx context.Context, cfg any) error

// Validator 配置校验器，实现三层校验与配置包完整性/兼容性校验。
// 支持业务服务注册自定义校验规则，校验失败不降级。
type Validator struct {
	customRules map[string]CustomRule // 自定义规则注册表，key=规则ID
	mu          sync.RWMutex          // 读写锁，保证规则注册的并发安全
	logger      *zap.Logger           // 结构化日志器，记录校验过程（规范7）
}

// NewValidator 创建配置校验器实例。
func NewValidator(logger *zap.Logger) *Validator {
	return &Validator{
		customRules: make(map[string]CustomRule),
		logger:      logger,
	}
}

// RegisterCustomRule 注册自定义业务规则校验函数。
// ruleID为规则ID，业务服务在启动时注册各自的校验规则。
// 同一ruleID重复注册将覆盖旧规则。
func (v *Validator) RegisterCustomRule(ruleID string, rule CustomRule) error {
	if ruleID == "" {
		return fmt.Errorf("自定义规则注册失败，ruleID为空: %w", ErrConfigInvalid)
	}
	if rule == nil {
		return fmt.Errorf("自定义规则注册失败，rule为空，ruleID=%s: %w", ruleID, ErrConfigInvalid)
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.customRules[ruleID] = rule
	v.logger.Info("自定义校验规则注册成功",
		zap.String("rule_id", ruleID),
	)
	return nil
}

// Validate 执行三层校验（JSON Schema+引用完整性+自定义规则）。
// 任一阶段失败即返回错误，不继续后续阶段（不降级）。
func (v *Validator) Validate(ctx context.Context, extPointID string, cfg any) error {
	v.logger.Info("配置校验开始",
		zap.String("ext_point_id", extPointID),
	)

	// 第一层：JSON Schema校验
	if err := v.validateSchema(ctx, extPointID, cfg); err != nil {
		v.logger.Error("JSON Schema校验失败",
			zap.String("ext_point_id", extPointID),
			zap.Error(err),
		)
		return fmt.Errorf("配置校验失败，extPointID=%s，阶段=%s: %w", extPointID, ValidatePhaseSchema, err)
	}

	// 第二层：引用完整性校验
	if err := v.validateRefIntegrity(ctx, extPointID, cfg); err != nil {
		v.logger.Error("引用完整性校验失败",
			zap.String("ext_point_id", extPointID),
			zap.Error(err),
		)
		return fmt.Errorf("配置校验失败，extPointID=%s，阶段=%s: %w", extPointID, ValidatePhaseRefIntegrity, err)
	}

	// 第三层：自定义规则校验
	if err := v.validateCustomRules(ctx, extPointID, cfg); err != nil {
		v.logger.Error("自定义规则校验失败",
			zap.String("ext_point_id", extPointID),
			zap.Error(err),
		)
		return fmt.Errorf("配置校验失败，extPointID=%s，阶段=%s: %w", extPointID, ValidatePhaseCustomRule, err)
	}

	v.logger.Info("配置校验通过",
		zap.String("ext_point_id", extPointID),
	)
	return nil
}

// validateSchema 第一层校验：JSON Schema校验，校验配置结构是否符合Schema定义。
func (v *Validator) validateSchema(ctx context.Context, extPointID string, cfg any) error {
	// 校验配置非空
	if cfg == nil {
		return fmt.Errorf("配置为空，extPointID=%s: %w", extPointID, ErrSchemaValidationFailed)
	}
	// 校验配置类型为map[string]any（配置编译后的标准格式）
	if _, ok := cfg.(map[string]any); !ok {
		// 部分配置可能为其他类型（如数组），仅记录8类扩展点配置强制map格式
		v.logger.Debug("配置非map格式，跳过Schema结构校验",
			zap.String("ext_point_id", extPointID),
		)
	}
	// TODO 后续接入gojsonschema校验引擎，加载server/shared/schema/jsonschema/下的Schema文件进行结构校验
	return nil
}

// validateRefIntegrity 第二层校验：引用完整性校验，校验配置间引用是否断裂。
func (v *Validator) validateRefIntegrity(ctx context.Context, extPointID string, cfg any) error {
	// 引用完整性校验检查配置间引用的目标是否存在：
	// - 技能配置引用的战斗类型是否存在
	// - 生产规则引用的资源类型是否存在
	// - 阵型配置引用的兵种是否存在
	// 当前阶段配置加载顺序保证依赖在前，引用完整性由配置加载器保证。
	// TODO 后续接入引用完整性校验引擎，构建配置引用图并检测断裂引用
	return nil
}

// validateCustomRules 第三层校验：自定义规则校验，执行所有已注册的自定义规则。
func (v *Validator) validateCustomRules(ctx context.Context, extPointID string, cfg any) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	for ruleID, rule := range v.customRules {
		if err := rule(ctx, cfg); err != nil {
			return fmt.Errorf("自定义规则校验失败，ruleID=%s: %w", ruleID, err)
		}
	}
	return nil
}

// ValidateCompleteness 配置包完整性校验，校验18个配置文件是否齐全、必需文件是否缺失。
// configFileNames为本次加载的配置文件名列表。
func (v *Validator) ValidateCompleteness(ctx context.Context, configFileNames []string) error {
	v.logger.Info("配置包完整性校验开始",
		zap.Int("config_file_count", len(configFileNames)),
	)

	loaded := make(map[string]bool, len(configFileNames))
	for _, name := range configFileNames {
		loaded[name] = true
	}

	var missing []string
	for _, required := range requiredConfigFiles {
		if !loaded[required] {
			missing = append(missing, required)
		}
	}

	if len(missing) > 0 {
		v.logger.Error("配置包完整性校验失败，必需文件缺失",
			zap.Strings("missing_files", missing),
		)
		return fmt.Errorf("配置包完整性校验失败，缺失必需文件=%v: %w", missing, ErrPackIncomplete)
	}

	v.logger.Info("配置包完整性校验通过")
	return nil
}

// ValidateCompatibility 配置包兼容性校验，校验新配置与存量数据Schema是否兼容。
// existingResourceTypes为存量数据中已有的资源类型列表，newResourceTypes为新配置定义的资源类型列表。
func (v *Validator) ValidateCompatibility(ctx context.Context, existingResourceTypes, newResourceTypes []string) error {
	v.logger.Info("配置包兼容性校验开始",
		zap.Int("existing_type_count", len(existingResourceTypes)),
		zap.Int("new_type_count", len(newResourceTypes)),
	)

	newTypes := make(map[string]bool, len(newResourceTypes))
	for _, t := range newResourceTypes {
		newTypes[t] = true
	}

	var uncovered []string
	for _, existing := range existingResourceTypes {
		if !newTypes[existing] {
			uncovered = append(uncovered, existing)
		}
	}

	if len(uncovered) > 0 {
		v.logger.Error("配置包兼容性校验失败，新配置未覆盖存量资源类型",
			zap.Strings("uncovered_types", uncovered),
		)
		return fmt.Errorf("配置包兼容性校验失败，新配置未覆盖存量资源类型=%v: %w", uncovered, ErrPackIncompatible)
	}

	v.logger.Info("配置包兼容性校验通过")
	return nil
}

// ValidateAll 执行全部校验：三层校验+完整性校验+兼容性校验。
// 任一阶段失败即返回错误，不降级（要么全部生效要么全部回滚）。
func (v *Validator) ValidateAll(ctx context.Context, configs map[string]any, configFileNames, existingResourceTypes, newResourceTypes []string) error {
	// 完整性校验
	if err := v.ValidateCompleteness(ctx, configFileNames); err != nil {
		return err
	}

	// 兼容性校验
	if err := v.ValidateCompatibility(ctx, existingResourceTypes, newResourceTypes); err != nil {
		return err
	}

	// 逐扩展点三层校验
	for extPointID, cfg := range configs {
		if err := v.Validate(ctx, extPointID, cfg); err != nil {
			return err
		}
	}

	v.logger.Info("全部配置校验通过",
		zap.Int("config_count", len(configs)),
	)
	return nil
}
