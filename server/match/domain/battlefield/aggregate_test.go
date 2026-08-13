// Package battlefield 限时战场聚合根单元测试。
package battlefield

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBattlefield 测试战场创建。
func TestNewBattlefield(t *testing.T) {
	b := NewBattlefield(1, 100, 200, 10, 1000)
	assert.Equal(t, int64(1), b.BattlefieldID())
	assert.Equal(t, StatusPreparing, b.Status())
}

// TestBattlefield_AddParticipant 测试添加参与方。
func TestBattlefield_AddParticipant(t *testing.T) {
	b := NewBattlefield(1, 100, 200, 10, 1000)
	err := b.AddParticipant(1001)
	require.NoError(t, err)
	err = b.AddParticipant(1002)
	require.NoError(t, err)
}

// TestBattlefield_AddParticipant_Full 测试战场已满。
func TestBattlefield_AddParticipant_Full(t *testing.T) {
	b := NewBattlefield(1, 100, 200, 1, 1000)
	_ = b.AddParticipant(1001)
	err := b.AddParticipant(1002)
	assert.Error(t, err)
}

// TestBattlefield_Start 测试开始战场。
func TestBattlefield_Start(t *testing.T) {
	b := NewBattlefield(1, 100, 200, 10, 1000)
	err := b.Start()
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, b.Status())
}

// TestBattlefield_Settle 测试战场结算。
func TestBattlefield_Settle(t *testing.T) {
	b := NewBattlefield(1, 100, 200, 10, 1000)
	_ = b.Start()

	settlement := &Settlement{
		WinnerIDs: []int64{1001},
		LoserIDs:  []int64{1002},
	}
	event, err := b.Settle(settlement, 3000)
	require.NoError(t, err)
	assert.Equal(t, StatusSettled, b.Status())
	assert.Equal(t, int64(3000), event.EndTime)
}

// TestBattlefield_Settle_NotRunning 测试非进行中状态结算失败。
func TestBattlefield_Settle_NotRunning(t *testing.T) {
	b := NewBattlefield(1, 100, 200, 10, 1000)
	_, err := b.Settle(&Settlement{}, 3000)
	assert.Error(t, err)
}

// TestBattlefield_Close 测试关闭战场。
func TestBattlefield_Close(t *testing.T) {
	b := NewBattlefield(1, 100, 200, 10, 1000)
	_ = b.Start()
	_, _ = b.Settle(&Settlement{}, 3000)
	b.Close()
	assert.Equal(t, StatusClosed, b.Status())
}
