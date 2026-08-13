// Package entity 实体共享内核，提供跨服务共享的实体基础类型与值对象。
//
// 本包定义SLG服务端跨服务复用的实体基础类型，包括实体ID、坐标、实体类型枚举、
// 实体状态枚举、时间戳等。所有类型遵循规范8（优先整型）：ID用int64、坐标用int32、
// 时间戳用int64毫秒。各服务domain层应引用本包类型，消除重复定义。
package entity

// EntityID 实体ID，全局唯一，由雪花算法生成。
// 使用int64而非字符串UUID，遵循规范8（优先整型）。
type EntityID int64

// EntityType 实体类型枚举，区分游戏中的不同实体种类。
// 使用int而非string，遵循规范8（优先整型表达枚举）。
type EntityType int

// EntityStatus 实体状态枚举，表示实体当前的行为状态。
// 使用int而非string，遵循规范8（优先整型表达枚举）。
type EntityStatus int

// Timestamp 时间戳，统一使用int64毫秒级Unix时间戳。
// 遵循规范8（优先整型），持久化与传输层统一用毫秒时间戳。
type Timestamp int64

// 实体类型枚举常量，覆盖SLG游戏中常见的实体种类。
// 取值映射：1=单位 2=建筑 3=英雄 4=资源点 5=据点
const (
	EntityTypeUnit       EntityType = 1 // 单位实体，可移动的战斗单位
	EntityTypeBuilding   EntityType = 2 // 建筑实体，固定在地图上的功能性建筑
	EntityTypeHero       EntityType = 3 // 英雄实体，具有特殊技能的强力单位
	EntityTypeResource   EntityType = 4 // 资源点实体，产出游戏资源的地图节点
	EntityTypeStronghold EntityType = 5 // 据点实体，可占领的战略要地
)

// 实体状态枚举常量，覆盖实体在游戏生命周期中的行为状态。
// 取值映射：1=空闲 2=移动中 3=战斗中 4=已死亡 5=采集中
const (
	EntityStatusIdle      EntityStatus = 1 // 空闲状态，实体无正在执行的动作
	EntityStatusMoving    EntityStatus = 2 // 移动中状态，实体正在执行移动订单
	EntityStatusCombat    EntityStatus = 3 // 战斗中状态，实体正在参与战斗回合
	EntityStatusDead      EntityStatus = 4 // 已死亡状态，实体被消灭等待清理
	EntityStatusGathering EntityStatus = 5 // 采集中状态，实体正在采集资源
)
