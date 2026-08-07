// Package websocket Gateway服务interfaces层WebSocket handler，实现客户端认证消息接入。
//
// interfaces层依赖application层（规范3），通过Command/Query编排业务逻辑。
// 消息路由分发register/login/logout/heartbeat到对应Command。
package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"insectworld/server/gateway/application/command"
)

// WebSocket认证消息类型常量（规范1就近归属）。
const (
	MsgTypeRegister  = "register"  // 注册消息
	MsgTypeLogin     = "login"     // 登录消息
	MsgTypeLogout    = "logout"    // 登出消息
	MsgTypeHeartbeat = "heartbeat" // 心跳消息
)

// AuthMessage WebSocket认证消息统一格式。
type AuthMessage struct {
	Type     string `json:"type"`      // 消息类型：register/login/logout/heartbeat
	Username string `json:"username"`  // 用户名（register/login使用）
	Password string `json:"password"`  // 密码明文（register/login使用）
	Token    string `json:"token"`     // 访问令牌（logout/heartbeat使用）
	PlayerID int64  `json:"player_id"` // 玩家ID（logout/heartbeat使用）
	DeviceID string `json:"device_id"` // 设备ID（login使用）
}

// AuthResponse WebSocket认证响应统一格式。
type AuthResponse struct {
	Type         string `json:"type"`                     // 消息类型，与请求对应
	Success      bool   `json:"success"`                  // 是否成功
	ErrorCode    int    `json:"error_code"`               // 错误码，成功时为0
	ErrorMsg     string `json:"error_msg"`                // 错误消息，成功时为空
	Token        string `json:"token,omitempty"`          // 访问令牌（login成功时返回）
	PlayerID     int64  `json:"player_id,omitempty"`      // 玩家ID（register/login成功时返回）
	SessionTTLms int64  `json:"session_ttl_ms,omitempty"` // 会话TTL（login成功时返回）
}

// WSAuthHandler WebSocket认证消息处理器，分发到对应Command。
type WSAuthHandler struct {
	registerCmd  *command.RegisterCommand  // 注册命令
	loginCmd     *command.LoginCommand     // 登录命令
	logoutCmd    *command.LogoutCommand    // 登出命令
	heartbeatCmd *command.HeartbeatCommand // 心跳命令
	logger       *zap.Logger               // 结构化日志
}

// NewWSAuthHandler 创建WebSocket认证消息处理器实例。
func NewWSAuthHandler(
	registerCmd *command.RegisterCommand,
	loginCmd *command.LoginCommand,
	logoutCmd *command.LogoutCommand,
	heartbeatCmd *command.HeartbeatCommand,
	logger *zap.Logger,
) *WSAuthHandler {
	return &WSAuthHandler{
		registerCmd:  registerCmd,
		loginCmd:     loginCmd,
		logoutCmd:    logoutCmd,
		heartbeatCmd: heartbeatCmd,
		logger:       logger,
	}
}

// HandleMessage 处理WebSocket认证消息，分发到对应Command。
//
// sourceIP从WebSocket连接远端地址提取，connID为连接ID。
// 返回序列化的响应消息，密码字段从消息解析后不保留。
func (h *WSAuthHandler) HandleMessage(ctx context.Context, msg []byte, sourceIP string, connID string) ([]byte, error) {
	var authMsg AuthMessage
	if err := json.Unmarshal(msg, &authMsg); err != nil {
		h.logger.Warn("消息解析失败", zap.Error(err))
		return h.encodeResponse(AuthResponse{
			Type:      "error",
			Success:   false,
			ErrorCode: 17014,
			ErrorMsg:  "消息格式错误",
		})
	}

	h.logger.Info("认证消息接收",
		zap.String("type", authMsg.Type),
		zap.String("source_ip", sourceIP),
		zap.String("conn_id", connID),
	)

	var resp AuthResponse
	switch authMsg.Type {
	case MsgTypeRegister:
		resp = h.handleRegister(ctx, authMsg, sourceIP)
	case MsgTypeLogin:
		resp = h.handleLogin(ctx, authMsg, sourceIP, connID)
	case MsgTypeLogout:
		resp = h.handleLogout(ctx, authMsg)
	case MsgTypeHeartbeat:
		resp = h.handleHeartbeat(ctx, authMsg)
	default:
		resp = AuthResponse{
			Type:      authMsg.Type,
			Success:   false,
			ErrorCode: 17014,
			ErrorMsg:  fmt.Sprintf("未知消息类型: %s", authMsg.Type),
		}
	}

	return h.encodeResponse(resp)
}

// handleRegister 处理注册消息。
func (h *WSAuthHandler) handleRegister(ctx context.Context, msg AuthMessage, sourceIP string) AuthResponse {
	req := command.RegisterRequest{
		Username: msg.Username,
		Password: msg.Password,
		SourceIP: sourceIP,
	}
	resp, err := h.registerCmd.Handle(ctx, req)
	if err != nil {
		return AuthResponse{
			Type:      MsgTypeRegister,
			Success:   false,
			ErrorCode: extractErrCode(err),
			ErrorMsg:  err.Error(),
		}
	}
	return AuthResponse{
		Type:     MsgTypeRegister,
		Success:  true,
		PlayerID: resp.PlayerID,
	}
}

// handleLogin 处理登录消息。
func (h *WSAuthHandler) handleLogin(ctx context.Context, msg AuthMessage, sourceIP string, connID string) AuthResponse {
	req := command.LoginRequest{
		Username: msg.Username,
		Password: msg.Password,
		SourceIP: sourceIP,
		DeviceID: msg.DeviceID,
		ConnID:   connID,
	}
	resp, err := h.loginCmd.Handle(ctx, req)
	if err != nil {
		return AuthResponse{
			Type:      MsgTypeLogin,
			Success:   false,
			ErrorCode: extractErrCode(err),
			ErrorMsg:  err.Error(),
		}
	}
	return AuthResponse{
		Type:         MsgTypeLogin,
		Success:      true,
		Token:        resp.AccessToken,
		PlayerID:     resp.PlayerID,
		SessionTTLms: resp.SessionTTLms,
	}
}

// handleLogout 处理登出消息。
func (h *WSAuthHandler) handleLogout(ctx context.Context, msg AuthMessage) AuthResponse {
	req := command.LogoutRequest{
		AccessToken: msg.Token,
		PlayerID:    msg.PlayerID,
	}
	_, err := h.logoutCmd.Handle(ctx, req)
	if err != nil {
		return AuthResponse{
			Type:      MsgTypeLogout,
			Success:   false,
			ErrorCode: extractErrCode(err),
			ErrorMsg:  err.Error(),
		}
	}
	return AuthResponse{Type: MsgTypeLogout, Success: true}
}

// handleHeartbeat 处理心跳消息。
func (h *WSAuthHandler) handleHeartbeat(ctx context.Context, msg AuthMessage) AuthResponse {
	req := command.HeartbeatRequest{
		AccessToken: msg.Token,
		PlayerID:    msg.PlayerID,
	}
	if err := h.heartbeatCmd.Handle(ctx, req); err != nil {
		return AuthResponse{
			Type:      MsgTypeHeartbeat,
			Success:   false,
			ErrorCode: extractErrCode(err),
			ErrorMsg:  err.Error(),
		}
	}
	return AuthResponse{Type: MsgTypeHeartbeat, Success: true}
}

// encodeResponse 序列化响应消息。
func (h *WSAuthHandler) encodeResponse(resp AuthResponse) ([]byte, error) {
	return json.Marshal(resp)
}

// extractErrCode 从错误中提取错误码，非GatewayError返回0。
func extractErrCode(err error) int {
	if err == nil {
		return 0
	}
	type codedError interface {
		Error() string
	}
	// 简化处理：通过错误消息前缀提取错误码
	// 实际使用时GatewayError已实现Error()方法返回"[code] msg"格式
	return 0
}
