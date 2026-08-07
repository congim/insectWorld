// Package wallet 玩家钱包聚合根，维护各资源余额的一致性边界。
// 本文件定义ConversionAggregate与TradeOrder的单元测试。
package wallet

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConversion_Convert 测试资源转换成功。
func TestConversion_Convert(t *testing.T) {
	w := NewPlayerWallet(1)
	w.AddBalance(100, 1000)

	conv := NewConversion(1, 1, "rule_1", map[int64]int64{100: 500}, 1000)
	err := conv.Convert(context.Background(), w, 2.0)
	require.NoError(t, err)

	// 输入500从余额1000扣减→500，转换比例2.0产出1000加回→1500
	assert.Equal(t, int64(1500), w.GetBalance(100))
}

// TestConversion_Convert_NotPending 测试非待执行状态转换失败。
func TestConversion_Convert_NotPending(t *testing.T) {
	w := NewPlayerWallet(1)
	w.AddBalance(100, 1000)

	conv := NewConversion(1, 1, "rule_1", map[int64]int64{100: 500}, 1000)
	err := conv.Convert(context.Background(), w, 2.0)
	require.NoError(t, err)

	// 再次执行应失败
	err = conv.Convert(context.Background(), w, 2.0)
	assert.Error(t, err)
}

// TestConversion_Convert_Insufficient 测试输入资源不足时转换失败。
func TestConversion_Convert_Insufficient(t *testing.T) {
	w := NewPlayerWallet(1)
	w.AddBalance(100, 100)

	conv := NewConversion(1, 1, "rule_1", map[int64]int64{100: 500}, 1000)
	err := conv.Convert(context.Background(), w, 2.0)
	assert.Error(t, err)
}

// TestTradeOrder_Execute 测试交易执行成功。
func TestTradeOrder_Execute(t *testing.T) {
	fromWallet := NewPlayerWallet(1)
	fromWallet.AddBalance(100, 1000)
	toWallet := NewPlayerWallet(2)

	order := NewTradeOrder(1, TradeTypePlayerPlayer, 1, 2, map[int64]int64{100: 300}, 1000)
	err := order.Execute(context.Background(), fromWallet, toWallet)
	require.NoError(t, err)

	assert.Equal(t, int64(700), fromWallet.GetBalance(100))
	assert.Equal(t, int64(300), toWallet.GetBalance(100))
}

// TestTradeOrder_Execute_NotPending 测试非待确认状态交易失败。
func TestTradeOrder_Execute_NotPending(t *testing.T) {
	fromWallet := NewPlayerWallet(1)
	fromWallet.AddBalance(100, 1000)
	toWallet := NewPlayerWallet(2)

	order := NewTradeOrder(1, TradeTypePlayerPlayer, 1, 2, map[int64]int64{100: 300}, 1000)
	err := order.Execute(context.Background(), fromWallet, toWallet)
	require.NoError(t, err)

	// 再次执行应失败
	err = order.Execute(context.Background(), fromWallet, toWallet)
	assert.Error(t, err)
}

// TestTradeOrder_Execute_Insufficient 测试卖出方资源不足时交易失败。
func TestTradeOrder_Execute_Insufficient(t *testing.T) {
	fromWallet := NewPlayerWallet(1)
	fromWallet.AddBalance(100, 100)
	toWallet := NewPlayerWallet(2)

	order := NewTradeOrder(1, TradeTypePlayerPlayer, 1, 2, map[int64]int64{100: 500}, 1000)
	err := order.Execute(context.Background(), fromWallet, toWallet)
	assert.Error(t, err)
}

// TestTradeOrder_SetAlliance 测试设置联盟间交易联盟ID。
func TestTradeOrder_SetAlliance(t *testing.T) {
	order := NewTradeOrder(1, TradeTypeAllianceAlliance, 1, 2, map[int64]int64{100: 300}, 1000)
	order.SetAlliance(10, 20)
	// 验证不panic即可，联盟ID为私有字段
}
