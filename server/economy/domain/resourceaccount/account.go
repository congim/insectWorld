// Package resourceaccount 定义稳定字符串资源ID账户及其事务仓储契约。
package resourceaccount

import "context"

// OperationStatus 是资源操作状态枚举。
type OperationStatus int32

// 资源操作状态常量。
const (
	OperationStatusApplied  OperationStatus = 1 // 已应用
	OperationStatusReversed OperationStatus = 2 // 已撤销
)

// Change 是一次幂等资源余额变更。
type Change struct {
	OperationID string           // OperationID 是调用方生成的全局幂等键
	PlayerID    int64            // PlayerID 是资源账户所属玩家ID
	Amounts     map[string]int64 // Amounts 是变更量，正数增加、负数扣减
	CreatedAt   int64            // CreatedAt 是首次操作时间戳，Unix毫秒
}

// Repository 在本地事务内维护资源余额和操作账本。
type Repository interface {
	Apply(ctx context.Context, change Change) error
	Reverse(ctx context.Context, operationID string, reversedAt int64) error
	Balances(ctx context.Context, playerID int64) (map[string]int64, error)
}
