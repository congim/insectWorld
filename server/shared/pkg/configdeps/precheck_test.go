// Package configdeps 共享内核配置依赖扫描域。
// 本文件定义热更删除类两阶段预检的单元测试（ADR-004 3.3）：有引用阻塞/force放行标记/无引用放行/超时保守。
package configdeps

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeScanner 测试用ConfigDependencyScanner实现，按注入结果返回引用或错误。
// blockUntilCtxDone为true时阻塞直到ctx截止（模拟扫描超时，验证保守阻塞语义）。
type fakeScanner struct {
	combatRefs        []InstanceRef // 战斗引用结果
	movementRefs      []InstanceRef // 行军引用结果
	productionRefs    []InstanceRef // 生产队列引用结果
	err               error         // 扫描错误（非超时），注入后所有扫描方法返回该错误
	blockUntilCtxDone bool          // 是否阻塞直到ctx截止（超时场景）
}

// ScanCombatRefs 实现ConfigDependencyScanner接口。
func (f *fakeScanner) ScanCombatRefs(ctx context.Context, deletedIDs []string) ([]InstanceRef, error) {
	if f.blockUntilCtxDone {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.combatRefs, nil
}

// ScanMovementRefs 实现ConfigDependencyScanner接口。
func (f *fakeScanner) ScanMovementRefs(ctx context.Context, deletedIDs []string) ([]InstanceRef, error) {
	if f.blockUntilCtxDone {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.movementRefs, nil
}

// ScanProductionRefs 实现ConfigDependencyScanner接口。
func (f *fakeScanner) ScanProductionRefs(ctx context.Context, deletedIDs []string) ([]InstanceRef, error) {
	if f.blockUntilCtxDone {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.productionRefs, nil
}

// testRef 构造测试用引用记录，instanceType为1=战斗 2=行军 3=生产队列。
func testRef(instanceType int, instanceID int64, refKey string) InstanceRef {
	return InstanceRef{
		InstanceType:  instanceType,
		InstanceID:    instanceID,
		RefExtPoint:   "entity.entity_types",
		RefKey:        refKey,
		ConfigVersion: 10,
	}
}

// TestPrecheckDeleteRefs_Allow 无引用放行（commit）：decision=ReloadAllow，不阻塞。
func TestPrecheckDeleteRefs_Allow(t *testing.T) {
	scanners := []ConfigDependencyScanner{&fakeScanner{}}
	result, err := PrecheckDeleteRefs(context.Background(), scanners, []string{"mantis"}, false, "")
	require.NoError(t, err)
	assert.False(t, result.Blocked)
	assert.Empty(t, result.Conflicts)
	assert.Empty(t, result.ForcedInstances)
	assert.False(t, result.BlockedByTimeout)
	assert.Equal(t, ReloadAllow, result.Decision())
}

// TestPrecheckDeleteRefs_Block 有引用且未强制：abort阻塞，返回冲突清单。
func TestPrecheckDeleteRefs_Block(t *testing.T) {
	refs := []InstanceRef{testRef(InstanceTypeCombat, 1001, "mantis")}
	scanners := []ConfigDependencyScanner{&fakeScanner{combatRefs: refs}}
	result, err := PrecheckDeleteRefs(context.Background(), scanners, []string{"mantis"}, false, "")
	require.NoError(t, err)
	assert.True(t, result.Blocked)
	assert.Equal(t, ReloadBlock, result.Decision())
	assert.Len(t, result.Conflicts, 1)
	assert.Equal(t, int64(1001), result.Conflicts[0].InstanceID)
	assert.Equal(t, "mantis", result.Conflicts[0].RefKey)
	assert.Empty(t, result.ForcedInstances)
}

// TestPrecheckDeleteRefs_Force 有引用且运营强制：放行但返回强制实例清单（标记待熔断）与操作人。
func TestPrecheckDeleteRefs_Force(t *testing.T) {
	refs := []InstanceRef{testRef(InstanceTypeCombat, 1001, "mantis")}
	scanners := []ConfigDependencyScanner{&fakeScanner{combatRefs: refs}}
	result, err := PrecheckDeleteRefs(context.Background(), scanners, []string{"mantis"}, true, "op_force_01")
	require.NoError(t, err)
	assert.False(t, result.Blocked)
	assert.Equal(t, ReloadForce, result.Decision())
	assert.Len(t, result.ForcedInstances, 1)
	assert.Len(t, result.Conflicts, 1)
	assert.Equal(t, "op_force_01", result.ForceBy)
	assert.Equal(t, []string{"mantis"}, result.DeletedIDs)
}

// TestPrecheckDeleteRefs_NoDeletedIDs 无删除项：直接放行，不调用任何Scanner。
// 用阻塞型Scanner验证：空deletedIDs应立即返回ReloadAllow，而非进入扫描等待超时。
func TestPrecheckDeleteRefs_NoDeletedIDs(t *testing.T) {
	blocker := &fakeScanner{blockUntilCtxDone: true}
	result, err := PrecheckDeleteRefs(context.Background(), []ConfigDependencyScanner{blocker}, nil, false, "")
	require.NoError(t, err)
	assert.Equal(t, ReloadAllow, result.Decision())
	assert.Empty(t, result.Conflicts)
}

// TestPrecheckDeleteRefs_TimeoutConservative 扫描超时按存在引用保守阻塞（fail-closed，ADR-004 3.3）。
func TestPrecheckDeleteRefs_TimeoutConservative(t *testing.T) {
	blocker := &fakeScanner{blockUntilCtxDone: true}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := PrecheckDeleteRefs(ctx, []ConfigDependencyScanner{blocker}, []string{"mantis"}, false, "")
	require.NoError(t, err)
	assert.True(t, result.Blocked)
	assert.True(t, result.BlockedByTimeout)
	assert.Equal(t, ReloadBlock, result.Decision())
	assert.Empty(t, result.Conflicts) // 超时未完成扫描，无冲突明细，但按存在引用阻塞
}

// TestPrecheckDeleteRefs_ScannerError 非超时扫描错误：返回错误（fail-closed），不静默放行。
func TestPrecheckDeleteRefs_ScannerError(t *testing.T) {
	scanners := []ConfigDependencyScanner{&fakeScanner{err: errors.New("扫描器内部故障")}}
	_, err := PrecheckDeleteRefs(context.Background(), scanners, []string{"mantis"}, false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "扫描战斗实例引用失败")
}

// TestPrecheckDeleteRefs_AggregateAllScanners 汇聚多Scanner多类型引用：战斗+行军+生产引用全部进入冲突清单。
func TestPrecheckDeleteRefs_AggregateAllScanners(t *testing.T) {
	scanners := []ConfigDependencyScanner{
		&fakeScanner{
			combatRefs:     []InstanceRef{testRef(InstanceTypeCombat, 1001, "mantis")},
			movementRefs:   []InstanceRef{testRef(InstanceTypeMovement, 2001, "mantis")},
			productionRefs: []InstanceRef{testRef(InstanceTypeProduction, 3001, "mantis")},
		},
	}
	result, err := PrecheckDeleteRefs(context.Background(), scanners, []string{"mantis"}, false, "")
	require.NoError(t, err)
	assert.True(t, result.Blocked)
	assert.Len(t, result.Conflicts, 3)
}

// TestDiffConfigIDs 对比新旧配置ID集合，区分新增与删除（ADR-004 3.3变更类型分派）。
func TestDiffConfigIDs(t *testing.T) {
	oldIDs := []string{"ant", "mantis", "beetle"}
	newIDs := []string{"ant", "beetle", "spider"}

	added, deleted := DiffConfigIDs(oldIDs, newIDs)
	assert.ElementsMatch(t, []string{"spider"}, added)
	assert.ElementsMatch(t, []string{"mantis"}, deleted)
}

// TestClassifyConfigChange 单个配置项变更类型判定（新增/修改/删除/非法输入）。
func TestClassifyConfigChange(t *testing.T) {
	assert.Equal(t, ChangeTypeAdd, ClassifyConfigChange(false, true))
	assert.Equal(t, ChangeTypeModify, ClassifyConfigChange(true, true))
	assert.Equal(t, ChangeTypeDelete, ClassifyConfigChange(true, false))
	assert.Equal(t, ChangeTypeNone, ClassifyConfigChange(false, false))
}
