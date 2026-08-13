// Package resource 定义成长上下文访问 Economy 资源账户的防腐层契约。
package resource

import "context"

// Account 是资源账户端口；operationID用于跨重试保证一次资源变更只执行一次。
type Account interface {
	Change(ctx context.Context, playerID int64, amounts map[string]int64, operationID string) error
	Reverse(ctx context.Context, operationID string) error
	Balances(ctx context.Context, playerID int64) (map[string]int64, error)
}
