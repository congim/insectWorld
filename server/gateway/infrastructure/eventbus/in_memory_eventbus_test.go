// Package eventbus 内存事件总线实现，用于开发环境与单测。
package eventbus

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"insectworld/server/shared/pkg/eventbus"

	"go.uber.org/zap"
)

// TestInMemoryEventBus_PublishSubscribe 测试事件发布订阅往返。
func TestInMemoryEventBus_PublishSubscribe(t *testing.T) {
	logger := zap.NewNop()
	bus := NewInMemoryEventBus(logger)

	var receivedMu sync.Mutex
	var receivedEvents []eventbus.DomainEvent
	handler := func(ctx context.Context, event eventbus.DomainEvent) error {
		receivedMu.Lock()
		defer receivedMu.Unlock()
		receivedEvents = append(receivedEvents, event)
		return nil
	}

	require.NoError(t, bus.Subscribe(context.Background(), "test.event", handler))

	evt := eventbus.DomainEvent{
		EventID:     "evt-001",
		EventType:   "test.event",
		AggregateID: 1001,
		Version:     1,
		Timestamp:   1700000000000,
		Payload:     []byte(`{"key":"value"}`),
	}
	require.NoError(t, bus.Publish(context.Background(), evt))

	receivedMu.Lock()
	defer receivedMu.Unlock()
	require.Len(t, receivedEvents, 1)
	assert.Equal(t, "evt-001", receivedEvents[0].EventID)
	assert.Equal(t, int64(1001), receivedEvents[0].AggregateID)
}

// TestInMemoryEventBus_MultipleSubscribers 测试同一事件类型多订阅者均被调用。
func TestInMemoryEventBus_MultipleSubscribers(t *testing.T) {
	logger := zap.NewNop()
	bus := NewInMemoryEventBus(logger)

	var mu sync.Mutex
	callCount := 0
	makeHandler := func() eventbus.EventHandler {
		return func(ctx context.Context, event eventbus.DomainEvent) error {
			mu.Lock()
			defer mu.Unlock()
			callCount++
			return nil
		}
	}

	require.NoError(t, bus.Subscribe(context.Background(), "multi.event", makeHandler()))
	require.NoError(t, bus.Subscribe(context.Background(), "multi.event", makeHandler()))
	require.NoError(t, bus.Subscribe(context.Background(), "multi.event", makeHandler()))

	evt := eventbus.DomainEvent{EventID: "evt-002", EventType: "multi.event"}
	require.NoError(t, bus.Publish(context.Background(), evt))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 3, callCount, "三个订阅者均应被调用")
}

// TestInMemoryEventBus_NoSubscriber 测试发布无订阅者的事件会报告失败。
func TestInMemoryEventBus_NoSubscriber(t *testing.T) {
	logger := zap.NewNop()
	bus := NewInMemoryEventBus(logger)
	evt := eventbus.DomainEvent{EventID: "evt-003", EventType: "no.subscriber"}
	err := bus.Publish(context.Background(), evt)
	require.Error(t, err)
}

// TestInMemoryEventBus_HandlerError 测试订阅者返回错误时继续执行其他订阅者并向发布方报告失败。
func TestInMemoryEventBus_HandlerError(t *testing.T) {
	logger := zap.NewNop()
	bus := NewInMemoryEventBus(logger)

	var mu sync.Mutex
	successCalled := false

	errorHandler := func(ctx context.Context, event eventbus.DomainEvent) error {
		return assert.AnError
	}
	successHandler := func(ctx context.Context, event eventbus.DomainEvent) error {
		mu.Lock()
		defer mu.Unlock()
		successCalled = true
		return nil
	}

	require.NoError(t, bus.Subscribe(context.Background(), "err.event", errorHandler))
	require.NoError(t, bus.Subscribe(context.Background(), "err.event", successHandler))

	evt := eventbus.DomainEvent{EventID: "evt-004", EventType: "err.event"}
	err := bus.Publish(context.Background(), evt)
	require.Error(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, successCalled, "错误订阅者不应阻塞后续订阅者")
}

// TestInMemoryEventBus_DifferentEventTypes 测试不同事件类型订阅者隔离。
func TestInMemoryEventBus_DifferentEventTypes(t *testing.T) {
	logger := zap.NewNop()
	bus := NewInMemoryEventBus(logger)

	var mu sync.Mutex
	typeAEvents := 0
	typeBEvents := 0

	require.NoError(t, bus.Subscribe(context.Background(), "type.a", func(ctx context.Context, e eventbus.DomainEvent) error {
		mu.Lock()
		defer mu.Unlock()
		typeAEvents++
		return nil
	}))
	require.NoError(t, bus.Subscribe(context.Background(), "type.b", func(ctx context.Context, e eventbus.DomainEvent) error {
		mu.Lock()
		defer mu.Unlock()
		typeBEvents++
		return nil
	}))

	require.NoError(t, bus.Publish(context.Background(), eventbus.DomainEvent{EventType: "type.a"}))
	require.NoError(t, bus.Publish(context.Background(), eventbus.DomainEvent{EventType: "type.a"}))
	require.NoError(t, bus.Publish(context.Background(), eventbus.DomainEvent{EventType: "type.b"}))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, typeAEvents)
	assert.Equal(t, 1, typeBEvents)
}
