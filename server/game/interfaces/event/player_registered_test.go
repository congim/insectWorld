// Package event 将跨上下文领域事件适配为Growth应用命令。
package event

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"insectworld/server/game/application/command"
	gameerr "insectworld/server/game/domain/errors"
	gamecatalog "insectworld/server/game/infrastructure/catalog"
	"insectworld/server/game/infrastructure/memory"
	"insectworld/server/shared/pkg/eventbus"
	"insectworld/server/shared/pkg/gamepack"
)

// TestPlayerRegisteredHandlerIsIdempotent 验证重复注册事件只创建一次玩家并发放一次资源。
func TestPlayerRegisteredHandlerIsIdempotent(t *testing.T) {
	t.Parallel()
	pack, err := gamepack.LoadAndCompile(filepath.Join("..", "..", "..", "..", "gamepacks", "insect-world"), "0.1.0")
	require.NoError(t, err)
	reader, err := gamecatalog.NewGamePackReader(pack)
	require.NoError(t, err)
	resources := memory.NewResourceAccount()
	service := command.NewService(memory.NewPlayerRepository(), memory.NewBuildingRepository(), memory.NewTrainingRepository(), memory.NewUnitRoster(), reader, resources, memory.NewIDGenerator(1), zap.NewNop())
	handler := NewPlayerRegisteredHandler(service)
	payload, err := json.Marshal(playerRegisteredPayload{PlayerID: 10, Username: "注册玩家", RegisteredAt: 1000})
	require.NoError(t, err)
	event := eventbus.DomainEvent{EventID: "auth.player_registered:10", EventType: EventTypePlayerRegistered, AggregateID: 10, Version: 1, Timestamp: 1000, Payload: payload}
	require.NoError(t, handler.Handle(context.Background(), event))
	require.NoError(t, handler.Handle(context.Background(), event))
	balances, err := resources.Balances(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, int64(100), balances["food"])
}

// TestPlayerRegisteredHandlerRejectsMismatchedPayload 验证事件头和负载不一致时拒绝创建玩家。
func TestPlayerRegisteredHandlerRejectsMismatchedPayload(t *testing.T) {
	t.Parallel()
	payload, err := json.Marshal(playerRegisteredPayload{PlayerID: 2, Username: "玩家", RegisteredAt: 1000})
	require.NoError(t, err)
	handler := NewPlayerRegisteredHandler(nil)
	err = handler.Handle(context.Background(), eventbus.DomainEvent{EventID: "event-1", EventType: EventTypePlayerRegistered, AggregateID: 1, Timestamp: 1000, Payload: payload})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrInvalidCommand))
}
