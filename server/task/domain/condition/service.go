// Package condition 条件判定domain service，从配置查询条件表达式并求值。
// 引用shared/pkg/rule的ConditionEvaluator，配置驱动的条件判定。
// 对应spec.md 5.1.7.3节Task上下文功能1"任务触发与推进"。
package condition

// ConditionType 条件类型，对应tasks.json/achievements.json的条件定义。
type ConditionType int

// 条件类型常量（规范1），标识任务/成就的触发条件类型。
const (
	ConditionCombatVictory   ConditionType = 1 // 战斗胜利
	ConditionResourceProduce ConditionType = 2 // 资源产出
	ConditionAllianceJoin    ConditionType = 3 // 联盟加入
	ConditionBuildingUpgrade ConditionType = 4 // 建筑升级
	ConditionTechResearch    ConditionType = 5 // 科技研究
	ConditionTerrainChange   ConditionType = 6 // 地形变更
	ConditionKillCount       ConditionType = 7 // 击杀数量
)

// ConditionContext 条件判定上下文，携带业务事件的参数。
type ConditionContext struct {
	EventType string           // 事件类型（如"combat.ended"）
	Params    map[string]int64 // 事件参数（如战斗胜利次数、资源产出量）
}

// ConditionEvaluator 条件判定器，从配置查询条件表达式并求值。
// 实际实现引用shared/pkg/rule的ConditionEvaluator，本接口为domain层声明。
type ConditionEvaluator interface {
	// Evaluate 判定条件是否满足，返回是否满足与增量进度。
	Evaluate(ctx ConditionContext, conditionExpr string) (satisfied bool, delta int64)
}
