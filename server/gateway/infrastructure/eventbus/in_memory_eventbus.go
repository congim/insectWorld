// Package eventbus 为Gateway保留共享本地事件总线的兼容入口。
package eventbus

import (
	"go.uber.org/zap"

	sharedeventbus "insectworld/server/shared/infrastructure/eventbus"
)

// InMemoryEventBus 是共享LocalBus的兼容别名。
type InMemoryEventBus = sharedeventbus.LocalBus

// NewInMemoryEventBus 创建共享的进程内同步事件总线。
func NewInMemoryEventBus(logger *zap.Logger) *InMemoryEventBus {
	return sharedeventbus.NewLocalBus(logger)
}
