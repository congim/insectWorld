// Package eventbus 提供共享事件契约的本地基础设施适配。
package eventbus

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"

	domainevent "insectworld/server/shared/pkg/eventbus"
)

// ErrNoSubscriber 表示当前进程没有目标事件类型的消费者。
var ErrNoSubscriber = errors.New("事件没有本地消费者")

// LocalBus 在同一进程内同步交付事件给全部订阅者。
// 任一消费者失败都会返回聚合错误，调用方不得据此标记Outbox已发布。
type LocalBus struct {
	subscribers map[string][]domainevent.EventHandler // 事件类型到处理函数列表
	mu          sync.RWMutex                          // 保护订阅关系的并发读写
	logger      *zap.Logger                           // 结构化事件日志器
}

// NewLocalBus 创建进程内同步事件总线。
func NewLocalBus(logger *zap.Logger) *LocalBus {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &LocalBus{subscribers: make(map[string][]domainevent.EventHandler), logger: logger}
}

// Publish 同步调用全部订阅者；没有订阅者或任一处理失败时返回错误。
func (b *LocalBus) Publish(ctx context.Context, event domainevent.DomainEvent) error {
	b.mu.RLock()
	handlers := append([]domainevent.EventHandler(nil), b.subscribers[event.EventType]...)
	b.mu.RUnlock()
	if len(handlers) == 0 {
		return fmt.Errorf("事件类型未注册，eventType=%s: %w", event.EventType, ErrNoSubscriber)
	}
	var handlerErrors []error
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			b.logger.Error("本地事件消费失败", zap.String("event_type", event.EventType), zap.String("event_id", event.EventID), zap.Error(err))
			handlerErrors = append(handlerErrors, err)
		}
	}
	return errors.Join(handlerErrors...)
}

// Subscribe 注册指定事件类型的同步处理函数。
func (b *LocalBus) Subscribe(_ context.Context, eventType string, handler domainevent.EventHandler) error {
	if eventType == "" || handler == nil {
		return fmt.Errorf("事件订阅参数非法")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
	return nil
}

// 确保LocalBus实现共享事件总线契约。
var _ domainevent.EventBus = (*LocalBus)(nil)
