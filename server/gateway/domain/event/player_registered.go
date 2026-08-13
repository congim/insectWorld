// Package event 用户认证领域事件，发布玩家身份生命周期通知。
package event

import (
	"encoding/json"
	"strconv"

	"insectworld/server/shared/pkg/eventbus"
)

// EventTypePlayerRegistered 是账号与玩家身份创建完成事件类型。
const EventTypePlayerRegistered = "auth.player_registered"

// PlayerRegisteredEvent 是Growth创建玩家档案的事实来源。
type PlayerRegisteredEvent struct {
	PlayerID     int64  // 玩家ID
	Username     string // 注册用户名，首版作为默认昵称
	RegisteredAt int64  // 注册完成时间戳，Unix毫秒
}

// ToDomainEvent 转换为可写入Outbox的共享领域事件。
func (e PlayerRegisteredEvent) ToDomainEvent() (eventbus.DomainEvent, error) {
	payload, err := json.Marshal(e)
	if err != nil {
		return eventbus.DomainEvent{}, err
	}
	return eventbus.DomainEvent{EventID: EventTypePlayerRegistered + ":" + formatPlayerID(e.PlayerID), EventType: EventTypePlayerRegistered, AggregateID: e.PlayerID, Version: 1, Timestamp: e.RegisteredAt, Payload: payload}, nil
}

func formatPlayerID(playerID int64) string {
	return strconv.FormatInt(playerID, 10)
}
