// Package eventbus 内存事件总线实现，用于开发环境与单测。
//
// infrastructure层技术适配，实现shared/kernel/eventbus.EventBus接口。
// 生产环境应替换为NATS适配，此实现仅用于无外部消息中间件的场景。
package eventbus

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"insectworld/server/shared/pkg/eventbus"
)

// InMemoryEventBus 内存事件总线，实现EventBus接口。
//
// 事件发布后同步调用订阅者处理函数，保证事件不丢。
// 仅用于开发环境与单测，生产环境应使用NATS适配。
type InMemoryEventBus struct {
	subscribers map[string][]eventbus.EventHandler // 订阅者表，key=事件类型
	mu          sync.RWMutex                       // 读写锁，保护订阅者表并发访问
	logger      *zap.Logger                        // 结构化日志
}

// NewInMemoryEventBus 创建内存事件总线实例。
func NewInMemoryEventBus(logger *zap.Logger) *InMemoryEventBus {
	return &InMemoryEventBus{
		subscribers: make(map[string][]eventbus.EventHandler),
		logger:      logger,
	}
}

// Publish 发布领域事件到事件总线。
//
// 同步调用所有订阅者处理函数，处理失败记录Error日志但不阻塞其他订阅者。
func (b *InMemoryEventBus) Publish(ctx context.Context, event eventbus.DomainEvent) error {
	b.mu.RLock()
	handlers := b.subscribers[event.EventType]
	b.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			b.logger.Error("事件处理失败",
				zap.String("event_type", event.EventType),
				zap.String("event_id", event.EventID),
				zap.Error(err),
			)
		}
	}
	return nil
}

// Subscribe 订阅指定事件类型，注册处理函数。
func (b *InMemoryEventBus) Subscribe(ctx context.Context, eventType string, handler eventbus.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
	return nil
}

// 确保 InMemoryEventBus 实现 EventBus 接口（编译期检查）。
var _ eventbus.EventBus = (*InMemoryEventBus)(nil)
