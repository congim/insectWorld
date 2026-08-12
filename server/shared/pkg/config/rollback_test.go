// Package config 共享内核配置模块。
// 本文件定义双向回滚命令的单元测试（ADR-004 3.4）：回滚后N-1生效/N保留至引用结束/校验失败保持N，
// 以及版本归属三类语义（快照类保持开始版本、即时类跟随当前版本、新建业务用回滚后版本）。
package config

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRollbackStore 构造版本9(N-1)与10(N)的版本化存储，用于回滚测试。
// 版本10新增兵种"beetle"（版本9没有），模拟N=10热更引入新配置后需回退到N-1=9的场景。
func setupRollbackStore() *VersionedConfigStore {
	store := NewVersionedConfigStore()
	store.PutEntry(9, ExtPointDamageFormulas, "dmg_001", "atk*2")
	store.PutEntry(9, ExtPointUnitTypes, "7", "ant")
	store.PutEntry(10, ExtPointDamageFormulas, "dmg_001", "atk*3")
	store.PutEntry(10, ExtPointUnitTypes, "7", "mantis")
	store.PutEntry(10, ExtPointUnitTypes, "8", "beetle")
	return store
}

// TestRollbackConfig_ToOlderVersion 回滚N→N-1后N-1生效，N保留（回滚本身不删除任何版本）。
func TestRollbackConfig_ToOlderVersion(t *testing.T) {
	store := setupRollbackStore()
	rb := NewConfigRollbacker(store, 10)

	result, err := rb.RollbackConfig(context.Background(), RollbackRequest{FromVersion: 10, ToVersion: 9, Operator: "op_test"})
	require.NoError(t, err)
	assert.Equal(t, int64(9), result.ToVersion)
	assert.Equal(t, int64(9), result.CurrentVersion)
	assert.Equal(t, int64(9), rb.CurrentVersion())

	// 回滚后N-1生效：当前版本配置查询取版本9
	dmg, err := store.Get(9, ExtPointDamageFormulas, "dmg_001")
	require.NoError(t, err)
	assert.Equal(t, "atk*2", dmg)

	// N保留：版本10仍可查询（引用计数保护，回滚不物理删除）
	beetle, err := store.Get(10, ExtPointUnitTypes, "8")
	require.NoError(t, err)
	assert.Equal(t, "beetle", beetle)
}

// TestRollbackConfig_ValidationFailure_KeepsN 校验失败的回滚也回滚（保持N）：
// 目标版本不可用时不切换当前版本，仅返回错误（ADR-004 3.4回滚原子性）。
func TestRollbackConfig_ValidationFailure_KeepsN(t *testing.T) {
	store := setupRollbackStore()
	rb := NewConfigRollbacker(store, 10)

	// 目标版本8从未编译写入存储 → 校验失败，保持当前版本N=10
	_, err := rb.RollbackConfig(context.Background(), RollbackRequest{FromVersion: 10, ToVersion: 8, Operator: "op_test"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrConfigVersionGone))
	assert.Equal(t, int64(10), rb.CurrentVersion())

	// 参数非法：目标版本与当前版本相同
	_, err = rb.RollbackConfig(context.Background(), RollbackRequest{FromVersion: 10, ToVersion: 10, Operator: "op_test"})
	require.Error(t, err)
	assert.Equal(t, int64(10), rb.CurrentVersion())

	// 参数非法：目标版本晚于当前版本（非回滚方向）
	_, err = rb.RollbackConfig(context.Background(), RollbackRequest{FromVersion: 9, ToVersion: 10, Operator: "op_test"})
	require.Error(t, err)
	assert.Equal(t, int64(10), rb.CurrentVersion())

	// 当前版本与请求不一致（并发热更/回滚撕裂防护）
	_, err = rb.RollbackConfig(context.Background(), RollbackRequest{FromVersion: 11, ToVersion: 9, Operator: "op_test"})
	require.Error(t, err)
	assert.Equal(t, int64(10), rb.CurrentVersion())

	// 校验失败不产生审计日志（无实际回滚发生）
	assert.Empty(t, rb.AuditLog())
}

// TestRollbackConfig_NPreservedUntilRefEnd N保留至引用结束（ADR-004 3.4关键配套）：
// 回滚后引用N的战斗未结束时N不清；引用结束（Unpin归零）后回滚命令仍不删除N，
// N仅在历史超上限且未引用时由PruneUnreferenced清理（与versioned_store_test的引用保护测试同构）。
func TestRollbackConfig_NPreservedUntilRefEnd(t *testing.T) {
	store := NewVersionedConfigStore()
	for v := int64(1); v <= 12; v++ {
		store.PutEntry(v, ExtPointDamageFormulas, "dmg_001", "atk*2")
	}
	// 快照类业务（战斗/生产队列）引用当前版本N=12（Pin），模拟进行中实例引用将被回滚的版本
	require.NoError(t, store.Pin(12))
	rb := NewConfigRollbacker(store, 12)

	result, err := rb.RollbackConfig(context.Background(), RollbackRequest{FromVersion: 12, ToVersion: 11, Operator: "op_test"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.AffectedInstances)

	// 引用未结束时版本12仍在（ADR-004 3.4禁止在存在引用时物理删除配置版本）
	_, err = store.Get(12, ExtPointDamageFormulas, "dmg_001")
	require.NoError(t, err)

	// 引用结束（Unpin归零）后，回滚命令本身仍不删除N；仅PruneUnreferenced按历史上限清理最旧未引用版本
	require.NoError(t, store.Unpin(12))
	_, err = store.Get(12, ExtPointDamageFormulas, "dmg_001")
	require.NoError(t, err)
	// 12个版本全部未引用（版本12曾引用现已归零），超出上限2个 → 清理最老的未引用版本1、2；
	// 版本12不在清理之列，证明N保留至引用结束、回滚命令从不主动删除N（仅历史滚动清理最旧版本）
	removed := store.PruneUnreferenced()
	assert.ElementsMatch(t, []int64{1, 2}, removed)
	_, err = store.Get(12, ExtPointDamageFormulas, "dmg_001")
	require.NoError(t, err)
}

// TestRollbackConfig_VersionAttribution 版本归属三类语义（ADR-004 3.4，以测试表达业务约定）：
// 快照类保持开始版本、即时类跟随当前版本、新建业务用回滚后版本。
func TestRollbackConfig_VersionAttribution(t *testing.T) {
	store := setupRollbackStore()
	rb := NewConfigRollbacker(store, 10)

	// 快照类（战斗/生产队列）：开战时Pin N=10，保持开始版本
	require.NoError(t, store.Pin(10))
	// 即时类（移动/外交）：不Pin，跟随当前版本（无版本化查询）
	// 新建业务：用回滚后版本N-1（由currentVersion决定）

	_, err := rb.RollbackConfig(context.Background(), RollbackRequest{FromVersion: 10, ToVersion: 9, Operator: "op_rollback"})
	require.NoError(t, err)

	// 快照类：进行中战斗继续用N=10快照（版本化查询GetWithVersion），回滚不追溯改变（ADR-004 3.4）
	// 战斗B1在版本10开战，快照冻结FormulaID=dmg_001 → 版本10的公式"atk*3"仍可查询，不因回滚改变
	snapshotDmg, err := store.Get(10, ExtPointDamageFormulas, "dmg_001")
	require.NoError(t, err)
	assert.Equal(t, "atk*3", snapshotDmg)

	// 即时类/新建业务：跟随回滚后当前版本N-1=9（下个Tick/下次查询自然用版本9）
	assert.Equal(t, int64(9), rb.CurrentVersion())
	currentDmg, err := store.Get(9, ExtPointDamageFormulas, "dmg_001")
	require.NoError(t, err)
	assert.Equal(t, "atk*2", currentDmg)
}

// TestRollbackConfig_AuditLog 回滚审计日志：记录操作人/从版本/到版本/受影响实例数（ADR-004 3.4审计）。
func TestRollbackConfig_AuditLog(t *testing.T) {
	store := setupRollbackStore()
	require.NoError(t, store.Pin(10)) // 1个进行中实例引用N=10
	rb := NewConfigRollbacker(store, 10)

	_, err := rb.RollbackConfig(context.Background(), RollbackRequest{FromVersion: 10, ToVersion: 9, Operator: "op_audit"})
	require.NoError(t, err)

	log := rb.AuditLog()
	require.Len(t, log, 1)
	assert.Equal(t, "op_audit", log[0].Operator)
	assert.Equal(t, int64(10), log[0].FromVersion)
	assert.Equal(t, int64(9), log[0].ToVersion)
	assert.Equal(t, int64(1), log[0].AffectedInstances)
	assert.Greater(t, log[0].TimestampMs, int64(0))

	// 再次回滚（9→8目标不存在，失败）不追加审计
	_, err = rb.RollbackConfig(context.Background(), RollbackRequest{FromVersion: 9, ToVersion: 8, Operator: "op_audit"})
	require.Error(t, err)
	assert.Len(t, rb.AuditLog(), 1)
}
