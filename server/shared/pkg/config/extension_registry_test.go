// Package config 共享内核配置模块，提供配置加载/校验/查询统一API。
// 本文件定义ExtensionRegistry的单元测试，测试注册/查询/契约校验/原子替换。
package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestExtensionRegistry_RegisterAndQuery 测试扩展点注册与查询。
func TestExtensionRegistry_RegisterAndQuery(t *testing.T) {
	logger := zap.NewNop()
	registry := NewExtensionRegistry(logger)
	ctx := context.Background()

	// 先注册契约
	err := registry.RegisterContract(ExtensionPointContract{
		ExtPointID:    ExtPointCombatSkills,
		InputContract: `{"type":"object"}`,
	})
	require.NoError(t, err)

	// 注册配置
	err = registry.Register(ctx, ExtPointCombatSkills, map[string]any{"skill1": "fireball"})
	require.NoError(t, err)

	// 查询配置
	cfg, err := registry.Query(ctx, ExtPointCombatSkills)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

// TestExtensionRegistry_Query_NotFound 测试查询未注册的扩展点。
func TestExtensionRegistry_Query_NotFound(t *testing.T) {
	logger := zap.NewNop()
	registry := NewExtensionRegistry(logger)
	ctx := context.Background()

	_, err := registry.Query(ctx, ExtPointCombatSkills)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrExtensionPointNotFound)
}

// TestExtensionRegistry_Register_NoContract 测试注册未注册契约的扩展点。
func TestExtensionRegistry_Register_NoContract(t *testing.T) {
	logger := zap.NewNop()
	registry := NewExtensionRegistry(logger)
	ctx := context.Background()

	err := registry.Register(ctx, ExtPointCombatSkills, map[string]any{"skill1": "fireball"})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrExtensionPointNotFound)
}

// TestExtensionRegistry_RegisterContract_EmptyID 测试注册空扩展点ID的契约。
func TestExtensionRegistry_RegisterContract_EmptyID(t *testing.T) {
	logger := zap.NewNop()
	registry := NewExtensionRegistry(logger)

	err := registry.RegisterContract(ExtensionPointContract{ExtPointID: ""})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrConfigInvalid)
}

// TestExtensionRegistry_HasContract 测试扩展点契约存在性检查。
func TestExtensionRegistry_HasContract(t *testing.T) {
	logger := zap.NewNop()
	registry := NewExtensionRegistry(logger)

	assert.False(t, registry.HasContract(ExtPointCombatSkills))

	err := registry.RegisterContract(ExtensionPointContract{ExtPointID: ExtPointCombatSkills})
	require.NoError(t, err)

	assert.True(t, registry.HasContract(ExtPointCombatSkills))
}

// TestExtensionRegistry_AllExtensionPointIDs 测试获取所有扩展点ID。
func TestExtensionRegistry_AllExtensionPointIDs(t *testing.T) {
	logger := zap.NewNop()
	registry := NewExtensionRegistry(logger)

	err := registry.RegisterContract(ExtensionPointContract{ExtPointID: ExtPointCombatSkills})
	require.NoError(t, err)
	err = registry.RegisterContract(ExtensionPointContract{ExtPointID: ExtPointTerrains})
	require.NoError(t, err)

	ids := registry.AllExtensionPointIDs()
	assert.Len(t, ids, 2)
}

// TestExtensionRegistry_AtomicReplace 测试热更时原子替换配置。
func TestExtensionRegistry_AtomicReplace(t *testing.T) {
	logger := zap.NewNop()
	registry := NewExtensionRegistry(logger)
	ctx := context.Background()

	// 注册契约
	err := registry.RegisterContract(ExtensionPointContract{ExtPointID: ExtPointCombatSkills})
	require.NoError(t, err)

	// 注册初始配置
	oldCfg := map[string]any{"version": 1}
	err = registry.Register(ctx, ExtPointCombatSkills, oldCfg)
	require.NoError(t, err)

	// 原子替换为新配置
	newCfg := map[string]any{"version": 2}
	err = registry.Register(ctx, ExtPointCombatSkills, newCfg)
	require.NoError(t, err)

	// 查询应返回新配置
	cfg, err := registry.Query(ctx, ExtPointCombatSkills)
	require.NoError(t, err)
	result := cfg.(map[string]any)
	assert.Equal(t, 2, result["version"])
}

// TestErrMsg 测试错误码到中文消息的映射。
func TestErrMsg(t *testing.T) {
	assert.Equal(t, "扩展点未注册", ErrMsg(ErrCodeExtensionPointNotFound))
	assert.Equal(t, "配置内容非法", ErrMsg(ErrCodeConfigInvalid))
	assert.Equal(t, "未知错误", ErrMsg(99999))
}

// TestConfigError_Error 测试ConfigError的Error方法。
func TestConfigError_Error(t *testing.T) {
	err := &ConfigError{Code: ErrCodeExtensionPointNotFound, Msg: "扩展点未注册"}
	assert.Contains(t, err.Error(), "14001")
	assert.Contains(t, err.Error(), "扩展点未注册")
}
