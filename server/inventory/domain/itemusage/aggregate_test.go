// Package itemusage 道具使用订单聚合根单元测试。
package itemusage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"insectworld/server/inventory/domain/vo"
)

// TestNewItemUsage 测试使用订单创建。
func TestNewItemUsage(t *testing.T) {
	u := NewItemUsage(1, 100, 1001, 2001, 3, 1, 1000)
	assert.Equal(t, int64(1), u.UsageID())
	assert.Equal(t, StatusPending, u.Status())
}

// TestItemUsage_StartExecution 测试开始执行。
func TestItemUsage_StartExecution(t *testing.T) {
	u := NewItemUsage(1, 100, 1001, 2001, 3, 1, 1000)
	err := u.StartExecution()
	require.NoError(t, err)
	assert.Equal(t, StatusExecuting, u.Status())
}

// TestItemUsage_StartExecution_AlreadyStarted 测试重复开始执行。
func TestItemUsage_StartExecution_AlreadyStarted(t *testing.T) {
	u := NewItemUsage(1, 100, 1001, 2001, 3, 1, 1000)
	_ = u.StartExecution()
	err := u.StartExecution()
	assert.Error(t, err)
}

// TestItemUsage_Complete 测试完成使用。
func TestItemUsage_Complete(t *testing.T) {
	u := NewItemUsage(1, 100, 1001, 2001, 3, 1, 1000)
	_ = u.StartExecution()
	event := u.Complete(2000)
	assert.Equal(t, int64(1), event.UsageID)
	assert.Equal(t, int64(3), event.Count)
	assert.Equal(t, StatusCompleted, u.Status())
}

// TestItemUsage_Fail 测试使用失败。
func TestItemUsage_Fail(t *testing.T) {
	u := NewItemUsage(1, 100, 1001, 2001, 3, 1, 1000)
	_ = u.StartExecution()
	event := u.Fail("效果执行失败", 2000)
	assert.Equal(t, "效果执行失败", event.ErrorMsg)
	assert.Equal(t, StatusFailed, u.Status())
}

// TestItemUsage_Rollback 测试回滚。
func TestItemUsage_Rollback(t *testing.T) {
	u := NewItemUsage(1, 100, 1001, 2001, 3, 1, 1000)
	_ = u.StartExecution()
	_ = u.Fail("失败", 2000)
	event := u.Rollback()
	assert.Equal(t, vo.ItemID(1001), event.ItemID)
	assert.Equal(t, int64(3), event.Count)
	assert.Equal(t, StatusRolledBack, u.Status())
}
