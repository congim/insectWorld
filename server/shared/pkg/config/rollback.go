// Package config 共享内核配置模块，提供配置加载/校验/查询统一API。
// 本文件实现双向回滚命令本体（ADR-004 3.4）：从N回退N-1，复用VersionedConfigStore版本保留机制。
// E4验证报告缺口#2：Pin/Unpin引用计数已落地，回滚操作本体（命令/审计/版本归属切换）在本文件补齐。
package config

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// 版本归属规则（ADR-004 3.4双向回滚，以代码注释表达语义，业务接入时按此执行，本文件不强制改业务代码）：
//  1. 快照类（战斗/生产队列）：保持开始版本——进行中实例继续用N快照（版本化查询GetWithVersion），
//     不因回滚追溯改变（战斗公平性与回放一致性由快照冻结保证）；
//  2. 即时类（移动/外交）：跟随当前版本——回滚后下个Tick/下次查询自然用N-1（spec.md规则9）；
//  3. 新建业务：用回滚后版本N-1——RollbackConfig切换currentVersion后，新实例Pin N-1。
//
// 版本保留：回滚只切换当前版本指针，不物理删除任何版本；N保留由引用计数保护（Pin/Unpin）——
// 存在进行中实例引用N（PinCount>0）时N不清，待最后引用结束（Unpin归零）且超出历史上限后
// 由PruneUnreferenced清理（ADR-004 3.4禁止在存在引用时物理删除配置版本，否则与场景B同构断裂）。

// RollbackRequest 配置回滚请求参数（ADR-004 3.4）。
type RollbackRequest struct {
	FromVersion int64  // 从版本（当前版本N，回滚前生效版本）
	ToVersion   int64  // 到版本（回退目标N-1，回滚后生效版本，必须早于FromVersion）
	Operator    string // 操作人（运营后台账号），审计日志记录（规范7审计）
}

// RollbackResult 配置回滚结果（ADR-004 3.4）。
type RollbackResult struct {
	FromVersion       int64 // 从版本N（回滚前生效版本）
	ToVersion         int64 // 到版本N-1（回滚后生效版本）
	AffectedInstances int64 // 受影响进行中实例数（=引用N的实例数PinCount；回滚不追溯改变其版本归属）
	CurrentVersion    int64 // 回滚后的当前版本（=ToVersion，新建业务/即时类从此版本生效）
}

// RollbackAuditEntry 回滚审计日志条目（ADR-004 3.4审计：操作人/从版本/到版本/受影响实例数）。
type RollbackAuditEntry struct {
	Operator          string // 操作人（运营后台账号）
	FromVersion       int64  // 从版本N
	ToVersion         int64  // 到版本N-1
	AffectedInstances int64  // 受影响进行中实例数（引用N的实例数）
	TimestampMs       int64  // 操作时间戳（毫秒，AGENTS.md规范8）
}

// ConfigRollbacker 配置回滚执行器（ADR-004 3.4双向回滚命令本体）。
// 维护当前版本指针与回滚审计日志；复用VersionedConfigStore——回滚只切换当前版本指针，
// 版本N保留由引用计数保护（Pin/Unpin），回滚命令本身不物理删除任何版本。
// 回滚与热更同路径：目标版本须为已编译写入存储的版本（等价于回滚前重新编译校验，ADR-004 3.4原子性）。
type ConfigRollbacker struct {
	store          *VersionedConfigStore // 版本化配置存储（复用，ADR-004 3.1版本保留机制）
	mu             sync.RWMutex          // 读写锁，保护当前版本与审计日志并发安全
	currentVersion int64                 // 当前配置版本号，回滚后指向ToVersion
	audit          []RollbackAuditEntry  // 回滚审计日志（内存态；持久化由Config Service审计仓储负责，AGENTS.md规范7.8）
}

// NewConfigRollbacker 创建配置回滚执行器实例。
// store为版本化配置存储，currentVersion为初始当前版本号。
func NewConfigRollbacker(store *VersionedConfigStore, currentVersion int64) *ConfigRollbacker {
	return &ConfigRollbacker{
		store:          store,
		currentVersion: currentVersion,
		audit:          make([]RollbackAuditEntry, 0),
	}
}

// CurrentVersion 返回当前配置版本号。
func (r *ConfigRollbacker) CurrentVersion() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.currentVersion
}

// RollbackConfig 执行配置回滚：从N回退N-1（ADR-004 3.4双向回滚命令）。
// 复用VersionedConfigStore：校验ToVersion在存储中存在后，切换当前版本到ToVersion；
// FromVersion（N）保留由引用计数保护——存在进行中实例引用N（PinCount>0）时不清，
// 待最后引用结束（Unpin归零）且超出历史上限后由PruneUnreferenced清理。
// 回滚原子性：任何校验失败（参数非法/当前版本不一致/目标版本不可用）保持当前版本N不变，
// 即"校验失败的回滚也回滚"（ADR-004 3.4）；校验全部通过后才切换当前版本。
// 审计日志记录操作人/从版本/到版本/受影响进行中实例数。
func (r *ConfigRollbacker) RollbackConfig(ctx context.Context, req RollbackRequest) (*RollbackResult, error) {
	_ = ctx // 预留：持久化审计写入时使用；当前内存态操作无阻塞
	if req.FromVersion <= 0 || req.ToVersion <= 0 {
		return nil, fmt.Errorf("配置回滚失败，版本号必须为正数，from=%d, to=%d", req.FromVersion, req.ToVersion)
	}
	if req.FromVersion == req.ToVersion {
		return nil, fmt.Errorf("配置回滚失败，目标版本与当前版本相同，version=%d", req.FromVersion)
	}
	if req.ToVersion > req.FromVersion {
		return nil, fmt.Errorf("配置回滚失败，目标版本必须早于当前版本，from=%d, to=%d", req.FromVersion, req.ToVersion)
	}

	// 原子性：先校验后切换（同路径热更的"编译校验通过后版本化切换"语义，ADR-004 3.4）
	r.mu.Lock()
	defer r.mu.Unlock()
	// FromVersion必须等于当前版本：防并发回滚/热更撕裂（校验失败保持N不变）
	if r.currentVersion != req.FromVersion {
		return nil, fmt.Errorf("配置回滚失败，当前版本与请求不一致，current=%d, from=%d", r.currentVersion, req.FromVersion)
	}
	// ToVersion必须存在于存储：目标版本须为已编译写入的版本（校验失败保持N不变，仅返回错误）
	if !r.store.HasVersion(req.ToVersion) {
		return nil, fmt.Errorf("配置回滚失败，目标版本不可用，to=%d: %w", req.ToVersion, ErrConfigVersionGone)
	}

	// 受影响进行中实例数 = 引用FromVersion(N)的实例数（快照类业务Pin计数）；
	// 回滚不追溯改变其版本归属（ADR-004 3.4快照类保持开始版本），仅记录供运营审计。
	affected := r.store.PinCount(req.FromVersion)

	// 校验全部通过后切换当前版本（原子提交）
	r.currentVersion = req.ToVersion
	r.audit = append(r.audit, RollbackAuditEntry{
		Operator:          req.Operator,
		FromVersion:       req.FromVersion,
		ToVersion:         req.ToVersion,
		AffectedInstances: affected,
		TimestampMs:       time.Now().UnixMilli(),
	})

	return &RollbackResult{
		FromVersion:       req.FromVersion,
		ToVersion:         req.ToVersion,
		AffectedInstances: affected,
		CurrentVersion:    r.currentVersion,
	}, nil
}

// AuditLog 返回回滚审计日志副本（不可变视图，运营查询/测试用）。
func (r *ConfigRollbacker) AuditLog() []RollbackAuditEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]RollbackAuditEntry, len(r.audit))
	copy(out, r.audit)
	return out
}
