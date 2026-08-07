// Package event 用户认证领域事件，发布玩家上线/下线通知供其他服务订阅。
//
// domain层零外部依赖（规范3），事件可转换为共享内核eventbus.DomainEvent通过EventBus发布。
// 事件类型常量在文件顶部const块定义（规范1就近归属）。
package event

import (
	"encoding/json"

	"insectworld/server/shared/pkg/eventbus"
)

// 事件类型常量，用于EventBus订阅分发（规范1就近归属）。
const (
	EventTypePlayerOnline  = "auth.player_online"  // 玩家上线事件
	EventTypePlayerOffline = "auth.player_offline" // 玩家下线事件
)

// 下线原因枚举常量，表示玩家下线的触发原因。
// 取值映射：1=主动登出 2=会话超时 3=被踢下线 4=封禁踢下线
const (
	OfflineReasonLogout         = 1 // 主动登出
	OfflineReasonSessionTimeout = 2 // 会话超时
	OfflineReasonKicked         = 3 // 被踢下线（单点登录）
	OfflineReasonBanned         = 4 // 封禁踢下线
)

// PlayerOnlineEvent 玩家上线事件，登录成功后发布。
//
// 所有时间戳用int64毫秒（规范8），其他服务订阅此事件初始化玩家相关数据。
type PlayerOnlineEvent struct {
	PlayerID  int64  // 玩家ID
	LoginTime int64  // 登录时间戳，毫秒级
	SourceIP  string // 登录来源IP
}

// EventType 返回事件类型。
func (e *PlayerOnlineEvent) EventType() string {
	return EventTypePlayerOnline
}

// ToDomainEvent 转换为共享内核DomainEvent，通过EventBus发布。
//
// eventID为事件ID（UUID），version为聚合根版本号。
// Payload由JSON序列化生成。
func (e *PlayerOnlineEvent) ToDomainEvent(eventID string, version int) (eventbus.DomainEvent, error) {
	payload, err := json.Marshal(e)
	if err != nil {
		return eventbus.DomainEvent{}, err
	}
	return eventbus.DomainEvent{
		EventID:     eventID,
		EventType:   EventTypePlayerOnline,
		AggregateID: e.PlayerID,
		Version:     version,
		Timestamp:   e.LoginTime,
		Payload:     payload,
	}, nil
}

// PlayerOfflineEvent 玩家下线事件，登出/超时/踢下线时发布。
//
// Reason字段注释列取值映射（规范6），其他服务订阅此事件清理玩家相关数据。
type PlayerOfflineEvent struct {
	PlayerID    int64 // 玩家ID
	OfflineTime int64 // 下线时间戳，毫秒级
	Reason      int   // 下线原因：1=主动登出 2=会话超时 3=被踢下线 4=封禁踢下线
}

// EventType 返回事件类型。
func (e *PlayerOfflineEvent) EventType() string {
	return EventTypePlayerOffline
}

// ToDomainEvent 转换为共享内核DomainEvent，通过EventBus发布。
//
// eventID为事件ID（UUID），version为聚合根版本号。
// Payload由JSON序列化生成。
func (e *PlayerOfflineEvent) ToDomainEvent(eventID string, version int) (eventbus.DomainEvent, error) {
	payload, err := json.Marshal(e)
	if err != nil {
		return eventbus.DomainEvent{}, err
	}
	return eventbus.DomainEvent{
		EventID:     eventID,
		EventType:   EventTypePlayerOffline,
		AggregateID: e.PlayerID,
		Version:     version,
		Timestamp:   e.OfflineTime,
		Payload:     payload,
	}, nil
}
