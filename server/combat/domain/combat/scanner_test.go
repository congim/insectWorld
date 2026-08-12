// Package combat 战斗聚合根。
// 本文件定义战斗实例配置引用扫描器的单元测试（ADR-004 3.3）：发现引用/无引用/加载错误/快照版本冻结。
package combat

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"insectworld/server/shared/pkg/config"
	"insectworld/server/shared/pkg/configdeps"
)

// buildTestCombat 构造带完整快照引用的进行中战斗：
// 战斗类型1、公式dmg_001、战利品loot_001、技能skill_001、攻击方兵种7、防守方兵种8、阵型3，快照版本10。
func buildTestCombat(combatID int64, configVersion int64) *Combat {
	c := NewCombat(combatID, 1, 10, []int64{101, 102}, []int64{201}, configVersion, 1000)
	c.BindSnapshot("dmg_001", "loot_001", []string{"skill_001"},
		map[int64]PropEntry{
			101: NewPropEntry(101, 100, 50, 1000, 7, map[string]struct{}{"tag_ant": {}}),
		},
		map[int64]PropEntry{
			201: NewPropEntry(201, 80, 40, 800, 8, map[string]struct{}{}),
		},
	)
	c.SetFormation(3)
	return c
}

// TestScanCombatRefs_FindsUnitTypeRef 扫描发现兵种引用：删除兵种7 → 命中攻击方兵种引用。
func TestScanCombatRefs_FindsUnitTypeRef(t *testing.T) {
	scanner := NewCombatRefScanner(func(ctx context.Context) ([]*Combat, error) {
		return []*Combat{buildTestCombat(1001, 10)}, nil
	})

	refs, err := scanner.ScanCombatRefs(context.Background(), []string{"7"})
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, configdeps.InstanceTypeCombat, refs[0].InstanceType)
	assert.Equal(t, int64(1001), refs[0].InstanceID)
	assert.Equal(t, config.ExtPointUnitTypes, refs[0].RefExtPoint)
	assert.Equal(t, "7", refs[0].RefKey)
	assert.Equal(t, int64(10), refs[0].ConfigVersion)
}

// TestScanCombatRefs_MultipleRefs 全引用域扫描：删除战斗类型/公式/战利品/技能/兵种/阵型全部命中。
func TestScanCombatRefs_MultipleRefs(t *testing.T) {
	scanner := NewCombatRefScanner(func(ctx context.Context) ([]*Combat, error) {
		return []*Combat{buildTestCombat(1001, 10)}, nil
	})

	deletedIDs := []string{"1", "dmg_001", "loot_001", "skill_001", "7", "8", "3"}
	refs, err := scanner.ScanCombatRefs(context.Background(), deletedIDs)
	require.NoError(t, err)
	require.Len(t, refs, 7)

	// 引用域覆盖战斗类型/公式/战利品/技能/攻守双方兵种/阵型，且全部绑定快照版本10
	extPoints := make(map[string]int)
	for _, ref := range refs {
		extPoints[ref.RefExtPoint]++
		assert.Equal(t, int64(1001), ref.InstanceID)
		assert.Equal(t, int64(10), ref.ConfigVersion)
	}
	assert.Equal(t, 1, extPoints[config.ExtPointCombatTypes])
	assert.Equal(t, 1, extPoints[config.ExtPointDamageFormulas])
	assert.Equal(t, 1, extPoints[config.ExtPointCombatLootRules])
	assert.Equal(t, 1, extPoints[config.ExtPointCombatSkills])
	assert.Equal(t, 2, extPoints[config.ExtPointUnitTypes]) // 攻击方7 + 防守方8
	assert.Equal(t, 1, extPoints[config.ExtPointCombatFormationEffects])
}

// TestScanCombatRefs_NoRefs 无引用：删除未引用的配置项返回空切片（放行）。
func TestScanCombatRefs_NoRefs(t *testing.T) {
	scanner := NewCombatRefScanner(func(ctx context.Context) ([]*Combat, error) {
		return []*Combat{buildTestCombat(1001, 10)}, nil
	})

	refs, err := scanner.ScanCombatRefs(context.Background(), []string{"999"})
	require.NoError(t, err)
	assert.Empty(t, refs)
}

// TestScanCombatRefs_NoInProgress 无进行中战斗：返回空切片（放行）。
func TestScanCombatRefs_NoInProgress(t *testing.T) {
	scanner := NewCombatRefScanner(func(ctx context.Context) ([]*Combat, error) {
		return []*Combat{}, nil
	})

	refs, err := scanner.ScanCombatRefs(context.Background(), []string{"7"})
	require.NoError(t, err)
	assert.Empty(t, refs)
}

// TestScanCombatRefs_EmptyDeletedIDs 空删除清单：不调用加载函数直接返回空（新增/修改类无需扫描）。
func TestScanCombatRefs_EmptyDeletedIDs(t *testing.T) {
	loadCalled := false
	scanner := NewCombatRefScanner(func(ctx context.Context) ([]*Combat, error) {
		loadCalled = true
		return nil, nil
	})

	refs, err := scanner.ScanCombatRefs(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, refs)
	assert.False(t, loadCalled)
}

// TestScanCombatRefs_LoadError 进行中战斗加载失败：返回错误（fail-closed，不允许静默放行）。
func TestScanCombatRefs_LoadError(t *testing.T) {
	scanner := NewCombatRefScanner(func(ctx context.Context) ([]*Combat, error) {
		return nil, errors.New("仓储故障")
	})

	_, err := scanner.ScanCombatRefs(context.Background(), []string{"7"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "加载进行中战斗失败")
}

// TestScanCombatRefs_MovementProductionEmpty 行军/生产队列扫描：战斗服务不持有对应实例，返回空。
func TestScanCombatRefs_MovementProductionEmpty(t *testing.T) {
	scanner := NewCombatRefScanner(func(ctx context.Context) ([]*Combat, error) {
		return []*Combat{buildTestCombat(1001, 10)}, nil
	})

	movementRefs, err := scanner.ScanMovementRefs(context.Background(), []string{"7"})
	require.NoError(t, err)
	assert.Empty(t, movementRefs)

	productionRefs, err := scanner.ScanProductionRefs(context.Background(), []string{"7"})
	require.NoError(t, err)
	assert.Empty(t, productionRefs)
}

// TestScanCombatRefs_SnapshotFrozenVersion 快照版本冻结（ADR-004 3.1/3.4版本归属）：
// 引用记录携带开战冻结的ConfigVersion，热更/回滚不改变进行中战斗的配置基准。
func TestScanCombatRefs_SnapshotFrozenVersion(t *testing.T) {
	// 战斗在版本10开战（快照冻结），随后热更到版本11——扫描引用仍以快照版本10为基准
	scanner := NewCombatRefScanner(func(ctx context.Context) ([]*Combat, error) {
		return []*Combat{buildTestCombat(1001, 10)}, nil
	})

	refs, err := scanner.ScanCombatRefs(context.Background(), []string{"dmg_001"})
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, int64(10), refs[0].ConfigVersion)
	assert.NotEqual(t, int64(11), refs[0].ConfigVersion)
}
