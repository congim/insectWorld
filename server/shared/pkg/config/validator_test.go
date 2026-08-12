// Package config 共享内核配置模块，提供配置加载/校验/查询统一API。
// 本文件定义Validator的单元测试，测试三层校验/完整性校验/兼容性校验。
package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestValidator_Validate_Success 测试三层校验通过。
func TestValidator_Validate_Success(t *testing.T) {
	logger := zap.NewNop()
	validator := NewValidator(logger)
	ctx := context.Background()

	err := validator.Validate(ctx, ExtPointCombatSkills, map[string]any{"skill1": "fireball"})
	assert.NoError(t, err)
}

// TestValidator_Validate_NilConfig 测试空配置校验失败。
func TestValidator_Validate_NilConfig(t *testing.T) {
	logger := zap.NewNop()
	validator := NewValidator(logger)
	ctx := context.Background()

	err := validator.Validate(ctx, ExtPointCombatSkills, nil)
	assert.Error(t, err)
}

// TestValidator_RegisterCustomRule 测试注册自定义校验规则。
func TestValidator_RegisterCustomRule(t *testing.T) {
	logger := zap.NewNop()
	validator := NewValidator(logger)

	rule := func(ctx context.Context, cfg any) error { return nil }
	err := validator.RegisterCustomRule("combat_skill_consistency", rule)
	assert.NoError(t, err)
}

// TestValidator_RegisterCustomRule_EmptyID 测试注册空规则ID。
func TestValidator_RegisterCustomRule_EmptyID(t *testing.T) {
	logger := zap.NewNop()
	validator := NewValidator(logger)

	rule := func(ctx context.Context, cfg any) error { return nil }
	err := validator.RegisterCustomRule("", rule)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrConfigInvalid)
}

// TestValidator_RegisterCustomRule_NilRule 测试注册空规则函数。
func TestValidator_RegisterCustomRule_NilRule(t *testing.T) {
	logger := zap.NewNop()
	validator := NewValidator(logger)

	err := validator.RegisterCustomRule("test_rule", nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrConfigInvalid)
}

// TestValidator_ValidateCompleteness_Success 测试完整性校验通过。
func TestValidator_ValidateCompleteness_Success(t *testing.T) {
	logger := zap.NewNop()
	validator := NewValidator(logger)
	ctx := context.Background()

	// 包含全部必需文件
	files := []string{"game", "units", "buildings", "resources", "terrains", "formulas", "combat", "movement"}
	err := validator.ValidateCompleteness(ctx, files)
	assert.NoError(t, err)
}

// TestValidator_ValidateCompleteness_MissingFiles 测试完整性校验失败（缺失必需文件）。
func TestValidator_ValidateCompleteness_MissingFiles(t *testing.T) {
	logger := zap.NewNop()
	validator := NewValidator(logger)
	ctx := context.Background()

	// 缺失terrains和formulas
	files := []string{"game", "units", "buildings", "resources"}
	err := validator.ValidateCompleteness(ctx, files)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrPackIncomplete)
}

// TestValidator_ValidateCompatibility_Success 测试兼容性校验通过。
func TestValidator_ValidateCompatibility_Success(t *testing.T) {
	logger := zap.NewNop()
	validator := NewValidator(logger)
	ctx := context.Background()

	existing := []string{"gold", "wood", "food"}
	newTypes := []string{"gold", "wood", "food", "stone"}
	err := validator.ValidateCompatibility(ctx, existing, newTypes)
	assert.NoError(t, err)
}

// TestValidator_ValidateCompatibility_Fail 测试兼容性校验失败（新配置未覆盖存量资源类型）。
func TestValidator_ValidateCompatibility_Fail(t *testing.T) {
	logger := zap.NewNop()
	validator := NewValidator(logger)
	ctx := context.Background()

	existing := []string{"gold", "wood", "food"}
	newTypes := []string{"gold", "wood"} // 缺少food
	err := validator.ValidateCompatibility(ctx, existing, newTypes)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrPackIncompatible)
}

// TestConfigCompiler_CompileConfigPack 测试配置编译器。
func TestConfigCompiler_CompileConfigPack(t *testing.T) {
	logger := zap.NewNop()
	registry := NewExtensionRegistry(logger)
	validator := NewValidator(logger)
	compiler := NewConfigCompiler(registry, validator, logger)
	ctx := context.Background()

	// 注册扩展点契约
	require.NoError(t, registry.RegisterContract(ExtensionPointContract{ExtPointID: ExtPointTerrains}))
	require.NoError(t, registry.RegisterContract(ExtensionPointContract{ExtPointID: ExtPointMapVisionRules}))
	require.NoError(t, registry.RegisterContract(ExtensionPointContract{ExtPointID: ExtPointDamageFormulas}))
	require.NoError(t, registry.RegisterContract(ExtensionPointContract{ExtPointID: ExtPointUnitTypes}))

	configPack := map[string]any{
		"game":      map[string]any{"vision": "quad"},
		"units":     map[string]any{"unit1": "infantry"},
		"buildings": map[string]any{"b1": "castle"},
		"resources": map[string]any{"gold": 1},
		"terrains":  map[string]any{"plain": "grass"},
		"formulas":  map[string]any{"physical": "atk-def"},
	}
	err := compiler.CompileConfigPack(ctx, configPack, 1)
	require.NoError(t, err)
}

// TestConfigFileToExtPoints 测试配置文件到扩展点映射。
func TestConfigFileToExtPoints(t *testing.T) {
	tests := []struct {
		fileName string
		expected int
	}{
		{"terrains", 1},
		{"units", 1},
		{"combat", 4},
		{"economy", 6},
		{"season", 6},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			extPoints := configFileToExtPoints(tt.fileName)
			assert.Len(t, extPoints, tt.expected)
		})
	}
}
