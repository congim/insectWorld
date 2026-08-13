// Package account 玩家账号聚合根与凭证值对象，维护账号档案的一致性边界。
package account

import (
	"context"

	"insectworld/server/shared/pkg/eventbus"
)

// AccountRepository 账号仓储接口，domain层声明，infrastructure层实现MySQL适配（规范3 DDD）。
//
// 仓储接口保证domain层零外部依赖，application层通过此接口操作账号聚合根。
// 方法第一个参数为context.Context（规范9），支持超时与链路追踪。
type AccountRepository interface {
	// Save 保存账号聚合根，INSERT或UPDATE到t_player_account表。
	// 存储故障返回ErrAccountRepoUnavailable包裹底层error。
	Save(ctx context.Context, account *PlayerAccount) error

	// FindByID 按玩家ID查询账号档案。
	// 账号不存在返回ErrAccountNotFoundSentinel（可用errors.Is判断）。
	FindByID(ctx context.Context, playerID int64) (*PlayerAccount, error)

	// FindByUsername 按用户名查询账号档案。
	// 账号不存在返回ErrAccountNotFoundSentinel。
	FindByUsername(ctx context.Context, username string) (*PlayerAccount, error)

	// ExistsByUsername 判断用户名是否已存在，返回true表示已占用。
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}

// RegistrationRepository 原子保存新账号与玩家注册Outbox事件。
type RegistrationRepository interface {
	AccountRepository
	SaveRegistered(ctx context.Context, account *PlayerAccount, event eventbus.DomainEvent) error
}
