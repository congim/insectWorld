// Package config 共享内核配置模块，提供配置加载/校验/查询统一API。
// 本文件定义VersionedConfigStore版本化配置存储的单元测试（ADR-004 3.1/3.4）。
package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionedConfigStore_GetPut 测试配置项写入与查询。
func TestVersionedConfigStore_GetPut(t *testing.T) {
	store := NewVersionedConfigStore()

	store.PutEntry(10, ExtPointDamageFormulas, "dmg_001", "atk*2")
	store.PutEntry(10, ExtPointDamageFormulas, "dmg_002", "atk*3")

	val, err := store.Get(10, ExtPointDamageFormulas, "dmg_001")
	require.NoError(t, err)
	assert.Equal(t, "atk*2", val)

	// 配置项缺失返回nil不报错（存在性用Has判断）
	val, err = store.Get(10, ExtPointDamageFormulas, "dmg_999")
	require.NoError(t, err)
	assert.Nil(t, val)
}

// TestVersionedConfigStore_GetVersionGone 测试版本不可用返回ErrConfigVersionGone。
func TestVersionedConfigStore_GetVersionGone(t *testing.T) {
	store := NewVersionedConfigStore()
	store.PutEntry(10, ExtPointDamageFormulas, "dmg_001", "atk*2")

	// 未记录的版本（旧版本已被清理或从未记录）→ ErrConfigVersionGone
	_, err := store.Get(9, ExtPointDamageFormulas, "dmg_001")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrConfigVersionGone))

	_, err = store.Get(0, ExtPointDamageFormulas, "dmg_001")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrConfigVersionGone))
}

// TestVersionedConfigStore_Has 测试配置项存在性判断。
func TestVersionedConfigStore_Has(t *testing.T) {
	store := NewVersionedConfigStore()
	store.PutEntry(10, ExtPointCombatLootRules, "loot_001", "gold:100")

	exists, err := store.Has(10, ExtPointCombatLootRules, "loot_001")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = store.Has(10, ExtPointCombatLootRules, "loot_999")
	require.NoError(t, err)
	assert.False(t, exists)

	// 版本不可用返回错误
	_, err = store.Has(8, ExtPointCombatLootRules, "loot_001")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrConfigVersionGone))
}

// TestVersionedConfigStore_PinUnpin 测试版本引用计数锁定与释放。
func TestVersionedConfigStore_PinUnpin(t *testing.T) {
	store := NewVersionedConfigStore()
	store.PutEntry(10, ExtPointDamageFormulas, "dmg_001", "atk*2")

	// 初始引用计数为0
	assert.Equal(t, int64(0), store.PinCount(10))

	// Pin两次（两个进行中战斗引用）
	require.NoError(t, store.Pin(10))
	require.NoError(t, store.Pin(10))
	assert.Equal(t, int64(2), store.PinCount(10))

	// Unpin一次
	require.NoError(t, store.Unpin(10))
	assert.Equal(t, int64(1), store.PinCount(10))

	// 未记录的版本Pin/Unpin报错
	assert.Error(t, store.Pin(99))
	assert.Error(t, store.Unpin(99))
}

// TestVersionedConfigStore_Prune_RefProtected 测试引用计数保护版本不被清理。
// 场景：版本1~12共12个版本，版本10被进行中战斗引用；未引用版本11个超出上限10，
// 最老的未引用版本被清理，被引用的版本10保留（ADR-004 3.4禁止删除引用版本）。
func TestVersionedConfigStore_Prune_RefProtected(t *testing.T) {
	store := NewVersionedConfigStore()
	for v := int64(1); v <= 12; v++ {
		store.PutEntry(v, ExtPointDamageFormulas, "dmg_001", "atk*2")
	}
	// 版本10被进行中实例引用，禁止清理
	require.NoError(t, store.Pin(10))

	removed := store.PruneUnreferenced()
	// 未引用版本1~9/11/12共11个，超出上限1个 → 清理最老的版本1
	assert.Equal(t, []int64{1}, removed)
	assert.Equal(t, 11, store.VersionCount())

	// 版本10仍在（引用保护）
	_, err := store.Get(10, ExtPointDamageFormulas, "dmg_001")
	require.NoError(t, err)
	// 版本1已被清理
	_, err = store.Get(1, ExtPointDamageFormulas, "dmg_001")
	assert.True(t, errors.Is(err, ErrConfigVersionGone))
}

// TestVersionedConfigStore_PutExtPoint 测试整体扩展点退化存储路径。
func TestVersionedConfigStore_PutExtPoint(t *testing.T) {
	store := NewVersionedConfigStore()
	store.PutExtPoint(10, ExtPointCounterMatrix, map[string]any{"mantis": "ant"})

	val, err := store.Get(10, ExtPointCounterMatrix, ExtPointCounterMatrix)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"mantis": "ant"}, val)
}

// TestVersionedConfigStore_HasVersion 测试版本存在性判断（回滚目标校验用，ADR-004 3.4）。
func TestVersionedConfigStore_HasVersion(t *testing.T) {
	store := NewVersionedConfigStore()
	store.PutEntry(10, ExtPointDamageFormulas, "dmg_001", "atk*2")

	assert.True(t, store.HasVersion(10))
	assert.False(t, store.HasVersion(9))
	assert.False(t, store.HasVersion(0))
}
