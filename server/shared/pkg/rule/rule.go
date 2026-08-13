// Package rule 通用规则引擎共享内核，提供规则执行契约与注册表。
//
// 本包定义SLG服务端跨服务复用的通用规则引擎模型，包括规则上下文、规则执行器接口、
// 规则注册表、规则结果、条件求值器接口。规则引擎通过扩展点ID驱动，各服务注册具体的
// RuleExecutor实现，业务代码通过ExtensionPointID查询并执行规则。
// 遵循规范4（不过度设计）：RuleContext的Input map[string]any由调用方保证强类型，
// 避免any滥用；RuleExecutor接口在存在多个规则实现时抽象。
package rule

import "fmt"

// RuleContext 规则执行上下文，封装规则执行所需的输入与配置。
type RuleContext struct {
	ExtensionPointID string         // 扩展点ID，如"combat.damage_formulas"/"economy.production_rules"，标识规则归属
	Input            map[string]any // 规则输入参数，由调用方保证强类型（规范4），如{"attacker": combatUnit, "target": targetUnit}
	Config           any            // 规则配置，从ExtensionRegistry查询的配置数据，类型由具体扩展点决定
}

// RuleExecutor 规则执行器接口，执行具体规则逻辑并返回结果。
// 各服务实现具体的RuleExecutor，如Combat服务实现伤害公式执行器、
// Economy服务实现产量公式执行器。接口在存在多个规则实现时抽象（规范4）。
type RuleExecutor interface {
	// Execute 执行规则，基于上下文计算规则结果。
	Execute(ctx RuleContext) RuleResult
}

// RuleResult 规则执行结果，描述规则执行的输出与判定。
type RuleResult struct {
	Success bool           // 规则是否执行成功，false表示条件不满足或执行异常
	Output  map[string]any // 规则输出，如{"damage": 150, "critical": true}，由调用方按约定类型读取
	Reason  string         // 结果说明，失败时记录原因（如"攻击者已死亡"/"资源不足"）
}

// ConditionEvaluator 条件求值器接口，判定规则前置条件是否满足。
// 在规则执行前调用，条件不满足时跳过规则执行，提升规则引擎效率。
type ConditionEvaluator interface {
	// Evaluate 求值条件，返回是否满足。
	Evaluate(ctx RuleContext) bool
}

// RuleRegistry 规则注册表，管理扩展点ID到规则执行器的映射。
// 各服务在启动时注册具体的RuleExecutor，业务代码通过ExtensionPointID查询执行器。
type RuleRegistry struct {
	executors map[string]RuleExecutor // 扩展点ID到执行器的映射，key为ExtensionPointID
}

// NewRuleRegistry 创建规则注册表实例。
func NewRuleRegistry() *RuleRegistry {
	return &RuleRegistry{
		executors: make(map[string]RuleExecutor),
	}
}

// Register 注册规则执行器到指定扩展点。
// extPointID: 扩展点ID，如"combat.damage_formulas"
// executor: 规则执行器实例
// 重复注册同一扩展点返回错误，避免覆盖已注册的执行器。
func (r *RuleRegistry) Register(extPointID string, executor RuleExecutor) error {
	if extPointID == "" {
		return fmt.Errorf("扩展点ID不能为空")
	}
	if executor == nil {
		return fmt.Errorf("规则执行器不能为空")
	}
	if _, exists := r.executors[extPointID]; exists {
		return fmt.Errorf("扩展点 %s 已注册执行器，禁止重复注册", extPointID)
	}
	r.executors[extPointID] = executor
	return nil
}

// Get 查询指定扩展点的规则执行器。
// extPointID: 扩展点ID
// 返回执行器实例，未注册时返回错误。
func (r *RuleRegistry) Get(extPointID string) (RuleExecutor, error) {
	executor, exists := r.executors[extPointID]
	if !exists {
		return nil, fmt.Errorf("扩展点 %s 未注册执行器", extPointID)
	}
	return executor, nil
}
