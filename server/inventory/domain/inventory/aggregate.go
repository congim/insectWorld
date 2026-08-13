// Package inventory 背包聚合根，维护玩家持有道具的一致性边界。
// Inventory聚合根提供道具获取/使用/出售/过期能力，处理堆叠规则与容量上限。
// 对应spec.md 5.1.7.2节Inventory上下文，按玩家ID分片。
package inventory

import (
	"context"
	"fmt"

	inventoryerr "insectworld/server/inventory/domain/errors"
	"insectworld/server/inventory/domain/vo"
)

// 背包状态常量（规范1）。
const (
	StatusNormal = 1 // 正常
	StatusLocked = 2 // 锁定（如合服迁移中）
)

// Inventory 背包聚合根，维护玩家持有道具的一致性边界。
// 一个玩家对应一个Inventory聚合根，聚合根内道具增减强一致。
type Inventory struct {
	playerID  int64                        // 玩家ID，背包归属者
	capacity  int64                        // 背包容量上限，从inventory.json配置驱动
	usedSlots int64                        // 已用槽位数，不可堆叠道具每占一个槽位
	items     map[vo.ItemID]vo.Item        // 道具映射，key=道具实例ID
	defIndex  map[vo.ItemDefID][]vo.ItemID // 道具定义ID到实例ID列表的索引，加速按定义查询
	status    int                          // 背包状态：1=正常 2=锁定
	updatedAt int64                        // 最后更新时间戳（毫秒）
}

// NewInventory 创建背包聚合根实例。
// capacity为背包容量上限，从inventory.json配置查询注入。
func NewInventory(playerID int64, capacity int64) *Inventory {
	return &Inventory{
		playerID: playerID,
		capacity: capacity,
		items:    make(map[vo.ItemID]vo.Item),
		defIndex: make(map[vo.ItemDefID][]vo.ItemID),
		status:   StatusNormal,
	}
}

// PlayerID 返回玩家ID。
func (inv *Inventory) PlayerID() int64 { return inv.playerID }

// Capacity 返回背包容量上限。
func (inv *Inventory) Capacity() int64 { return inv.capacity }

// UsedSlots 返回已用槽位数。
func (inv *Inventory) UsedSlots() int64 { return inv.usedSlots }

// GetItem 查询指定实例ID的道具。
func (inv *Inventory) GetItem(itemID vo.ItemID) (vo.Item, bool) {
	item, ok := inv.items[itemID]
	return item, ok
}

// GetItemsByDef 查询指定定义ID的所有道具实例。
func (inv *Inventory) GetItemsByDef(defID vo.ItemDefID) []vo.Item {
	ids := inv.defIndex[defID]
	result := make([]vo.Item, 0, len(ids))
	for _, id := range ids {
		if item, ok := inv.items[id]; ok {
			result = append(result, item)
		}
	}
	return result
}

// AddItem 道具获取，处理堆叠规则与容量上限。
// item为待添加道具，stackPolicy为堆叠策略（从items.json配置查询）。
// 返回道具获取事件，含实际新增的道具实例ID与数量。
func (inv *Inventory) AddItem(ctx context.Context, item vo.Item, stackPolicy vo.StackPolicy) (*ItemAddedEvent, error) {
	if inv.status == StatusLocked {
		return nil, fmt.Errorf("背包已锁定，playerID=%d: %w", inv.playerID, inventoryerr.ErrInventoryFull)
	}

	// 可堆叠道具尝试合并到已有堆叠
	if stackPolicy.Stackable {
		for _, id := range inv.defIndex[item.DefID] {
			existing := inv.items[id]
			if existing.ExpireTime != item.ExpireTime {
				continue // 过期时间不同不堆叠
			}
			if existing.Count+item.Count > stackPolicy.MaxStack {
				continue // 超出堆叠上限不合并
			}
			existing.Count += item.Count
			inv.items[id] = existing
			inv.updatedAt = item.ObtainTime
			return &ItemAddedEvent{
				PlayerID:   inv.playerID,
				ItemID:     id,
				DefID:      item.DefID,
				AddedCount: item.Count,
			}, nil
		}
	}

	// 不可堆叠或堆叠失败，新增槽位
	if inv.usedSlots >= inv.capacity {
		return nil, fmt.Errorf("背包已满，playerID=%d，used=%d，cap=%d: %w",
			inv.playerID, inv.usedSlots, inv.capacity, inventoryerr.ErrInventoryFull)
	}

	inv.items[item.ItemID] = item
	inv.defIndex[item.DefID] = append(inv.defIndex[item.DefID], item.ItemID)
	if !stackPolicy.Stackable {
		inv.usedSlots++
	} else {
		inv.usedSlots++ // 可堆叠新增堆也占一个槽位
	}
	inv.updatedAt = item.ObtainTime

	return &ItemAddedEvent{
		PlayerID:   inv.playerID,
		ItemID:     item.ItemID,
		DefID:      item.DefID,
		AddedCount: item.Count,
	}, nil
}

// UseItem 道具使用，校验道具存在、数量足够、未过期，消耗道具。
// itemID为道具实例ID，count为使用数量，now为当前时间戳（毫秒）。
// 返回道具使用事件，application层据此执行道具效果。
func (inv *Inventory) UseItem(ctx context.Context, itemID vo.ItemID, count int64, now int64) (*ItemUsedEvent, error) {
	item, ok := inv.items[itemID]
	if !ok {
		return nil, fmt.Errorf("道具不存在，itemID=%d: %w", itemID, inventoryerr.ErrItemNotFound)
	}

	if item.IsExpired(now) {
		return nil, fmt.Errorf("道具已过期，itemID=%d: %w", itemID, inventoryerr.ErrItemExpired)
	}

	if item.Count < count {
		return nil, fmt.Errorf("道具数量不足，itemID=%d，当前=%d，需要=%d: %w",
			itemID, item.Count, count, inventoryerr.ErrItemInsufficient)
	}

	item.Count -= count
	if item.Count == 0 {
		delete(inv.items, itemID)
		inv.removeFromDefIndex(item.DefID, itemID)
		inv.usedSlots--
	} else {
		inv.items[itemID] = item
	}
	inv.updatedAt = now

	return &ItemUsedEvent{
		PlayerID:  inv.playerID,
		ItemID:    itemID,
		DefID:     item.DefID,
		UsedCount: count,
	}, nil
}

// RemoveExpired 清理过期道具，返回被清理的道具列表。
func (inv *Inventory) RemoveExpired(now int64) []vo.Item {
	expired := make([]vo.Item, 0)
	for id, item := range inv.items {
		if item.IsExpired(now) {
			expired = append(expired, item)
			delete(inv.items, id)
			inv.removeFromDefIndex(item.DefID, id)
			inv.usedSlots--
		}
	}
	if len(expired) > 0 {
		inv.updatedAt = now
	}
	return expired
}

// removeFromDefIndex 从定义索引中移除指定实例ID。
func (inv *Inventory) removeFromDefIndex(defID vo.ItemDefID, itemID vo.ItemID) {
	ids := inv.defIndex[defID]
	for i, id := range ids {
		if id == itemID {
			inv.defIndex[defID] = append(ids[:i], ids[i+1:]...)
			return
		}
	}
}

// ItemAddedEvent 道具获取事件，聚合根状态变更后产生。
type ItemAddedEvent struct {
	PlayerID   int64        // 玩家ID
	ItemID     vo.ItemID    // 道具实例ID
	DefID      vo.ItemDefID // 道具定义ID
	AddedCount int64        // 实际新增数量
}

// ItemUsedEvent 道具使用事件，聚合根状态变更后产生。
type ItemUsedEvent struct {
	PlayerID  int64        // 玩家ID
	ItemID    vo.ItemID    // 道具实例ID
	DefID     vo.ItemDefID // 道具定义ID
	UsedCount int64        // 使用数量
}

// InventoryRepository 背包聚合根仓储接口，在domain层声明（规范3）。
type InventoryRepository interface {
	// LoadInventory 按玩家ID加载背包聚合根
	LoadInventory(ctx context.Context, playerID int64) (*Inventory, error)
	// SaveInventory 保存背包聚合根
	SaveInventory(ctx context.Context, inv *Inventory) error
}
