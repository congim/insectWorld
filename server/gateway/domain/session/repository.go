// Package session 在线会话聚合根，维护玩家在线会话的一致性边界。
package session

import "context"

// SessionRepository 会话仓储接口，domain层声明，infrastructure层实现Redis与内存适配（规范3 DDD）。
//
// 仓储接口保证domain层零外部依赖，application层通过此接口操作会话聚合根。
// 方法第一个参数为context.Context（规范9），支持超时与链路追踪。
type SessionRepository interface {
	// Save 保存会话聚合根，写入Redis或内存map。
	// 存储故障返回ErrSessionRepoUnavailable包裹底层error。
	Save(ctx context.Context, sess *OnlineSession) error

	// FindByPlayerID 按玩家ID查询在线会话。
	// 会话不存在返回ErrSessionNotFound（可用errors.Is判断）。
	FindByPlayerID(ctx context.Context, playerID int64) (*OnlineSession, error)

	// Delete 删除会话，玩家登出或踢下线时调用。
	Delete(ctx context.Context, playerID int64) error

	// FindExpired 查询超时会话，供SessionTimeoutCleaner周期清理。
	// thresholdTime为过期阈值时间戳，返回heartbeatTime < thresholdTime的会话。
	// limit控制单次返回数，避免单次拉取过多。
	FindExpired(ctx context.Context, thresholdTime int64, limit int) ([]*OnlineSession, error)
}
