// Package websocket WebSocket连接管理能力接口，domain层声明，infrastructure层实现适配。
//
// 仅声明application层使用的ConnectionManager小接口（规范4不过度设计），infrastructure/websocket.ConnectionManager隐式实现。
package websocket

import "context"

// ConnectionManager WebSocket连接管理能力接口，供application层推送踢下线通知。
//
// infrastructure/websocket.ConnectionManager通过Send方法隐式实现此接口。
// 连接不存在时返回error，application层忽略错误（踢下线为尽力推送）。
type ConnectionManager interface {
	// Send 向指定玩家推送消息，连接不存在返回error。
	Send(ctx context.Context, playerID int64, msg []byte) error
}
