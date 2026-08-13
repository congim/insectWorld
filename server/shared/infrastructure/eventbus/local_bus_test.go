package eventbus

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainevent "insectworld/server/shared/pkg/eventbus"
)

// TestLocalBusReportsConsumerFailures 验证全部消费者都会执行且失败会返回发布方。
func TestLocalBusReportsConsumerFailures(t *testing.T) {
	t.Parallel()
	bus := NewLocalBus(nil)
	called := 0
	require.NoError(t, bus.Subscribe(context.Background(), "test", func(_ context.Context, _ domainevent.DomainEvent) error {
		called++
		return errors.New("消费失败")
	}))
	require.NoError(t, bus.Subscribe(context.Background(), "test", func(_ context.Context, _ domainevent.DomainEvent) error {
		called++
		return nil
	}))
	err := bus.Publish(context.Background(), domainevent.DomainEvent{EventID: "event-1", EventType: "test"})
	require.Error(t, err)
	assert.Equal(t, 2, called)
}

// TestLocalBusRejectsUnknownEvent 验证未知事件不会被误判为投递成功。
func TestLocalBusRejectsUnknownEvent(t *testing.T) {
	t.Parallel()
	err := NewLocalBus(nil).Publish(context.Background(), domainevent.DomainEvent{EventID: "event-1", EventType: "unknown"})
	require.ErrorIs(t, err, ErrNoSubscriber)
}
