// Package wallet 玩家钱包聚合根，维护各资源余额的一致性边界。
// PlayerWallet提供资源产出（含修正器应用）与消耗能力，处理存储溢出。
package wallet

import (
	"context"
	"fmt"

	economyerr "insectworld/server/economy/domain/errors"
)

// 溢出处理方式常量（规范1）。
const (
	OverflowDiscard        = 1 // 丢弃溢出
	OverflowStopProduction = 2 // 停止生产
	OverflowConvert        = 3 // 转换为其他资源
)

// PlayerWallet 玩家钱包聚合根，维护各资源余额的一致性边界。
type PlayerWallet struct {
	playerID   int64           // 玩家ID
	balances   map[int64]int64 // 资源余额映射，key=资源类型ID，value=余额数量（int64，资源数量用整型）
	capacities map[int64]int64 // 资源容量上限映射，key=资源类型ID，value=容量上限
	updatedAt  int64           // 余额最后更新时间戳（毫秒）
}

// NewPlayerWallet 创建玩家钱包聚合根实例。
func NewPlayerWallet(playerID int64) *PlayerWallet {
	return &PlayerWallet{
		playerID:   playerID,
		balances:   make(map[int64]int64),
		capacities: make(map[int64]int64),
	}
}

// PlayerID 返回玩家ID。
func (w *PlayerWallet) PlayerID() int64 { return w.playerID }

// GetBalance 查询资源余额。
func (w *PlayerWallet) GetBalance(resourceType int64) int64 {
	return w.balances[resourceType]
}

// GetCapacity 查询资源容量上限。
func (w *PlayerWallet) GetCapacity(resourceType int64) int64 {
	return w.capacities[resourceType]
}

// SetCapacity 设置资源容量上限。
func (w *PlayerWallet) SetCapacity(resourceType, capacity int64) {
	w.capacities[resourceType] = capacity
}

// CheckSufficient 校验资源是否充足。
func (w *PlayerWallet) CheckSufficient(required map[int64]int64) bool {
	for resType, amount := range required {
		if w.balances[resType] < amount {
			return false
		}
	}
	return true
}

// Produce 产出资源，应用修正器并处理溢出。
// baseAmount为基础产出量，modifiers为修正器加成比例（科技→联盟→赛季优先级链式应用）。
// overflowBehavior为溢出处理方式，capacity为容量上限。
func (w *PlayerWallet) Produce(ctx context.Context, resourceType int64, baseAmount int64, modifiers []float64, overflowBehavior int, capacity int64) (int64, error) {
	// 应用修正器（按优先级链式应用：科技加成→联盟加成→赛季加成）
	amount := baseAmount
	for _, modifier := range modifiers {
		amount = int64(float64(amount) * (1 + modifier))
	}

	// 处理溢出
	current := w.balances[resourceType]
	total := current + amount
	if capacity > 0 && total > capacity {
		switch overflowBehavior {
		case OverflowDiscard:
			total = capacity
		case OverflowStopProduction:
			total = current
		case OverflowConvert:
			// 溢出转换通过领域事件通知application层编排：
			// application层订阅溢出事件后查询config获取转换规则，调用Convert方法转换。
			// 当前domain层不直接依赖config，按丢弃处理，溢出量通过事件通知application层。
			total = capacity
		}
	}

	w.balances[resourceType] = total
	return amount, nil
}

// Consume 消耗资源，校验余额充足后扣减。
func (w *PlayerWallet) Consume(ctx context.Context, resourceType, amount int64) error {
	if w.balances[resourceType] < amount {
		return fmt.Errorf("资源消耗失败，余额不足，playerID=%d，resourceType=%d，余额=%d，需要=%d: %w",
			w.playerID, resourceType, w.balances[resourceType], amount, economyerr.ErrResourceInsufficient)
	}
	w.balances[resourceType] -= amount
	return nil
}

// AddBalance 增加余额（用于交易/转换等场景，不经过修正器）。
func (w *PlayerWallet) AddBalance(resourceType, amount int64) {
	w.balances[resourceType] += amount
}

// WalletRepository PlayerWallet仓储接口，在domain层声明（规范3）。
type WalletRepository interface {
	LoadWallet(ctx context.Context, playerID int64) (*PlayerWallet, error)
	SaveWallet(ctx context.Context, w *PlayerWallet) error
}
