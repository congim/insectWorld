// Package session Gateway服务会话存储，基于Redis实现会话管理与TTL过期。
//
// infrastructure层技术适配，实现domain层SessionRepository接口。
package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Session 玩家会话信息。
type Session struct {
	PlayerID      int64     // 玩家ID
	ConnID        string    // 连接ID
	LoginTime     time.Time // 登录时间
	HeartbeatTime time.Time // 最后心跳时间
}

// Store 会话存储管理器。
type Store struct {
	sessions map[int64]*Session // 会话池，key=玩家ID
	ttl      time.Duration      // 会话TTL，超时自动清理
	mu       sync.RWMutex       // 读写锁，保护会话池并发访问
	logger   *zap.Logger        // 结构化日志
}

// NewStore 创建会话存储实例。
func NewStore(ttl time.Duration, logger *zap.Logger) *Store {
	return &Store{
		sessions: make(map[int64]*Session),
		ttl:      ttl,
		logger:   logger,
	}
}

// Set 保存会话。
func (s *Store) Set(ctx context.Context, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.PlayerID] = session
	return nil
}

// Get 查询会话。
func (s *Store) Get(ctx context.Context, playerID int64) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[playerID]
	if !ok {
		return nil, fmt.Errorf("玩家 %d 会话不存在", playerID)
	}
	return session, nil
}

// Delete 删除会话。
func (s *Store) Delete(ctx context.Context, playerID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, playerID)
	return nil
}

// CleanupExpired 清理过期会话。
func (s *Store) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	count := 0
	for id, session := range s.sessions {
		if now.Sub(session.HeartbeatTime) > s.ttl {
			delete(s.sessions, id)
			count++
		}
	}
	return count
}
