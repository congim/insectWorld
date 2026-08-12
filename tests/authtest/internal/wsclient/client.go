// Package wsclient 测试端WebSocket客户端，连接被测服务认证端点发送AuthMessage接收AuthResponse。
package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"insectworld/tests/authtest/internal/contract"
)

// 连接状态枚举常量。
// 取值映射：1=未连接 2=连接中 3=已连接 4=已断开
const (
	StateNotConnected = 1 // 未连接
	StateConnecting   = 2 // 连接中
	StateConnected    = 3 // 已连接
	StateDisconnected = 4 // 已断开
)

// 读写超时时间。
const (
	writeTimeout = 5 * time.Second  // 写超时
	readTimeout  = 10 * time.Second // 读超时
)

// RequestResult 请求结果。
type RequestResult struct {
	Response   contract.AuthResponse // 响应
	DurationMs int64                 // 耗时毫秒
	Err        error                 // 错误
}

// AuthWSClient 认证WebSocket客户端。
type AuthWSClient struct {
	conn   *websocket.Conn // WebSocket连接
	state  int             // 连接状态
	wsURL  string          // WebSocket URL
	mu     sync.Mutex      // 互斥锁
	logger *zap.Logger     // 结构化日志
}

// NewAuthWSClient 创建认证WebSocket客户端实例。
func NewAuthWSClient(wsURL string, logger *zap.Logger) *AuthWSClient {
	return &AuthWSClient{
		state:  StateNotConnected,
		wsURL:  wsURL,
		logger: logger,
	}
}

// Send 发送认证消息并接收响应。
func (c *AuthWSClient) Send(ctx context.Context, msg contract.AuthMessage) (*RequestResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := &RequestResult{}
	startTime := time.Now()

	if c.conn == nil || c.state != StateConnected {
		if err := c.connect(); err != nil {
			result.Err = err
			result.DurationMs = time.Since(startTime).Milliseconds()
			return result, err
		}
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return result, fmt.Errorf("消息序列化失败: %w", err)
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.state = StateDisconnected
		return result, fmt.Errorf("被测服务未启动或连接失败: %w", err)
	}

	_, respData, err := c.conn.ReadMessage()
	if err != nil {
		c.state = StateDisconnected
		return result, fmt.Errorf("请求超时，请检查被测服务状态: %w", err)
	}

	if err := json.Unmarshal(respData, &result.Response); err != nil {
		return result, fmt.Errorf("响应解析失败: %w", err)
	}

	result.DurationMs = time.Since(startTime).Milliseconds()
	c.logger.Info("认证请求完成",
		zap.String("type", msg.Type),
		zap.Int64("duration_ms", result.DurationMs),
		zap.Bool("success", result.Response.Success),
	)
	return result, nil
}

// Close 关闭连接。
func (c *AuthWSClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.state = StateDisconnected
	return nil
}

// State 返回当前连接状态。
func (c *AuthWSClient) State() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// connect 建立WebSocket连接。
func (c *AuthWSClient) connect() error {
	c.state = StateConnecting
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.Dial(c.wsURL, nil)
	if err != nil {
		c.state = StateDisconnected
		return fmt.Errorf("连接被测服务失败: %w", err)
	}
	conn.SetReadLimit(1 << 20)
	c.conn = conn
	c.state = StateConnected
	return nil
}
