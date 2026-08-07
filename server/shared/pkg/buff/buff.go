// Package buff buff系统共享内核，提供buff模型与效果执行契约。
//
// 本包定义SLG服务端跨服务复用的buff基础模型，包括buff值对象、buff效果接口、
// buff叠加策略、buff类型枚举等。Combat/Economy/Social各服务实现具体的BuffEffect，
// 共享内核仅提供通用模型与契约。遵循规范8（优先整型）：BuffID用int64、
// Duration/ExpireTime用int64毫秒、EffectValue为float64（配置数值例外，规范8允许）。
package buff

// BuffType buff类型枚举，区分不同种类的buff效果。
// 使用int而非string，遵循规范8（优先整型表达枚举）。
type BuffType int

// buff类型枚举常量，覆盖SLG游戏中常见的buff种类。
// 取值映射：1=属性修改 2=控制 3=触发 4=治疗
const (
	BuffTypeAttrModifier BuffType = 1 // 属性修改型buff，修改实体的攻击/防御/速度等属性
	BuffTypeControl      BuffType = 2 // 控制型buff，眩晕/沉默/定身等限制实体行动
	BuffTypeTrigger      BuffType = 3 // 触发型buff，满足条件时触发额外效果（如反击/吸血）
	BuffTypeHeal         BuffType = 4 // 治疗型buff，持续恢复实体生命值
)

// BuffVO buff值对象，表示一个buff实例的完整描述。
// 值对象不可变，状态变更通过创建新实例实现。
type BuffVO struct {
	BuffID      int64    // buff实例ID，全局唯一，由雪花算法生成
	Type        BuffType // buff类型，决定效果执行方式（取值映射见BuffType枚举常量）
	Duration    int64    // buff持续时间，单位毫秒，0表示永久buff
	EffectValue float64  // 效果数值，如攻击加成比例/治疗量/控制概率，配置数值用float64（规范8例外）
	StackCount  int      // 当前叠加层数，影响效果数值的倍数
	ExpireTime  int64    // buff过期时间戳，int64毫秒级Unix时间戳，到达此时间buff失效
}

// BuffEffect buff效果执行接口，由各服务实现具体效果。
// Combat服务实现战斗buff效果（增伤/减伤/眩晕），
// Economy服务实现经济buff效果（产量加成/消耗减免），
// Social服务实现联盟buff效果（成员属性加成）。
// 接口在存在多个实现时抽象（规范4），非单一实现抽象。
type BuffEffect interface {
	// Apply 将buff效果应用到目标，返回应用结果。
	Apply(target interface{}, buff BuffVO) BuffApplyResult
}

// BuffApplyResult buff应用结果，描述buff效果应用的输出。
type BuffApplyResult struct {
	Success     bool   // 是否应用成功，false表示被免疫/抵抗/条件不满足
	ActualValue int64  // 实际生效的数值，整型（如实际增伤值/实际治疗量），遵循规范8
	Reason      string // 应用结果说明，应用失败时记录原因（如"被眩晕免疫"/"抵抗控制"）
}

// 叠加策略枚举常量，决定同种buff重复施加时的层数计算方式。
// 取值映射：1=replace 2=stack 3=refresh
const (
	stackStrategyReplace  = 1 // 替换策略，新buff替换旧buff，层数重置为incoming
	stackStrategyStack    = 2 // 叠加策略，层数累加，上限为maxStack
	stackStrategyRefresh  = 3 // 刷新策略，层数取较大值，并刷新过期时间
)

// BuffStack buff叠加策略值对象，管理同种buff的叠加层数计算。
type BuffStack struct {
	stackStrategy int // 叠加策略：1=replace 2=stack 3=refresh
	maxStack      int // 最大叠加层数，stack策略下层数上限，0表示无上限
}

// NewBuffStack 创建buff叠加策略实例。
// strategy: 1=replace 2=stack 3=refresh
// maxStack: 最大叠加层数，stack策略下生效，0表示无上限
func NewBuffStack(strategy int, maxStack int) BuffStack {
	return BuffStack{
		stackStrategy: strategy,
		maxStack:      maxStack,
	}
}

// ApplyStack 计算叠加后的层数。
// current: 当前已叠加的层数
// incoming: 新施加的层数
// 返回叠加后的最终层数。
func (s BuffStack) ApplyStack(current, incoming int) int {
	switch s.stackStrategy {
	case stackStrategyReplace:
		// 替换策略：新buff替换旧buff，层数重置为incoming
		return incoming
	case stackStrategyStack:
		// 叠加策略：层数累加，上限为maxStack
		total := current + incoming
		if s.maxStack > 0 && total > s.maxStack {
			return s.maxStack
		}
		return total
	case stackStrategyRefresh:
		// 刷新策略：层数取较大值，并刷新过期时间（过期时间由调用方处理）
		if incoming > current {
			return incoming
		}
		return current
	default:
		// 未知策略默认不叠加，保持当前层数
		return current
	}
}