// Package query Inventory服务application层读侧查询，CQRS读模型查询handler。
// 对接interfaces/grpc层的查询请求，从读模型查询数据（不经过domain聚合根）。
package query

import (
	"context"

	"go.uber.org/zap"

	"insectworld/server/inventory/domain/inventory"
	"insectworld/server/inventory/domain/vo"
)

// InventoryQuery 背包查询DTO。
type InventoryQuery struct {
	PlayerID int64 // 玩家ID
}

// InventoryResult 背包查询结果DTO。
type InventoryResult struct {
	Capacity  int64     // 背包容量上限
	UsedSlots int64     // 已用槽位数
	Items     []vo.Item // 道具列表
}

// InventoryQueryHandler 背包查询handler，CQRS读侧。
type InventoryQueryHandler struct {
	inventoryRepo inventory.InventoryRepository // 背包仓储接口，用于读侧查询
	logger        *zap.Logger                   // 结构化日志器（规范7）
}

// NewInventoryQueryHandler 创建背包查询handler实例。
// inventoryRepo由infrastructure层实现，cmd/main.go组装时注入。
func NewInventoryQueryHandler(inventoryRepo inventory.InventoryRepository, logger *zap.Logger) *InventoryQueryHandler {
	return &InventoryQueryHandler{inventoryRepo: inventoryRepo, logger: logger}
}

// Handle 处理背包查询。
// 通过Repository加载聚合根并提取查询结果（简化模式，后续可改为读模型投影）。
func (h *InventoryQueryHandler) Handle(ctx context.Context, q InventoryQuery) (*InventoryResult, error) {
	inv, err := h.inventoryRepo.LoadInventory(ctx, q.PlayerID)
	if err != nil {
		return nil, err
	}

	result := &InventoryResult{
		Capacity:  inv.Capacity(),
		UsedSlots: inv.UsedSlots(),
		Items:     make([]vo.Item, 0),
	}

	h.logger.Debug("查询背包",
		zap.Int64("player_id", q.PlayerID),
		zap.Int64("used_slots", result.UsedSlots),
	)
	return result, nil
}
