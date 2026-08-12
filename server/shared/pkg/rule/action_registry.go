package rule

import "fmt"

// RuleActionHandler 规则动作处理器，注册到ActionRegistry后由规则引擎按动作类型分发调用。
// 每个动作类型全局唯一（Type()返回值），题材层新动作实现本接口并注册，框架零改动。
type RuleActionHandler interface {
	// Type 返回动作类型标识，如"apply_buff"，作为注册与配置引用的唯一key。
	Type() string
	// Execute 执行动作，基于ActionContext执行并写入结果；返回错误时由调用方记录并跳过后续动作。
	Execute(ctx ActionContext) error
}

// ActionContext 规则动作执行上下文，封装触发上下文、动作参数与结果写入。
// 由规则引擎在执行动作前构造，同一规则内的多个动作共享一份上下文。
type ActionContext struct {
	RuleID    string         // 规则ID，用于日志与审计关联
	EventName string         // 触发事件名，如"on_combat_start"
	Input     map[string]any // 触发上下文变量（attacker/defender/terrain等），由事件源提供
	Params    map[string]any // 动作参数，来自配置action_params，字段含义由各动作契约定义
	Output    map[string]any // 结果写入区，动作副作用（如生成的实体ID）写回供后续动作与审计使用
	Accessor  ActionAccessor // 领域服务访问端口，由宿主服务实现并注入，动作经此操作实体/资源/Buff等
}

// ActionAccessor 动作访问端口，由宿主服务实现并注入ActionContext，
// 使共享内核的动作处理器不依赖任何服务domain实现（端口-适配器模式，符合DDD依赖方向）。
type ActionAccessor interface {
	// ApplyBuff 给目标实体挂载Buff。
	ApplyBuff(ctx ActionContext, target string, buffID string) error
	// RemoveBuff 按BuffID移除目标实体上的Buff。
	RemoveBuff(ctx ActionContext, target string, buffID string) error
	// RemoveBuffByTag 按来源标记批量移除目标实体上的Buff。
	RemoveBuffByTag(ctx ActionContext, target string, sourceTag string) error
	// ModifyResource 修改目标资源余额，amount为正增加、为负扣减（int64，AGENTS.md规范8）。
	ModifyResource(ctx ActionContext, target string, resourceID string, amount int64) error
	// SpawnEntity 按模板生成实体，返回生成的实体ID。
	SpawnEntity(ctx ActionContext, templateID string, posX int32, posY int32, owner string) (int64, error)
	// TriggerCombat 触发一场战斗。
	TriggerCombat(ctx ActionContext, attackerID string, defenderID string) error
	// SendNotify 向目标发送通知消息。
	SendNotify(ctx ActionContext, target string, msgID string, params map[string]any) error
	// ChangeTerrain 修改指定位置的地形。
	ChangeTerrain(ctx ActionContext, posX int32, posY int32, terrainID string) error
}

// ActionRegistry 动作注册表，管理动作类型到处理器的映射。
// 规则引擎执行规则时按action_type查询处理器，未注册的动作类型返回错误。
type ActionRegistry struct {
	handlers map[string]RuleActionHandler // 动作类型到处理器的映射，key为Type()返回值
}

// NewActionRegistry 创建动作注册表实例。
func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{handlers: make(map[string]RuleActionHandler)}
}

// Register 注册动作处理器，Type()为空、handler为nil或重复注册返回错误。
func (r *ActionRegistry) Register(handler RuleActionHandler) error {
	if handler == nil {
		return fmt.Errorf("动作处理器不能为空")
	}
	actionType := handler.Type()
	if actionType == "" {
		return fmt.Errorf("动作类型不能为空")
	}
	if _, exists := r.handlers[actionType]; exists {
		return fmt.Errorf("动作类型 %s 已注册，禁止重复注册", actionType)
	}
	r.handlers[actionType] = handler
	return nil
}

// Get 查询动作处理器，未注册时返回错误。
func (r *ActionRegistry) Get(actionType string) (RuleActionHandler, error) {
	handler, exists := r.handlers[actionType]
	if !exists {
		return nil, fmt.Errorf("动作类型 %s 未注册处理器", actionType)
	}
	return handler, nil
}

// Types 返回全部已注册的动作类型，用于启动全量校验与诊断输出。
func (r *ActionRegistry) Types() []string {
	types := make([]string, 0, len(r.handlers))
	for actionType := range r.handlers {
		types = append(types, actionType)
	}
	return types
}

// globalActionRegistry 全局动作注册表，供各包init()注册（blank import扩展）。
var globalActionRegistry = NewActionRegistry()

// RegisterAction 全局注册动作处理器，供init()调用。
// 注册失败（重复/空类型）属于编程错误，panic让进程在启动期失败，符合"启动全量校验"原则。
func RegisterAction(handler RuleActionHandler) {
	if err := globalActionRegistry.Register(handler); err != nil {
		panic("规则动作注册失败: " + err.Error())
	}
}

// GlobalActionRegistry 返回全局动作注册表，规则引擎启动时获取并注入校验器。
func GlobalActionRegistry() *ActionRegistry { return globalActionRegistry }

// ValidateTypes 启动全量校验：检查配置引用的动作类型是否全部已注册，返回缺失类型列表。
// 非空时服务拒绝启动——配置引用未注册动作=配置错误，越早暴露越好。
func ValidateTypes(configuredTypes []string) []string {
	registered := globalActionRegistry.Types()
	regSet := make(map[string]struct{}, len(registered))
	for _, actionType := range registered {
		regSet[actionType] = struct{}{}
	}
	var missing []string
	for _, actionType := range configuredTypes {
		if _, ok := regSet[actionType]; !ok {
			missing = append(missing, actionType)
		}
	}
	return missing
}

// parseInt64Param 从动作参数中解析int64数值，支持int64/int/int32/float64常见配置类型。
// 配置来源可能为JSON解析后的float64（AGENTS.md 8.1.8配置数值例外），需兼容转换；
// 资源数量、坐标等最终以整型参与运算（AGENTS.md 8.1.2/8.1.6）。
func parseInt64Param(params map[string]any, key string) (int64, bool) {
	raw, ok := params[key]
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

// parsePositionParam 从动作参数中解析位置参数position{x,y}，返回int32格子坐标。
// 参数结构为嵌套map：{"position": {"x": 数值, "y": 数值}}，坐标为格子坐标（AGENTS.md 8.1.6）。
func parsePositionParam(params map[string]any, key string) (posX int32, posY int32, ok bool) {
	raw, exists := params[key]
	if !exists {
		return 0, 0, false
	}
	posMap, isMap := raw.(map[string]any)
	if !isMap {
		return 0, 0, false
	}
	x, okX := parseInt64Param(posMap, "x")
	y, okY := parseInt64Param(posMap, "y")
	if !okX || !okY {
		return 0, 0, false
	}
	return int32(x), int32(y), true
}
