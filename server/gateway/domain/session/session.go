// Package session 在线会话聚合根，维护玩家在线会话的一致性边界。
//
// domain层零外部依赖（规范3），SessionRepository接口在本包声明，
// infrastructure层实现Redis与内存适配。会话状态机：活跃→待销毁→已销毁。
// 整合现有gateway/infrastructure/session/store.go的Session结构，补全status/tokenVersion/deviceID字段。
package session

import (
	"fmt"

	gatewayerr "insectworld/server/gateway/domain/errors"
)

// 会话状态枚举常量，表示在线会话的当前状态。
// 取值映射：1=活跃 2=待销毁
const (
	SessionStatusActive     = 1 // 活跃状态，可更新心跳
	SessionStatusDestroying = 2 // 待销毁状态，拒绝心跳更新，等待清理
)

// OnlineSession 在线会话聚合根，维护玩家在线会话的一致性边界。
//
// 聚合根字段私有，外部通过方法访问与变更（DDD聚合根一致性边界）。
// 状态变更（心跳更新/销毁）通过方法保证状态机流转合法性。
// 所有ID与时间戳用int64（规范8），状态用int枚举（规范8）。
type OnlineSession struct {
	playerID      int64  // 玩家ID
	connID        string // 连接ID，标识WebSocket连接
	loginTime     int64  // 登录时间戳，毫秒级
	heartbeatTime int64  // 最后心跳时间戳，毫秒级
	status        int    // 会话状态：1=活跃 2=待销毁
	tokenVersion  int    // 令牌版本号，与AccessToken.Version对齐
	deviceID      string // 设备ID，标识客户端设备
}

// NewOnlineSession 创建在线会话聚合根实例，初始状态为活跃，心跳时间=登录时间。
func NewOnlineSession(playerID int64, connID string, loginTime int64, tokenVersion int, deviceID string) *OnlineSession {
	return &OnlineSession{
		playerID:      playerID,
		connID:        connID,
		loginTime:     loginTime,
		heartbeatTime: loginTime,
		status:        SessionStatusActive,
		tokenVersion:  tokenVersion,
		deviceID:      deviceID,
	}
}

// PlayerID 返回玩家ID。
func (s *OnlineSession) PlayerID() int64 { return s.playerID }

// ConnID 返回连接ID。
func (s *OnlineSession) ConnID() string { return s.connID }

// LoginTime 返回登录时间戳，毫秒级。
func (s *OnlineSession) LoginTime() int64 { return s.loginTime }

// HeartbeatTime 返回最后心跳时间戳，毫秒级。
func (s *OnlineSession) HeartbeatTime() int64 { return s.heartbeatTime }

// Status 返回会话状态：1=活跃 2=待销毁。
func (s *OnlineSession) Status() int { return s.status }

// TokenVersion 返回令牌版本号。
func (s *OnlineSession) TokenVersion() int { return s.tokenVersion }

// DeviceID 返回设备ID。
func (s *OnlineSession) DeviceID() string { return s.deviceID }

// IsExpired 判断会话是否已超时，now为当前时间戳，毫秒级。
// now - heartbeatTime > timeoutMs视为超时。
func (s *OnlineSession) IsExpired(timeoutMs int64, now int64) bool {
	return now-s.heartbeatTime > timeoutMs
}

// UpdateHeartbeat 更新会话心跳时间，仅活跃状态可更新。
//
// 待销毁状态返回错误，防止已销毁会话被重新激活（状态机不可逆）。
func (s *OnlineSession) UpdateHeartbeat(now int64) error {
	if s.status != SessionStatusActive {
		return fmt.Errorf("会话非活跃状态，无法更新心跳: %w", gatewayerr.ErrSessionNotFound)
	}
	s.heartbeatTime = now
	return nil
}

// Destroy 销毁会话，状态机：活跃→待销毁。
//
// 已待销毁状态再次调用为幂等，不报错。
func (s *OnlineSession) Destroy() error {
	s.status = SessionStatusDestroying
	return nil
}
