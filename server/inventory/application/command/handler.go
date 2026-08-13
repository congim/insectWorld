// Package command Inventory服务application层写侧命令，编排聚合根调用、事务边界与领域事件Outbox投递。
// 对应DDD application层，不直接import infrastructure，通过依赖注入的接口操作。
package command

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/inventory/domain/inventory"
	"insectworld/server/inventory/domain/vo"
)

// AddItemCommand 道具获取命令DTO。
type AddItemCommand struct {
	PlayerID   int64        // 玩家ID
	ItemID     vo.ItemID    // 道具实例ID
	DefID      vo.ItemDefID // 道具定义ID
	Count      int64        // 数量
	ExpireTime int64        // 过期时间戳（毫秒），0表示永不过期
	Source     int          // 获取来源
	Now        int64        // 当前时间戳（毫秒）
}

// UseItemCommand 道具使用命令DTO。
type UseItemCommand struct {
	PlayerID int64     // 玩家ID
	ItemID   vo.ItemID // 道具实例ID
	Count    int64     // 使用数量
	Now      int64     // 当前时间戳（毫秒）
}

// AddItemHandler 道具获取命令handler。
type AddItemHandler struct {
	inventoryRepo inventory.InventoryRepository // 背包仓储接口，infrastructure层注入
	logger        *zap.Logger                   // 结构化日志器（规范7）
}

// NewAddItemHandler 创建道具获取命令handler实例。
// inventoryRepo由infrastructure层实现，cmd/main.go组装时注入。
func NewAddItemHandler(inventoryRepo inventory.InventoryRepository, logger *zap.Logger) *AddItemHandler {
	return &AddItemHandler{inventoryRepo: inventoryRepo, logger: logger}
}

// Handle 处理道具获取命令。
// 编排：加载背包聚合根→查询堆叠策略→调用聚合根AddItem→保存+Outbox。
func (h *AddItemHandler) Handle(ctx context.Context, cmd AddItemCommand) (*inventory.ItemAddedEvent, error) {
	// 1. 加载背包聚合根
	inv, err := h.inventoryRepo.LoadInventory(ctx, cmd.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("加载背包失败，playerID=%d: %w", cmd.PlayerID, err)
	}

	// 2. 构造道具值对象
	item := vo.Item{
		ItemID:     cmd.ItemID,
		DefID:      cmd.DefID,
		Count:      cmd.Count,
		ExpireTime: cmd.ExpireTime,
		Source:     cmd.Source,
		ObtainTime: cmd.Now,
	}

	// 3. 查询堆叠策略（TODO 后续接入ConfigQueryAPI查询items.json配置）
	stackPolicy := vo.StackPolicy{Stackable: false, MaxStack: 1}

	// 4. 调用聚合根AddItem
	event, err := inv.AddItem(ctx, item, stackPolicy)
	if err != nil {
		h.logger.Warn("道具获取失败",
			zap.Int64("player_id", cmd.PlayerID),
			zap.Int64("def_id", int64(cmd.DefID)),
			zap.Error(err),
		)
		return nil, err
	}

	// 5. 保存聚合根
	if err := h.inventoryRepo.SaveInventory(ctx, inv); err != nil {
		return nil, fmt.Errorf("保存背包失败，playerID=%d: %w", cmd.PlayerID, err)
	}

	h.logger.Info("道具获取",
		zap.Int64("player_id", cmd.PlayerID),
		zap.Int64("def_id", int64(cmd.DefID)),
		zap.Int64("count", cmd.Count),
	)
	// TODO 后续接入Outbox投递ItemAddedEvent
	return event, nil
}

// UseItemHandler 道具使用命令handler。
type UseItemHandler struct {
	inventoryRepo inventory.InventoryRepository // 背包仓储接口，infrastructure层注入
	logger        *zap.Logger                   // 结构化日志器（规范7）
}

// NewUseItemHandler 创建道具使用命令handler实例。
func NewUseItemHandler(inventoryRepo inventory.InventoryRepository, logger *zap.Logger) *UseItemHandler {
	return &UseItemHandler{inventoryRepo: inventoryRepo, logger: logger}
}

// Handle 处理道具使用命令。
// 编排：加载背包聚合根→调用聚合根UseItem→创建使用订单→执行效果→保存+Outbox。
func (h *UseItemHandler) Handle(ctx context.Context, cmd UseItemCommand) (*inventory.ItemUsedEvent, error) {
	// 1. 加载背包聚合根
	inv, err := h.inventoryRepo.LoadInventory(ctx, cmd.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("加载背包失败，playerID=%d: %w", cmd.PlayerID, err)
	}

	// 2. 调用聚合根UseItem
	event, err := inv.UseItem(ctx, cmd.ItemID, cmd.Count, cmd.Now)
	if err != nil {
		h.logger.Warn("道具使用失败",
			zap.Int64("player_id", cmd.PlayerID),
			zap.Int64("item_id", int64(cmd.ItemID)),
			zap.Error(err),
		)
		return nil, err
	}

	// 3. 保存聚合根
	if err := h.inventoryRepo.SaveInventory(ctx, inv); err != nil {
		return nil, fmt.Errorf("保存背包失败，playerID=%d: %w", cmd.PlayerID, err)
	}

	h.logger.Info("道具使用",
		zap.Int64("player_id", cmd.PlayerID),
		zap.Int64("item_id", int64(cmd.ItemID)),
		zap.Int64("count", cmd.Count),
	)
	// TODO 后续接入道具效果执行器、Outbox投递ItemUsedEvent
	return event, nil
}
