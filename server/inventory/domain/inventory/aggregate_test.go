// Package inventory 背包聚合根单元测试。
package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"insectworld/server/inventory/domain/vo"
)

// TestNewInventory 测试背包聚合根创建。
func TestNewInventory(t *testing.T) {
	inv := NewInventory(1, 100)
	assert.Equal(t, int64(1), inv.PlayerID())
	assert.Equal(t, int64(100), inv.Capacity())
	assert.Equal(t, int64(0), inv.UsedSlots())
}

// TestInventory_AddItem 测试道具获取。
func TestInventory_AddItem(t *testing.T) {
	inv := NewInventory(1, 100)
	item := vo.Item{
		ItemID:     1001,
		DefID:      2001,
		Count:      1,
		ExpireTime: 0,
		Source:     vo.ItemSourceDrop,
		ObtainTime: 1000,
	}
	policy := vo.StackPolicy{Stackable: false, MaxStack: 1}

	event, err := inv.AddItem(context.Background(), item, policy)
	require.NoError(t, err)
	assert.Equal(t, int64(1), inv.UsedSlots())
	assert.Equal(t, vo.ItemID(1001), event.ItemID)
	assert.Equal(t, int64(1), event.AddedCount)
}

// TestInventory_AddItem_Stackable 测试可堆叠道具获取。
func TestInventory_AddItem_Stackable(t *testing.T) {
	inv := NewInventory(1, 100)
	policy := vo.StackPolicy{Stackable: true, MaxStack: 99}

	item1 := vo.Item{ItemID: 1001, DefID: 2001, Count: 10, ObtainTime: 1000}
	_, err := inv.AddItem(context.Background(), item1, policy)
	require.NoError(t, err)

	item2 := vo.Item{ItemID: 1002, DefID: 2001, Count: 20, ObtainTime: 2000}
	event, err := inv.AddItem(context.Background(), item2, policy)
	require.NoError(t, err)
	assert.Equal(t, vo.ItemID(1001), event.ItemID)
	assert.Equal(t, int64(20), event.AddedCount)
}

// TestInventory_AddItem_Full 测试背包已满。
func TestInventory_AddItem_Full(t *testing.T) {
	inv := NewInventory(1, 1)
	policy := vo.StackPolicy{Stackable: false, MaxStack: 1}

	item1 := vo.Item{ItemID: 1001, DefID: 2001, Count: 1, ObtainTime: 1000}
	_, err := inv.AddItem(context.Background(), item1, policy)
	require.NoError(t, err)

	item2 := vo.Item{ItemID: 1002, DefID: 2002, Count: 1, ObtainTime: 2000}
	_, err = inv.AddItem(context.Background(), item2, policy)
	assert.Error(t, err)
}

// TestInventory_UseItem 测试道具使用。
func TestInventory_UseItem(t *testing.T) {
	inv := NewInventory(1, 100)
	policy := vo.StackPolicy{Stackable: false, MaxStack: 1}

	item := vo.Item{ItemID: 1001, DefID: 2001, Count: 5, ObtainTime: 1000}
	_, err := inv.AddItem(context.Background(), item, policy)
	require.NoError(t, err)

	event, err := inv.UseItem(context.Background(), 1001, 2, 2000)
	require.NoError(t, err)
	assert.Equal(t, int64(2), event.UsedCount)

	got, ok := inv.GetItem(1001)
	require.True(t, ok)
	assert.Equal(t, int64(3), got.Count)
}

// TestInventory_UseItem_NotFound 测试道具不存在。
func TestInventory_UseItem_NotFound(t *testing.T) {
	inv := NewInventory(1, 100)
	_, err := inv.UseItem(context.Background(), 9999, 1, 1000)
	assert.Error(t, err)
}

// TestInventory_UseItem_Insufficient 测试道具数量不足。
func TestInventory_UseItem_Insufficient(t *testing.T) {
	inv := NewInventory(1, 100)
	policy := vo.StackPolicy{Stackable: false, MaxStack: 1}
	item := vo.Item{ItemID: 1001, DefID: 2001, Count: 1, ObtainTime: 1000}
	_, err := inv.AddItem(context.Background(), item, policy)
	require.NoError(t, err)

	_, err = inv.UseItem(context.Background(), 1001, 5, 2000)
	assert.Error(t, err)
}

// TestInventory_UseItem_Expired 测试过期道具使用。
func TestInventory_UseItem_Expired(t *testing.T) {
	inv := NewInventory(1, 100)
	policy := vo.StackPolicy{Stackable: false, MaxStack: 1}
	item := vo.Item{ItemID: 1001, DefID: 2001, Count: 1, ExpireTime: 1500, ObtainTime: 1000}
	_, err := inv.AddItem(context.Background(), item, policy)
	require.NoError(t, err)

	_, err = inv.UseItem(context.Background(), 1001, 1, 2000)
	assert.Error(t, err)
}

// TestInventory_GetItemsByDef 测试按定义ID查询道具。
func TestInventory_GetItemsByDef(t *testing.T) {
	inv := NewInventory(1, 100)
	policy := vo.StackPolicy{Stackable: false, MaxStack: 1}

	item1 := vo.Item{ItemID: 1001, DefID: 2001, Count: 1, ObtainTime: 1000}
	item2 := vo.Item{ItemID: 1002, DefID: 2001, Count: 1, ObtainTime: 1000}
	item3 := vo.Item{ItemID: 1003, DefID: 2002, Count: 1, ObtainTime: 1000}
	_, _ = inv.AddItem(context.Background(), item1, policy)
	_, _ = inv.AddItem(context.Background(), item2, policy)
	_, _ = inv.AddItem(context.Background(), item3, policy)

	items := inv.GetItemsByDef(2001)
	assert.Equal(t, 2, len(items))
}
