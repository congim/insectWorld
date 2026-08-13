// Package event 将跨上下文领域事件适配为Growth应用命令。
package event

import (
	"context"
	"encoding/json"
	"fmt"

	"insectworld/server/game/application/command"
	gameerr "insectworld/server/game/domain/errors"
	"insectworld/server/shared/pkg/eventbus"
)

// EventTypePlayerRegistered 是Gateway发布的玩家注册事件稳定类型。
const EventTypePlayerRegistered = "auth.player_registered"

type playerRegisteredPayload struct {
	PlayerID     int64  // 玩家ID
	Username     string // 注册用户名，首版作为默认昵称
	RegisteredAt int64  // 注册完成时间戳，Unix毫秒
}

// PlayerRegisteredHandler 将注册事件幂等转换为创建玩家命令。
type PlayerRegisteredHandler struct {
	service *command.Service // Growth写命令服务
}

// NewPlayerRegisteredHandler 创建玩家注册事件处理器。
func NewPlayerRegisteredHandler(service *command.Service) *PlayerRegisteredHandler {
	return &PlayerRegisteredHandler{service: service}
}

// Handle 消费玩家注册事件；eventID直接作为创建玩家幂等键。
func (h *PlayerRegisteredHandler) Handle(ctx context.Context, event eventbus.DomainEvent) error {
	if event.EventID == "" || event.EventType != EventTypePlayerRegistered || event.AggregateID <= 0 || event.Timestamp <= 0 {
		return fmt.Errorf("玩家注册事件头非法，eventID=%s: %w", event.EventID, gameerr.ErrInvalidCommand)
	}
	var payload playerRegisteredPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("玩家注册事件负载解析失败，eventID=%s: %w", event.EventID, gameerr.ErrInvalidCommand)
	}
	if payload.PlayerID != event.AggregateID || payload.RegisteredAt != event.Timestamp || payload.Username == "" {
		return fmt.Errorf("玩家注册事件头与负载不一致，eventID=%s: %w", event.EventID, gameerr.ErrInvalidCommand)
	}
	_, err := h.service.CreatePlayer(ctx, command.CreatePlayerCommand{CommandID: event.EventID, PlayerID: payload.PlayerID, Nickname: payload.Username, NowMs: payload.RegisteredAt})
	return err
}
