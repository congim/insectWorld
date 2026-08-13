// Package memory 提供Growth纵切本地运行与测试使用的并发安全内存适配器。
package memory

import (
	"context"
	"fmt"
	"maps"
	"math"
	"sync"
	"sync/atomic"

	gameerr "insectworld/server/game/domain/errors"
)

// IDGenerator 是基于原子计数器的进程内ID生成器，仅用于本地运行和测试。
type IDGenerator struct {
	value atomic.Int64 // 当前已分配的最大ID
}

// NewIDGenerator 创建从startExclusive之后开始分配的ID生成器。
func NewIDGenerator(startExclusive int64) *IDGenerator {
	generator := &IDGenerator{}
	generator.value.Store(startExclusive)
	return generator
}

// Next 返回下一个大于0的实例ID。
func (g *IDGenerator) Next() int64 { return g.value.Add(1) }

type operation struct {
	playerID int64            // 资源变更所属玩家ID
	amounts  map[string]int64 // 资源变更量，正数增加、负数扣减
	applied  bool             // 变更当前是否已应用
}

// ResourceAccount 是并发安全且支持操作幂等与补偿的内存资源账户适配器。
type ResourceAccount struct {
	mu         sync.RWMutex               // 保护余额和操作账本
	balances   map[int64]map[string]int64 // 玩家资源余额，第一层key为玩家ID
	operations map[string]*operation      // operationID到资源变更记录
}

// UnitRoster 是并发安全且按operationID幂等入账的内存单位名册。
type UnitRoster struct {
	mu         sync.RWMutex               // 保护单位数量与操作账本
	counts     map[int64]map[string]int64 // 玩家单位数量，第一层key为玩家ID
	operations map[string]rosterOperation // operationID到发放记录
}

type rosterOperation struct {
	playerID   int64  // 获得单位的玩家ID
	unitTypeID string // 获得的单位类型稳定ID
	count      int64  // 发放数量
}

// NewUnitRoster 创建空的内存单位名册。
func NewUnitRoster() *UnitRoster {
	return &UnitRoster{counts: make(map[int64]map[string]int64), operations: make(map[string]rosterOperation)}
}

// Grant 幂等增加玩家已训练单位数量。
func (r *UnitRoster) Grant(_ context.Context, playerID int64, unitTypeID string, count int64, operationID string) error {
	if playerID <= 0 || unitTypeID == "" || count <= 0 || operationID == "" {
		return fmt.Errorf("单位发放参数非法，playerID=%d: %w", playerID, gameerr.ErrInvalidCommand)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.operations[operationID]; ok {
		if existing.playerID != playerID || existing.unitTypeID != unitTypeID || existing.count != count {
			return fmt.Errorf("单位发放幂等键参数冲突，operationID=%s: %w", operationID, gameerr.ErrStateConflict)
		}
		return nil
	}
	current := r.counts[playerID]
	if current == nil {
		current = make(map[string]int64)
	}
	if current[unitTypeID] > math.MaxInt64-count {
		return fmt.Errorf("单位数量溢出，unitTypeID=%s: %w", unitTypeID, gameerr.ErrStateConflict)
	}
	current[unitTypeID] += count
	r.counts[playerID] = current
	r.operations[operationID] = rosterOperation{playerID: playerID, unitTypeID: unitTypeID, count: count}
	return nil
}

// Count 返回玩家指定类型的已训练单位数量。
func (r *UnitRoster) Count(_ context.Context, playerID int64, unitTypeID string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.counts[playerID][unitTypeID], nil
}

// NewResourceAccount 创建空的内存资源账户。
func NewResourceAccount() *ResourceAccount {
	return &ResourceAccount{balances: make(map[int64]map[string]int64), operations: make(map[string]*operation)}
}

// Change 幂等应用一次资源变更；同一operationID携带不同参数会被拒绝。
func (a *ResourceAccount) Change(_ context.Context, playerID int64, amounts map[string]int64, operationID string) error {
	if playerID <= 0 || operationID == "" || len(amounts) == 0 {
		return fmt.Errorf("资源变更参数非法，playerID=%d: %w", playerID, gameerr.ErrInvalidCommand)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if existing, ok := a.operations[operationID]; ok {
		if existing.playerID != playerID || !maps.Equal(existing.amounts, amounts) {
			return fmt.Errorf("资源操作幂等键参数冲突，operationID=%s: %w", operationID, gameerr.ErrStateConflict)
		}
		if existing.applied {
			return nil
		}
		if err := a.apply(playerID, amounts); err != nil {
			return err
		}
		existing.applied = true
		return nil
	}
	if err := a.apply(playerID, amounts); err != nil {
		return err
	}
	a.operations[operationID] = &operation{playerID: playerID, amounts: maps.Clone(amounts), applied: true}
	return nil
}

// Reverse 幂等撤销已记录的资源操作，供应用层失败补偿。
func (a *ResourceAccount) Reverse(_ context.Context, operationID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	value, ok := a.operations[operationID]
	if !ok || !value.applied {
		return nil
	}
	inverse := make(map[string]int64, len(value.amounts))
	for resourceID, amount := range value.amounts {
		inverse[resourceID] = -amount
	}
	if err := a.apply(value.playerID, inverse); err != nil {
		return fmt.Errorf("撤销资源操作失败，operationID=%s: %w", operationID, err)
	}
	value.applied = false
	return nil
}

// Balances 返回玩家当前资源余额的防御性副本。
func (a *ResourceAccount) Balances(_ context.Context, playerID int64) (map[string]int64, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return maps.Clone(a.balances[playerID]), nil
}

func (a *ResourceAccount) apply(playerID int64, amounts map[string]int64) error {
	current := a.balances[playerID]
	if current == nil {
		current = make(map[string]int64)
	}
	for resourceID, amount := range amounts {
		if resourceID == "" || amount == 0 {
			return fmt.Errorf("资源变更项非法，resourceID=%s，amount=%d: %w", resourceID, amount, gameerr.ErrInvalidCommand)
		}
		balance := current[resourceID]
		if amount < 0 && balance < -amount {
			return fmt.Errorf("资源余额不足，resourceID=%s，balance=%d，required=%d: %w", resourceID, balance, -amount, gameerr.ErrResourceInsufficient)
		}
		if amount > 0 && balance > math.MaxInt64-amount {
			return fmt.Errorf("资源余额溢出，resourceID=%s: %w", resourceID, gameerr.ErrStateConflict)
		}
	}
	for resourceID, amount := range amounts {
		current[resourceID] += amount
	}
	a.balances[playerID] = current
	return nil
}
