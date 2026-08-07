// Package contract 测试端契约定义，与Gateway服务消息格式对齐，独立维护避免跨服务依赖。
//
// 对齐：server/gateway/interfaces/websocket/auth_handler.go
// 契约变更时需人工同步本文件与Gateway源文件（design 2.5.5）。
package contract

// WebSocket认证消息类型常量（规范1就近归属），与Gateway对齐。
const (
	MsgTypeRegister     = "register"     // 注册消息
	MsgTypeLogin        = "login"        // 登录消息
	MsgTypeLogout       = "logout"       // 登出消息
	MsgTypeHeartbeat    = "heartbeat"    // 心跳消息
	MsgTypeAuthenticate = "authenticate" // 鉴权消息
)

// AuthMessage WebSocket认证消息统一格式，与Gateway AuthMessage完全对齐。
//
// JSON标签与server/gateway/interfaces/websocket/auth_handler.go完全一致。
type AuthMessage struct {
	Type     string `json:"type"`      // 消息类型：register/login/logout/heartbeat
	Username string `json:"username"`  // 用户名（register/login使用）
	Password string `json:"password"`  // 密码明文（register/login使用）
	Token    string `json:"token"`     // 访问令牌（logout/heartbeat使用）
	PlayerID int64  `json:"player_id"` // 玩家ID（logout/heartbeat使用）
	DeviceID string `json:"device_id"` // 设备ID（login使用）
}

// AuthResponse WebSocket认证响应统一格式，与Gateway AuthResponse完全对齐。
//
// JSON标签含omitempty与Gateway完全一致。
type AuthResponse struct {
	Type         string `json:"type"`                     // 消息类型，与请求对应
	Success      bool   `json:"success"`                  // 是否成功
	ErrorCode    int    `json:"error_code"`               // 错误码，成功时为0
	ErrorMsg     string `json:"error_msg"`                // 错误消息，成功时为空
	Token        string `json:"token,omitempty"`          // 访问令牌（login成功时返回）
	PlayerID     int64  `json:"player_id,omitempty"`      // 玩家ID（register/login成功时返回）
	SessionTTLms int64  `json:"session_ttl_ms,omitempty"` // 会话TTL（login成功时返回）
}
