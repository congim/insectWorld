// Package wallet 玩家钱包聚合根，维护各资源余额的一致性边界。
// 本文件定义PlayerWallet聚合根的单元测试。
package wallet

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPlayerWallet 测试钱包创建。
func TestNewPlayerWallet(t *testing.T) {
	w := NewPlayerWallet(1)
	assert.Equal(t, int64(1), w.PlayerID())
	assert.Equal(t, int64(0), w.GetBalance(100))
}

// TestPlayerWallet_Produce 测试资源产出。
func TestPlayerWallet_Produce(t *testing.T) {
	w := NewPlayerWallet(1)

	amount, err := w.Produce(context.Background(), 100, 500, nil, OverflowDiscard, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(500), amount)
	assert.Equal(t, int64(500), w.GetBalance(100))
}

// TestPlayerWallet_Produce_WithModifiers 测试带修正器的资源产出。
func TestPlayerWallet_Produce_WithModifiers(t *testing.T) {
	w := NewPlayerWallet(1)

	// 科技加成20%，联盟加成10%
	modifiers := []float64{0.2, 0.1}
	amount, err := w.Produce(context.Background(), 100, 1000, modifiers, OverflowDiscard, 0)
	require.NoError(t, err)
	// 1000 * 1.2 = 1200, 1200 * 1.1 = 1320
	assert.Equal(t, int64(1320), amount)
	assert.Equal(t, int64(1320), w.GetBalance(100))
}

// TestPlayerWallet_Produce_OverflowDiscard 测试溢出丢弃处理。
func TestPlayerWallet_Produce_OverflowDiscard(t *testing.T) {
	w := NewPlayerWallet(1)
	w.SetCapacity(100, 1000)

	// 当前0，产出1500，容量1000，溢出丢弃→总额=1000
	_, err := w.Produce(context.Background(), 100, 1500, nil, OverflowDiscard, 1000)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), w.GetBalance(100))
}

// TestPlayerWallet_Produce_OverflowStopProduction 测试溢出停止生产处理。
func TestPlayerWallet_Produce_OverflowStopProduction(t *testing.T) {
	w := NewPlayerWallet(1)
	w.AddBalance(100, 800)

	// 当前800，产出500，容量1000，溢出停止生产→总额=800（不变）
	_, err := w.Produce(context.Background(), 100, 500, nil, OverflowStopProduction, 1000)
	require.NoError(t, err)
	assert.Equal(t, int64(800), w.GetBalance(100))
}

// TestPlayerWallet_Produce_NoOverflow 测试无溢出时正常产出。
func TestPlayerWallet_Produce_NoOverflow(t *testing.T) {
	w := NewPlayerWallet(1)

	_, err := w.Produce(context.Background(), 100, 500, nil, OverflowDiscard, 1000)
	require.NoError(t, err)
	assert.Equal(t, int64(500), w.GetBalance(100))
}

// TestPlayerWallet_Consume 测试资源消耗。
func TestPlayerWallet_Consume(t *testing.T) {
	w := NewPlayerWallet(1)
	w.AddBalance(100, 1000)

	err := w.Consume(context.Background(), 100, 300)
	require.NoError(t, err)
	assert.Equal(t, int64(700), w.GetBalance(100))
}

// TestPlayerWallet_Consume_Insufficient 测试余额不足时消耗失败。
func TestPlayerWallet_Consume_Insufficient(t *testing.T) {
	w := NewPlayerWallet(1)
	w.AddBalance(100, 100)

	err := w.Consume(context.Background(), 100, 500)
	assert.Error(t, err)
}

// TestPlayerWallet_CheckSufficient 测试资源充足校验。
func TestPlayerWallet_CheckSufficient(t *testing.T) {
	w := NewPlayerWallet(1)
	w.AddBalance(100, 500)
	w.AddBalance(200, 300)

	assert.True(t, w.CheckSufficient(map[int64]int64{100: 300, 200: 200}))
	assert.False(t, w.CheckSufficient(map[int64]int64{100: 600}))
	assert.False(t, w.CheckSufficient(map[int64]int64{100: 300, 200: 400}))
}

// TestPlayerWallet_SetCapacity 测试设置容量上限。
func TestPlayerWallet_SetCapacity(t *testing.T) {
	w := NewPlayerWallet(1)
	w.SetCapacity(100, 5000)
	assert.Equal(t, int64(5000), w.GetCapacity(100))
}
