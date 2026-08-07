// Package event Persist服务事件订阅，监听领域事件触发归档与快照操作。
//
// 订阅combat.ended事件触发战报归档，订阅operation.season.ended事件触发赛季快照。
// interfaces层依赖application层，不直接依赖infrastructure（规范3）。
package event

import (
	"context"

	"go.uber.org/zap"

	"insectworld/server/shared/pkg/eventbus"
)

// Handler Persist服务事件订阅handler。
type Handler struct {
	eventBus eventbus.EventBus // 事件总线，订阅领域事件
	logger   *zap.Logger       // 结构化日志
}

// NewHandler 创建事件订阅handler实例。
func NewHandler(eventBus eventbus.EventBus, logger *zap.Logger) *Handler {
	return &Handler{
		eventBus: eventBus,
		logger:   logger,
	}
}

// SubscribeAll 订阅所有相关领域事件。
// 订阅combat.ended触发战报归档，operation.season.ended触发赛季快照。
func (h *Handler) SubscribeAll(ctx context.Context) error {
	if err := h.eventBus.Subscribe(ctx, "combat.ended", h.handleCombatEnded); err != nil {
		return err
	}
	if err := h.eventBus.Subscribe(ctx, "operation.season.ended", h.handleSeasonEnded); err != nil {
		return err
	}
	h.logger.Info("Persist服务事件订阅启动成功")
	return nil
}

// handleCombatEnded 处理战斗结束事件，触发战报归档。
func (h *Handler) handleCombatEnded(ctx context.Context, event eventbus.DomainEvent) error {
	h.logger.Info("收到战斗结束事件，触发战报归档",
		zap.String("event_id", event.EventID),
		zap.Int64("aggregate_id", event.AggregateID),
	)
	// TODO 后续调用archiveColdDataHandler执行战报归档
	return nil
}

// handleSeasonEnded 处理赛季结束事件，触发赛季快照。
func (h *Handler) handleSeasonEnded(ctx context.Context, event eventbus.DomainEvent) error {
	h.logger.Info("收到赛季结束事件，触发赛季快照",
		zap.String("event_id", event.EventID),
		zap.Int64("aggregate_id", event.AggregateID),
	)
	// TODO 后续调用createSnapshotHandler执行赛季快照
	return nil
}
