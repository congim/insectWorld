// Package config 共享内核配置模块，提供配置加载/校验/查询统一API。
// 本文件定义配置编译器，将配置包编译为内部数据结构并注册到ExtensionRegistry。
// 对应design.md 2.3.2.2节换皮配置驱动5步链路第一步。
package config

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ConfigCompiler 配置编译器，将配置包编译为内部数据结构并注册到ExtensionRegistry。
// 归属infrastructure层（规范3），实现domain层声明的配置编译接口。
type ConfigCompiler struct {
	registry  *ExtensionRegistry    // 扩展点注册表
	validator *Validator            // 配置校验器
	versioned *VersionedConfigStore // 版本化配置存储，可选注入；非空时编译结果按版本写入供快照类业务查询（ADR-004 3.1）
	logger    *zap.Logger           // 结构化日志器（规范7）
}

// NewConfigCompiler 创建配置编译器实例。
func NewConfigCompiler(registry *ExtensionRegistry, validator *Validator, logger *zap.Logger) *ConfigCompiler {
	return &ConfigCompiler{
		registry:  registry,
		validator: validator,
		logger:    logger,
	}
}

// SetVersionedStore 注入版本化配置存储，注入后编译结果按版本写入该存储（ADR-004 3.1）。
// store为nil时恢复不写入（旧行为，纯当前版本查询）。
func (cc *ConfigCompiler) SetVersionedStore(store *VersionedConfigStore) {
	cc.versioned = store
}

// CompileConfigPack 编译配置包，校验后注册到ExtensionRegistry。
// configPack为配置包数据（key=配置文件名，value=配置内容），configVersion为配置版本号。
func (cc *ConfigCompiler) CompileConfigPack(ctx context.Context, configPack map[string]any, configVersion int64) error {
	startTime := time.Now()

	cc.logger.Info("配置编译开始",
		zap.Int64("config_version", configVersion),
		zap.Int("file_count", len(configPack)),
	)

	// 1. 配置包完整性校验
	fileNames := make([]string, 0, len(configPack))
	for name := range configPack {
		fileNames = append(fileNames, name)
	}
	if err := cc.validator.ValidateCompleteness(ctx, fileNames); err != nil {
		return fmt.Errorf("配置编译失败，完整性校验未通过: %w", err)
	}

	// 2. 逐配置文件编译+校验+注册
	for name, cfg := range configPack {
		if err := cc.compileFile(ctx, name, cfg, configVersion); err != nil {
			return fmt.Errorf("配置编译失败，文件=%s: %w", name, err)
		}
	}

	cc.logger.Info("配置编译完成",
		zap.Int64("config_version", configVersion),
		zap.Int("compiled_files", len(configPack)),
		zap.Duration("compile_duration", time.Since(startTime)),
	)

	return nil
}

// compileFile 编译单个配置文件，校验后注册到ExtensionRegistry。
func (cc *ConfigCompiler) compileFile(ctx context.Context, fileName string, cfg any, configVersion int64) error {
	// 2.1 确定配置文件对应的扩展点列表
	extPoints := configFileToExtPoints(fileName)
	if len(extPoints) == 0 {
		cc.logger.Warn("配置文件无对应扩展点，跳过",
			zap.String("file_name", fileName),
		)
		return nil
	}

	// 2.2 三层校验
	for _, extPointID := range extPoints {
		if err := cc.validator.Validate(ctx, extPointID, cfg); err != nil {
			return fmt.Errorf("扩展点%s校验失败: %w", extPointID, err)
		}
	}

	// 2.3 注册到ExtensionRegistry（ID解析为指针在注册时完成）
	for _, extPointID := range extPoints {
		if err := cc.registry.Register(ctx, extPointID, cfg); err != nil {
			return fmt.Errorf("扩展点%s注册失败: %w", extPointID, err)
		}
	}

	// 2.4 写入版本化配置存储（可选，未注入store时跳过；ADR-004 3.1版本保留机制）
	if cc.versioned != nil {
		if items, ok := cfg.(map[string]any); ok {
			// 配置为键值结构（如map[string]any）时逐配置项写入，版本化查询可按配置项ID定位
			for _, extPointID := range extPoints {
				for k, v := range items {
					cc.versioned.PutEntry(configVersion, extPointID, k, v)
				}
			}
		} else {
			// 配置无法拆分为条目时退化存储整个扩展点值
			for _, extPointID := range extPoints {
				cc.versioned.PutExtPoint(configVersion, extPointID, cfg)
			}
		}
	}

	cc.logger.Debug("配置文件编译完成",
		zap.String("file_name", fileName),
		zap.Int64("config_version", configVersion),
		zap.Int("ext_point_count", len(extPoints)),
	)

	return nil
}

// configFileToExtPoints 配置文件名到扩展点ID列表的映射。
// 一个配置文件可对应多个扩展点（如units.json对应多个扩展点）。
func configFileToExtPoints(fileName string) []string {
	switch fileName {
	case "terrains":
		return []string{ExtPointTerrains}
	case "game":
		return []string{ExtPointMapVisionRules}
	case "movement":
		return []string{ExtPointMovementTypes, ExtPointMovementBlocking}
	case "combat":
		return []string{ExtPointCombatTypes, ExtPointCombatSkills, ExtPointCombatFormationEffects, ExtPointCombatLootRules}
	case "units":
		return []string{ExtPointUnitTypes}
	case "formulas":
		return []string{ExtPointDamageFormulas}
	case "counter_matrix":
		return []string{ExtPointCounterMatrix}
	case "economy":
		return []string{ExtPointProductionRules, ExtPointTradeRules, ExtPointConversionRules, ExtPointStorageRules, ExtPointEconomyModifiers, ExtPointCollectionRules}
	case "alliance":
		return []string{ExtPointAlliancePermissions, ExtPointAllianceWelfare}
	case "season":
		return []string{ExtPointSeasonPhases, ExtPointSeasonTransition, ExtPointSeasonReset, ExtPointSeasonInherit, ExtPointSeasonRewards, ExtPointSeasonScoring}
	case "event":
		return []string{ExtPointEventTypes}
	case "rules":
		return []string{ExtPointEventTriggerRules}
	default:
		return nil
	}
}
