// Package config 共享内核配置模块，提供配置加载/校验/查询统一API。
// 本文件定义配置查询返回的类型结构体，这些结构体代表配置包编译后的内部数据结构。
// 业务服务通过ConfigQueryAPI查询这些配置驱动业务执行。
package config

// TerrainConfig 地形配置，描述一种地形类型的移动消耗、防御加成等属性。
type TerrainConfig struct {
	TerrainID    string // 地形类型ID，如"plain"/"mountain"/"water"
	MoveCost     int64  // 移动力消耗，进入该地形格需消耗的移动力
	DefenseBonus int64  // 防御加成百分比，该地形上部队获得的防御加成
	IsBlock      bool   // 是否阻挡移动，true表示该地形不可通行
	IsBuildable  bool   // 是否可建造，true表示该地形上可建造建筑
}

// MovementTypeConfig 移动类型配置，描述一种移动类型的速度、消耗等属性。
type MovementTypeConfig struct {
	TypeID          string // 移动类型ID，如"march"/"reinforce"/"teleport"
	BaseSpeed       int64  // 基础速度，每Tick移动的格子数
	MoveCostPerCell int64  // 每格移动力消耗
	MaxDistance     int64  // 最大移动距离，单次移动允许的最大格子数
	AllowFormation  bool   // 是否允许编队移动
}

// CombatTypeConfig 战斗类型配置，描述一种战斗类型的最大轮数等属性。
type CombatTypeConfig struct {
	TypeID         string // 战斗类型ID，如"field"/"siege"/"city"
	MaxRounds      int    // 最大轮数，超过则平局
	AllowSkill     bool   // 是否允许使用技能
	AllowFormation bool   // 是否允许阵型效果
	DefenderBonus  int64  // 防守方加成百分比
}

// SkillConfig 技能配置，描述一个技能的触发条件、效果、冷却等属性。
type SkillConfig struct {
	SkillID        string // 技能ID，全局唯一
	SkillName      string // 技能名称，用于日志展示
	CooldownRounds int    // 冷却轮数，技能使用后需等待的轮数
	TriggerPhase   int    // 触发阶段：1=轮次开始 2=轮次中 3=轮次结束
	EffectType     int    // 效果类型：1=伤害 2=治疗 3=增益 4=减益
	EffectValue    int64  // 效果数值，如伤害值/治疗量
	TargetType     int    // 目标类型：1=自身 2=敌方 3=友方 4=全体
}

// FormationEffectConfig 阵型效果配置，描述一种阵型对战斗的加成效果。
type FormationEffectConfig struct {
	FormationID       string // 阵型ID，如"standard"/"offensive"/"defensive"
	AttackBonus       int64  // 攻击加成百分比
	DefenseBonus      int64  // 防御加成百分比
	ApplyPhase        int    // 应用时机：1=轮次开始 2=轮次结束
	RequiredUnitCount int    // 所需最低部队数量
}

// FormulaConfig 公式配置，描述一个伤害/治疗计算公式的系数。
type FormulaConfig struct {
	FormulaID  string  // 公式ID，如"physical_damage"/"magical_damage"
	BaseCoeff  float64 // 基础系数，配置公式可为小数（规范8例外）
	AttackExp  float64 // 攻击力指数
	DefenseExp float64 // 防御力指数
	FlatBonus  int64   // 固定加成值
}

// CounterMatrixConfig 兵种克制矩阵配置，描述兵种间的克制系数。
type CounterMatrixConfig struct {
	Matrix map[string]map[string]float64 // 克制矩阵，key=攻击兵种ID，value=防御兵种ID→克制系数
}

// ProductionRuleConfig 生产规则配置，描述一种资源的生产规则。
type ProductionRuleConfig struct {
	RuleID         string           // 规则ID，全局唯一
	OutputResource string           // 产出资源类型
	OutputRate     int64            // 产出速率，每Tick产出数量
	InputResources map[string]int64 // 消耗资源类型→数量
	RequiredLevel  int              // 所需建筑等级
}

// TradeRuleConfig 交易规则配置，描述资源交易的汇率与税率。
type TradeRuleConfig struct {
	RuleID       string  // 规则ID，全局唯一
	FromResource string  // 卖出资源类型
	ToResource   string  // 买入资源类型
	ExchangeRate float64 // 汇率，配置数值可为小数（规范8例外）
	TaxRate      float64 // 税率，0-1之间
	MinAmount    int64   // 最小交易量
	MaxAmount    int64   // 最大交易量
}

// ConversionRuleConfig 资源转换规则配置，描述资源间的转换比例。
type ConversionRuleConfig struct {
	RuleID       string  // 规则ID，全局唯一
	FromResource string  // 转换源资源类型
	ToResource   string  // 转换目标资源类型
	Ratio        float64 // 转换比例，配置数值可为小数（规范8例外）
	CooldownMs   int64   // 冷却时间（毫秒）
}

// StorageRuleConfig 存储规则配置，描述资源存储上限与溢出处理。
type StorageRuleConfig struct {
	ResourceType     string // 资源类型
	MaxCapacity      int64  // 最大容量
	OverflowBehavior int    // 溢出处理：1=丢弃 2=停止生产 3=转换为其他
	ConvertTo        string // 溢出转换目标资源类型，OverflowBehavior=3时生效
}

// EconomyModifiersConfig 经济修正器配置，描述科技/联盟/赛季对经济的加成。
type EconomyModifiersConfig struct {
	TechModifier     map[string]float64 // 科技加成，key=资源类型，value=加成系数
	AllianceModifier map[string]float64 // 联盟加成，key=资源类型，value=加成系数
	SeasonModifier   map[string]float64 // 赛季加成，key=资源类型，value=加成系数
}

// WelfareConfig 联盟福利配置，描述一种福利的触发条件与效果。
type WelfareConfig struct {
	WelfareID             string // 福利ID，全局唯一
	WelfareType           int    // 福利类型：1=每日 2=每周 3=一次性
	EffectType            int    // 效果类型：1=资源 2=加速 3=buff
	EffectValue           int64  // 效果数值
	RequiredAllianceLevel int    // 所需联盟等级
}

// PhaseConfig 赛季阶段配置，描述一个阶段的持续时间与允许操作。
type PhaseConfig struct {
	PhaseID        string   // 阶段ID，如"preparation"/"war"/"settlement"
	PhaseName      string   // 阶段名称，用于日志展示
	DurationMs     int64    // 持续时间（毫秒）
	AllowedActions []string // 允许的操作列表
}

// TransitionRuleConfig 阶段切换规则配置，描述阶段间切换的条件。
type TransitionRuleConfig struct {
	FromPhase      string // 源阶段ID
	ToPhase        string // 目标阶段ID
	Condition      string // 切换条件描述
	AutoTransition bool   // 是否自动切换
}

// ResetRulesConfig 赛季重置规则配置，描述重置范围与规则。
type ResetRulesConfig struct {
	ResetScope     []string // 重置范围，如["player","alliance","economy"]
	KeepData       []string // 保留数据，如["player_profile","alliance_basic"]
	ResetTimestamp int64    // 重置时间点（毫秒时间戳）
}

// InheritRuleConfig 赛季继承规则配置，描述跨赛季数据继承的公式。
type InheritRuleConfig struct {
	RuleID       string  // 规则ID，全局唯一
	DataType     string  // 继承数据类型，如"player_level"/"alliance_tech"
	InheritRatio float64 // 继承比例，配置数值可为小数（规范8例外）
	MaxInherit   int64   // 最大继承值
}

// RewardConfig 赛季奖励配置，描述一个奖励档位的条件与内容。
type RewardConfig struct {
	RewardID  string           // 奖励ID，全局唯一
	MinRank   int              // 最低排名
	MaxRank   int              // 最高排名
	Resources map[string]int64 // 奖励资源类型→数量
	Items     map[string]int64 // 奖励道具类型→数量
}

// ScoringRuleConfig 积分规则配置，描述积分计算方式。
type ScoringRuleConfig struct {
	RuleID     string  // 规则ID，全局唯一
	ActionType string  // 行为类型，如"kill"/"capture"/"build"
	ScoreValue int64   // 积分值
	Weight     float64 // 权重，配置数值可为小数（规范8例外）
}

// EventTypeConfig 事件类型配置，描述一种游戏事件的触发与持续时间。
type EventTypeConfig struct {
	TypeID           string // 事件类型ID，全局唯一
	TypeName         string // 事件名称，用于日志展示
	DurationMs       int64  // 持续时间（毫秒），0表示永久
	TriggerCondition string // 触发条件描述
	EffectScope      string // 效果范围，如"global"/"player"/"alliance"
}
