// Package crossserveractivity 跨服活动聚合根单元测试。
package crossserveractivity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCrossServerActivity 测试跨服活动创建。
func TestNewCrossServerActivity(t *testing.T) {
	a := NewCrossServerActivity(1, TypeCrossWar, []int64{100, 200}, 1, 1000, 5000)
	assert.Equal(t, int64(1), a.ActivityID())
	assert.Equal(t, StatusCreated, a.Status())
}

// TestCrossServerActivity_Start 测试开始活动。
func TestCrossServerActivity_Start(t *testing.T) {
	a := NewCrossServerActivity(1, TypeCrossWar, []int64{100, 200}, 1, 1000, 5000)
	err := a.Start(1000)
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, a.Status())
}

// TestCrossServerActivity_Start_NotCreated 测试非已创建状态开始失败。
func TestCrossServerActivity_Start_NotCreated(t *testing.T) {
	a := NewCrossServerActivity(1, TypeCrossWar, []int64{100, 200}, 1, 1000, 5000)
	_ = a.Start(1000)
	err := a.Start(2000)
	assert.Error(t, err)
}

// TestCrossServerActivity_Start_BeforeTime 测试未到开始时间。
func TestCrossServerActivity_Start_BeforeTime(t *testing.T) {
	a := NewCrossServerActivity(1, TypeCrossWar, []int64{100, 200}, 1, 1000, 5000)
	err := a.Start(500)
	assert.Error(t, err)
}

// TestCrossServerActivity_Settle 测试活动结算。
func TestCrossServerActivity_Settle(t *testing.T) {
	a := NewCrossServerActivity(1, TypeCrossWar, []int64{100, 200}, 1, 1000, 5000)
	_ = a.Start(1000)

	event, err := a.Settle(6000)
	require.NoError(t, err)
	assert.Equal(t, StatusSettled, a.Status())
	assert.Equal(t, int64(6000), event.SettleTime)
}

// TestCrossServerActivity_Settle_NotRunning 测试非进行中状态结算失败。
func TestCrossServerActivity_Settle_NotRunning(t *testing.T) {
	a := NewCrossServerActivity(1, TypeCrossWar, []int64{100, 200}, 1, 1000, 5000)
	_, err := a.Settle(6000)
	assert.Error(t, err)
}

// TestCrossServerActivity_Close 测试关闭活动。
func TestCrossServerActivity_Close(t *testing.T) {
	a := NewCrossServerActivity(1, TypeCrossWar, []int64{100, 200}, 1, 1000, 5000)
	_ = a.Start(1000)
	_, _ = a.Settle(6000)
	a.Close()
	assert.Equal(t, StatusClosed, a.Status())
}
