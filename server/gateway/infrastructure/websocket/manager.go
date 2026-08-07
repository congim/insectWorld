// Package websocket Gateway服务WebSocket连接管理，提供连接池管理与心跳维护。
//
// infrastructure层技术适配，实现domain层ConnectionRepository接口。
// 依赖方向infrastructure → domain（规范3）。
package websocket

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConnectionManager WebSocket连接管理器，维护在线玩家的WebSocket连接池。
type ConnectionManager struct {
	connections map[int64]*Connection // 连接池，key=玩家ID
	mu          sync.RWMutex          // 读写锁，保护连接池并发访问
	logger      *zap.Logger           // 结构化日志
}

// MessageSender 消息发送能力接口，由WebSocket连接实现。
//
// 通过此接口向客户端推送消息，如踢下线通知。
type MessageSender interface {
	// Send 向客户端发送消息，连接关闭时返回错误。
	Send(msg []byte) error
}

// Connection WebSocket连接封装。
type Connection struct {
	playerID  int64         // 玩家ID
	connID    string        // 连接ID
	heartbeat int64         // 最后心跳时间戳，毫秒级
	sender    MessageSender // 消息发送器，用于向客户端推送消息
}

// NewConnectionManager 创建WebSocket连接管理器实例。
func NewConnectionManager(logger *zap.Logger) *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[int64]*Connection),
		logger:      logger,
	}
}

// AddConnection 添加新连接到连接池。
//
// sender为消息发送器，用于向客户端推送消息（如踢下线通知）。
// 传入nil时连接不支持Send推送，Send方法将返回错误。
// heartbeat初始化为当前时间，修复现有零值问题（design整合步骤）。
func (m *ConnectionManager) AddConnection(ctx context.Context, playerID int64, connID string, sender MessageSender) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.connections[playerID] = &Connection{
		playerID:  playerID,
		connID:    connID,
		heartbeat: time.Now().UnixMilli(),
		sender:    sender,
	}
	m.logger.Info("WebSocket连接添加成功",
		zap.Int64("player_id", playerID),
		zap.String("conn_id", connID),
	)
	return nil
}

// RemoveConnection 从连接池移除连接。
func (m *ConnectionManager) RemoveConnection(ctx context.Context, playerID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.connections[playerID]; !ok {
		return fmt.Errorf("玩家 %d 连接不存在", playerID)
	}
	delete(m.connections, playerID)
	m.logger.Info("WebSocket连接移除", zap.Int64("player_id", playerID))
	return nil
}

// IsOnline 检查玩家是否在线。
func (m *ConnectionManager) IsOnline(playerID int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.connections[playerID]
	return ok
}

// OnlineCount 返回当前在线连接数。
func (m *ConnectionManager) OnlineCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// Send 向指定玩家推送消息，如踢下线通知。
//
// 连接不存在或sender未设置时返回错误。
func (m *ConnectionManager) Send(ctx context.Context, playerID int64, msg []byte) error {
	m.mu.RLock()
	conn, ok := m.connections[playerID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("玩家 %d 连接不存在", playerID)
	}
	if conn.sender == nil {
		return fmt.Errorf("玩家 %d 连接不支持消息发送", playerID)
	}
	if err := conn.sender.Send(msg); err != nil {
		m.logger.Error("消息推送失败",
			zap.Int64("player_id", playerID),
			zap.String("conn_id", conn.connID),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// UpdateHeartbeat 更新连接心跳时间。
func (m *ConnectionManager) UpdateHeartbeat(playerID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conn, ok := m.connections[playerID]; ok {
		conn.heartbeat = time.Now().UnixMilli()
	}
}
