// Package integration 跨服务集成测试，验证模块化单体内限界上下文协作。
package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	gamecommand "insectworld/server/game/application/command"
	gamecatalog "insectworld/server/game/infrastructure/catalog"
	"insectworld/server/game/infrastructure/memory"
	gameevent "insectworld/server/game/interfaces/event"
	gatewayevent "insectworld/server/gateway/domain/event"
	sharedeventbus "insectworld/server/shared/infrastructure/eventbus"
	"insectworld/server/shared/pkg/eventbus"
	"insectworld/server/shared/pkg/eventbus/publisher"
	"insectworld/server/shared/pkg/gamepack"
)

type deliveryOutbox struct {
	record    eventbus.OutboxRecord // 唯一测试事件
	published bool                  // 是否已标记发布
	failed    bool                  // 是否经历发布失败
}

func (o *deliveryOutbox) Save(_ context.Context, record eventbus.OutboxRecord) error {
	o.record = record
	return nil
}
func (o *deliveryOutbox) MarkPublished(_ context.Context, _ string, publishTime int64) error {
	o.published = true
	o.record.Status = eventbus.OutboxStatusPublished
	o.record.PublishTime = publishTime
	return nil
}
func (o *deliveryOutbox) MarkFailed(_ context.Context, _ string, nextAttemptMs int64, _ string) error {
	o.failed = true
	o.record.Status = eventbus.OutboxStatusFailed
	o.record.RetryCount++
	o.record.AvailableTime = nextAttemptMs
	return nil
}
func (o *deliveryOutbox) GetPending(_ context.Context, _ int) ([]eventbus.OutboxRecord, error) {
	return []eventbus.OutboxRecord{o.record}, nil
}
func (o *deliveryOutbox) ClaimPending(_ context.Context, nowMs int64, eventTypes []string, _ int, leaseUntilMs int64) ([]eventbus.OutboxRecord, error) {
	if o.published || o.record.AvailableTime > nowMs {
		return nil, nil
	}
	if len(eventTypes) != 1 || eventTypes[0] != o.record.EventType {
		return nil, nil
	}
	o.record.Status = eventbus.OutboxStatusProcessing
	o.record.AvailableTime = leaseUntilMs
	return []eventbus.OutboxRecord{o.record}, nil
}

// TestRegistrationOutboxDeliversToGrowth 验证注册Outbox经发布器和事件总线幂等创建玩家。
func TestRegistrationOutboxDeliversToGrowth(t *testing.T) {
	t.Parallel()
	pack, err := gamepack.LoadAndCompile(filepath.Join("..", "..", "gamepacks", "insect-world"), "0.1.0")
	require.NoError(t, err)
	reader, err := gamecatalog.NewGamePackReader(pack)
	require.NoError(t, err)
	resources := memory.NewResourceAccount()
	growth := gamecommand.NewService(memory.NewPlayerRepository(), memory.NewBuildingRepository(), memory.NewTrainingRepository(), memory.NewUnitRoster(), reader, resources, memory.NewIDGenerator(1), zap.NewNop())
	handler := gameevent.NewPlayerRegisteredHandler(growth)
	bus := sharedeventbus.NewLocalBus(zap.NewNop())
	require.NoError(t, bus.Subscribe(context.Background(), gameevent.EventTypePlayerRegistered, handler.Handle))
	registered, err := (gatewayevent.PlayerRegisteredEvent{PlayerID: 77, Username: "投递玩家", RegisteredAt: 1000}).ToDomainEvent()
	require.NoError(t, err)
	outbox := &deliveryOutbox{record: eventbus.OutboxRecord{EventID: registered.EventID, AggregateID: registered.AggregateID, EventType: registered.EventType, Version: registered.Version, Payload: registered.Payload, Status: eventbus.OutboxStatusPending, CreateTime: registered.Timestamp}}
	delivery, err := publisher.New(outbox, bus, publisher.Config{EventTypes: []string{gameevent.EventTypePlayerRegistered}, BatchSize: 10, PollInterval: time.Second, LeaseDuration: time.Minute, BaseRetryDelay: time.Second, MaxRetryDelay: time.Minute}, zap.NewNop())
	require.NoError(t, err)
	require.NoError(t, delivery.PollOnce(context.Background()))
	assert.True(t, outbox.published)
	balances, err := resources.Balances(context.Background(), 77)
	require.NoError(t, err)
	assert.Equal(t, int64(100), balances["food"])

	outbox.published = false
	outbox.record.Status = eventbus.OutboxStatusPending
	outbox.record.AvailableTime = 0
	require.NoError(t, delivery.PollOnce(context.Background()))
	balances, err = resources.Balances(context.Background(), 77)
	require.NoError(t, err)
	assert.Equal(t, int64(100), balances["food"])
}
