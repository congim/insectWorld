// Package command Gateway服务application层命令，编排用户认证操作。
//
// application层不直接import infrastructure（规范3），通过domain层接口 + DI组装。
// 命令编排顺序严格遵循design.md，覆盖spec全部业务规则与异常场景。
package command

// RegisterRequest 注册请求DTO。
type RegisterRequest struct {
	Username string // 用户名
	Password string // 密码明文，处理完立即置零（spec 4.3 安全性2）
	SourceIP string // 注册来源IP
}

// RegisterResponse 注册响应DTO。
type RegisterResponse struct {
	PlayerID int64 // 玩家ID，雪花算法生成
}

// LoginRequest 登录请求DTO。
type LoginRequest struct {
	Username string // 用户名
	Password string // 密码明文，处理完立即置零
	SourceIP string // 登录来源IP
	DeviceID string // 设备ID，标识客户端设备
	ConnID   string // 连接ID，标识WebSocket连接
}

// LoginResponse 登录响应DTO。
//
// 不含密码/哈希/盐（spec 5.2.1 规则11）。
type LoginResponse struct {
	AccessToken  string // 访问令牌
	PlayerID     int64  // 玩家ID
	SessionTTLms int64  // 会话TTL，毫秒级
}

// LogoutRequest 登出请求DTO。
type LogoutRequest struct {
	AccessToken string // 访问令牌
	PlayerID    int64  // 玩家ID
}

// LogoutResponse 登出响应DTO。
type LogoutResponse struct {
	Success bool // 登出是否成功
}

// HeartbeatRequest 心跳请求DTO。
type HeartbeatRequest struct {
	AccessToken string // 访问令牌
	PlayerID    int64  // 玩家ID
}

// BanRequest 封禁请求DTO。
type BanRequest struct {
	PlayerID   int64  // 玩家ID
	DurationMs int64  // 封禁时长，毫秒级，0=永久封禁
	Reason     string // 封禁原因
	AdminID    string // 操作管理员ID，用于审计
}
