// Package configdeps 共享内核配置依赖扫描域。
// 本文件实现热更删除类变更的两阶段预检决策纯逻辑（ADR-004 3.3/3.6）：
// prepare阶段汇聚各业务服务Scanner引用 → 有引用且非force→abort阻塞；有引用且force→标记待熔断放行；
// 无引用→commit放行；超时按存在引用保守阻塞。Config Service在写etcd之前调用本函数（application层编排）。
package configdeps

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// 热更预检决策常量（ADR-004 3.6，规范1）。
const (
	// ReloadAllow 预检通过，允许发布（无引用或非删除类变更）。
	ReloadAllow = 1
	// ReloadBlock 预检发现引用且未强制，阻塞发布（默认，保守安全）。
	ReloadBlock = 2
	// ReloadForce 运营强制发布，冲突实例标记"待熔断"，结算/产出时走熔断协议（ADR-004 3.2）。
	ReloadForce = 3
)

// 变更类型常量（ADR-004 3.3变更类型分派）。
const (
	// ChangeTypeNone 非法输入：配置项ID新旧版本均不存在，调用方不应传入。
	ChangeTypeNone = 0
	// ChangeTypeAdd 新增：新配置项ID不在旧版本，无引用断裂风险，直接放行。
	ChangeTypeAdd = 1
	// ChangeTypeModify 修改：ID存在值变化，降级放行——进行中快照类业务走版本化查询继续用旧版本，
	// 即时类业务下个Tick用新值，无需扫描（ADR-004 3.3降级语义）。
	ChangeTypeModify = 2
	// ChangeTypeDelete 删除：旧配置项ID在新版本缺失，需扫描进行中实例引用（本包核心场景）。
	ChangeTypeDelete = 3
)

// DefaultValidationTimeoutSeconds 预检默认超时秒数（config_validation_timeout_seconds，默认10，ADR-004 3.3）。
// 框架基础设施默认值白名单（AGENTS.md规范1.1.5），Config Service可按配置覆盖。
const DefaultValidationTimeoutSeconds = 10

// ReloadPrecheckResult 热更发布预检结果，由各业务服务Scanner汇聚生成（ADR-004 3.3/3.6）。
type ReloadPrecheckResult struct {
	Blocked          bool          // 是否阻塞发布：true=发现引用且未强制（decision=ReloadBlock）；false=放行或强制
	Conflicts        []InstanceRef // 冲突引用清单（阻塞/强制时非空），供运营后台定位实例ID+引用项+开始版本
	ForcedInstances  []InstanceRef // 强制发布时需标记"待熔断"的实例清单（decision=ReloadForce时非空）
	DeletedIDs       []string      // 本次删除的配置项ID，审计用
	ForceBy          string        // 强制发布操作人（decision=ReloadForce时记录，审计用）
	BlockedByTimeout bool          // 是否因扫描超时保守阻塞（超时按存在引用处理，ADR-004 3.3）
}

// Decision 返回预检决策常量（ReloadAllow/ReloadBlock/ReloadForce）。
// 阻塞优先级高于强制：超时/发现引用且未强制→ReloadBlock；强制发布→ReloadForce；否则放行。
func (r *ReloadPrecheckResult) Decision() int {
	switch {
	case r.Blocked:
		return ReloadBlock
	case len(r.ForcedInstances) > 0:
		return ReloadForce
	default:
		return ReloadAllow
	}
}

// PrecheckDeleteRefs 删除类热更发布前两阶段预检（prepare→commit/abort，ADR-004 3.3/3.6）。
// prepare阶段汇聚各业务服务Scanner对deletedIDs的引用，决策如下：
//   - 有引用且非force → abort：返回ReloadBlock（含冲突清单），拒绝发布；
//   - 有引用且force → 标记circuitPending：返回ReloadForce（含强制实例清单，供业务标记"待熔断"）；
//   - 无引用 → commit：返回ReloadAllow放行。
//
// 超时保守处理：按config_validation_timeout_seconds（默认10秒）约束扫描，超时按"存在引用"阻塞，
// 避免漏扫导致删除配置项被进行中实例引用（ADR-004 3.3）；超时场景不返回错误（保守放行风险更高）。
// ctx若已携带更早截止时间则沿用调用方截止时间。
// scanners为各业务服务实现的ConfigDependencyScanner，Config Service通过gRPC汇聚结果后传入；
// 本函数为领域层决策纯逻辑，不依赖gRPC/存储。
func PrecheckDeleteRefs(ctx context.Context, scanners []ConfigDependencyScanner, deletedIDs []string, force bool, operator string) (*ReloadPrecheckResult, error) {
	// 无删除项无需预检（新增/修改类变更无引用断裂风险，ADR-004 3.3变更类型分派）
	if len(deletedIDs) == 0 {
		return &ReloadPrecheckResult{DeletedIDs: deletedIDs}, nil
	}

	// 超时约束：调用方未携带截止时间时按默认超时派生，保证"超时按存在引用保守阻塞"可观测
	scanCtx, cancel := timeoutCtx(ctx)
	defer cancel()

	var conflicts []InstanceRef
	for _, sc := range scanners {
		if sc == nil {
			continue
		}
		combatRefs, err := sc.ScanCombatRefs(scanCtx, deletedIDs)
		if err != nil {
			// 超时按存在引用保守阻塞（不返回错误，避免绕过预检直接发布）
			if isTimeoutErr(scanCtx, err) {
				return &ReloadPrecheckResult{Blocked: true, DeletedIDs: deletedIDs, BlockedByTimeout: true}, nil
			}
			return nil, fmt.Errorf("扫描战斗实例引用失败: %w", err)
		}
		conflicts = append(conflicts, combatRefs...)

		movementRefs, err := sc.ScanMovementRefs(scanCtx, deletedIDs)
		if err != nil {
			if isTimeoutErr(scanCtx, err) {
				return &ReloadPrecheckResult{Blocked: true, DeletedIDs: deletedIDs, BlockedByTimeout: true}, nil
			}
			return nil, fmt.Errorf("扫描行军实例引用失败: %w", err)
		}
		conflicts = append(conflicts, movementRefs...)

		productionRefs, err := sc.ScanProductionRefs(scanCtx, deletedIDs)
		if err != nil {
			if isTimeoutErr(scanCtx, err) {
				return &ReloadPrecheckResult{Blocked: true, DeletedIDs: deletedIDs, BlockedByTimeout: true}, nil
			}
			return nil, fmt.Errorf("扫描生产队列实例引用失败: %w", err)
		}
		conflicts = append(conflicts, productionRefs...)
	}

	if len(conflicts) == 0 {
		return &ReloadPrecheckResult{DeletedIDs: deletedIDs}, nil
	}
	if force {
		// 强制发布：冲突实例标记"待熔断"，结算/产出时走熔断协议，不静默错发（ADR-004 3.3阻塞与降级语义）
		return &ReloadPrecheckResult{
			Conflicts:       conflicts,
			ForcedInstances: conflicts,
			DeletedIDs:      deletedIDs,
			ForceBy:         operator,
		}, nil
	}
	return &ReloadPrecheckResult{Blocked: true, Conflicts: conflicts, DeletedIDs: deletedIDs}, nil
}

// DiffConfigIDs 对比新旧配置ID集合，返回新增/删除的ID清单（ADR-004 3.3变更类型分派）。
// oldIDs为旧版本配置项ID集合，newIDs为新版本配置项ID集合；
// 新增=仅新集合存在（ChangeTypeAdd），删除=仅旧集合存在（ChangeTypeDelete）。
func DiffConfigIDs(oldIDs, newIDs []string) (added, deleted []string) {
	oldSet := make(map[string]struct{}, len(oldIDs))
	for _, id := range oldIDs {
		oldSet[id] = struct{}{}
	}
	newSet := make(map[string]struct{}, len(newIDs))
	for _, id := range newIDs {
		newSet[id] = struct{}{}
	}
	for id := range newSet {
		if _, ok := oldSet[id]; !ok {
			added = append(added, id)
		}
	}
	for id := range oldSet {
		if _, ok := newSet[id]; !ok {
			deleted = append(deleted, id)
		}
	}
	return added, deleted
}

// ClassifyConfigChange 判定单个配置项的变更类型（ADR-004 3.3变更类型分派）。
// oldHas为该ID旧版本是否存在，newHas为新版本是否存在；
// 返回ChangeTypeAdd/Modify/Delete；新旧均不存在为调用方非法输入，返回ChangeTypeNone。
func ClassifyConfigChange(oldHas, newHas bool) int {
	switch {
	case !oldHas && newHas:
		return ChangeTypeAdd
	case oldHas && newHas:
		return ChangeTypeModify
	case oldHas && !newHas:
		return ChangeTypeDelete
	default:
		return ChangeTypeNone
	}
}

// timeoutCtx 派生带超时的扫描上下文：调用方已设置截止时间则沿用（不缩短），否则按默认预检超时。
func timeoutCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, DefaultValidationTimeoutSeconds*time.Second)
}

// isTimeoutErr 判断扫描错误是否因超时/取消导致（调用方主动取消同样按存在引用保守阻塞，fail-closed）。
func isTimeoutErr(ctx context.Context, err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	return ctx.Err() != nil
}
