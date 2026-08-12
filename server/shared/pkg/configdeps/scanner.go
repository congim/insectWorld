// Package configdeps 共享内核配置依赖扫描域，定义热更删除类变更的进行中实例引用扫描契约（ADR-004 3.3）。
// ConfigDependencyScanner接口由各业务服务实现（Combat/Movement/Production），Config Service发布预检时汇聚结果；
// InstanceRef为冲突清单条目；PrecheckDeleteRefs为两阶段预检决策纯逻辑（prepare→commit/abort，见precheck.go）。
// 本包不依赖pkg/config，避免Config Service与业务服务双向引用；扩展点ID以字符串透传，取值见pkg/config扩展点常量。
package configdeps

import "context"

// 实例类型常量（规范1，就近归属扫描域，对应ADR-004 3.3 InstanceRef.instanceType）。
const (
	// InstanceTypeCombat 战斗实例，引用公式/克制/技能/战利品/兵种/阵型等战斗配置。
	InstanceTypeCombat = 1
	// InstanceTypeMovement 行军（移动订单）实例，引用移动类型/消耗表/地形等移动配置。
	InstanceTypeMovement = 2
	// InstanceTypeProduction 生产/采集队列实例，引用产出公式/资源类型等经济配置。
	InstanceTypeProduction = 3
)

// InstanceRef 进行中实例配置引用记录，热更预检冲突清单条目（ADR-004 3.3）。
type InstanceRef struct {
	InstanceType  int    // 实例类型：1=战斗 2=行军 3=生产队列
	InstanceID    int64  // 实例ID（战斗ID/移动订单ID/生产队列ID），雪花算法生成（规范8用int64）
	RefExtPoint   string // 被引用扩展点ID，如config.ExtPointUnitTypes（"entity.entity_types"）
	RefKey        string // 被引用配置项ID，如被删除的兵种ID"mantis"
	ConfigVersion int64  // 实例开始时的配置版本号，快照类业务冻结该版本（ADR-004 3.1），回滚不追溯改变
}

// ConfigDependencyScanner 进行中实例配置引用扫描接口，热更删除类变更发布前调用（ADR-004 3.3）。
// 各业务服务实现，Config Service发布预检时通过gRPC汇聚结果（spec.md缺口11跨服务一致性）：
//   - Combat服务实现ScanCombatRefs：本仓库server/combat/domain/combat.CombatRefScanner已完整落地；
//   - Movement/Production扫描器归属Social/Economy服务，P3六骨头落地后实现（本接口为唯一契约，见各方法TODO）。
//
// 领域层决策纯逻辑见PrecheckDeleteRefs：有引用→阻塞（abort）；运营强制→实例标记"待熔断"（force）。
type ConfigDependencyScanner interface {
	// ScanCombatRefs 扫描进行中战斗对指定配置项ID的引用（公式/克制/技能/战利品/兵种/阵型）。
	// deletedIDs为本次热更删除的配置项ID清单；返回引用记录，无引用返回空切片。
	ScanCombatRefs(ctx context.Context, deletedIDs []string) ([]InstanceRef, error)

	// ScanMovementRefs 扫描进行中行军对指定配置项ID的引用（移动类型/消耗表/地形）。
	// TODO 归属Social服务（行军聚合根），P3移动骨头落地后实现；当前返回空表示无引用。
	ScanMovementRefs(ctx context.Context, deletedIDs []string) ([]InstanceRef, error)

	// ScanProductionRefs 扫描进行中生产/采集队列对指定配置项ID的引用（产出公式/资源类型）。
	// TODO 归属Economy服务（生产队列聚合根），P3生产骨头落地后实现；当前返回空表示无引用。
	ScanProductionRefs(ctx context.Context, deletedIDs []string) ([]InstanceRef, error)
}
