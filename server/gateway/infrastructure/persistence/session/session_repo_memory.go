// Package session 会话仓储infrastructure层实现，提供Redis与内存适配。
package session

import (
	"context"

	"sync"
	"time"

	"go.uber.org/zap"

	gatewayerr "insectworld/server/gateway/domain/errors"
	domainsession "insectworld/server/gateway/domain/session"
)

// SessionRepoMemory 内存会话仓储，实现SessionRepository接口。
//
// 改造现有gateway/infrastructure/session/store.go的Store，使其实现SessionRepository接口。
// 保留并发map结构与sync.RWMutex，用于单测与无Redis环境降级（design整合步骤3）。
type SessionRepoMemory struct {
	sessions map[int64]*domainsession.OnlineSession // 会话池，key=玩家ID
	ttl      time.Duration                          // 会话TTL，超时自动清理
	mu       sync.RWMutex                           // 读写锁，保护会话池并发访问
	logger   *zap.Logger                            // 结构化日志
}

// NewSessionRepoMemory 创建内存会话仓储实例。
func NewSessionRepoMemory(ttl time.Duration, logger *zap.Logger) *SessionRepoMemory {
	return &SessionRepoMemory{
		sessions: make(map[int64]*domainsession.OnlineSession),
		ttl:      ttl,
		logger:   logger,
	}
}

// Save 保存会话聚合根到内存map。
func (m *SessionRepoMemory) Save(ctx context.Context, sess *domainsession.OnlineSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sess.PlayerID()] = sess
	return nil
}

// FindByPlayerID 按玩家ID查询在线会话。
//
// 会话不存在返回ErrSessionNotFound。
func (m *SessionRepoMemory) FindByPlayerID(ctx context.Context, playerID int64) (*domainsession.OnlineSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sess, ok := m.sessions[playerID]
	if !ok {
		return nil, gatewayerr.ErrSessionNotFound
	}
	return sess, nil
}

// Delete 删除会话。
func (m *SessionRepoMemory) Delete(ctx context.Context, playerID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, playerID)
	return nil
}

// FindExpired 查询超时会话，供SessionTimeoutCleaner周期清理。
func (m *SessionRepoMemory) FindExpired(ctx context.Context, thresholdTime int64, limit int) ([]*domainsession.OnlineSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var expired []*domainsession.OnlineSession
	count := 0
	for _, sess := range m.sessions {
		if count >= limit {
			break
		}
		if sess.HeartbeatTime() < thresholdTime {
			expired = append(expired, sess)
			count++
		}
	}
	return expired, nil
}

// 确保 SessionRepoMemory 实现 SessionRepository 接口（编译期检查）。
var _ domainsession.SessionRepository = (*SessionRepoMemory)(nil)
