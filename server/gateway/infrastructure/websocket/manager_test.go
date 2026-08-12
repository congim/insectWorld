// Package websocket Gateway服务WebSocket连接管理，提供连接池管理与心跳维护。
package websocket

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

// mockSender 测试用消息发送器mock。
type mockSender struct {
	mu       sync.Mutex
	messages [][]byte
	sendErr  error
}

func (m *mockSender) Send(msg []byte) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockSender) getMessages() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([][]byte, len(m.messages))
	copy(cp, m.messages)
	return cp
}

// 确保 mockSender 实现 MessageSender 接口（编译期检查）。
var _ MessageSender = (*mockSender)(nil)

// TestConnectionManager_AddAndRemove 测试连接添加与移除。
func TestConnectionManager_AddAndRemove(t *testing.T) {
	logger := zap.NewNop()
	mgr := NewConnectionManager(logger)
	ctx := context.Background()

	require.NoError(t, mgr.AddConnection(ctx, 1001, "conn-001", &mockSender{}))
	assert.True(t, mgr.IsOnline(1001))
	assert.Equal(t, 1, mgr.OnlineCount())

	require.NoError(t, mgr.RemoveConnection(ctx, 1001))
	assert.False(t, mgr.IsOnline(1001))
	assert.Equal(t, 0, mgr.OnlineCount())
}

// TestConnectionManager_RemoveNonExistent 测试移除不存在的连接返回错误。
func TestConnectionManager_RemoveNonExistent(t *testing.T) {
	logger := zap.NewNop()
	mgr := NewConnectionManager(logger)
	err := mgr.RemoveConnection(context.Background(), 9999)
	require.Error(t, err)
}

// TestConnectionManager_Send 测试向在线玩家推送消息。
func TestConnectionManager_Send(t *testing.T) {
	logger := zap.NewNop()
	mgr := NewConnectionManager(logger)
	ctx := context.Background()

	sender := &mockSender{}
	require.NoError(t, mgr.AddConnection(ctx, 1001, "conn-001", sender))

	msg := []byte(`{"type":"kick_out"}`)
	require.NoError(t, mgr.Send(ctx, 1001, msg))

	messages := sender.getMessages()
	require.Len(t, messages, 1)
	assert.Equal(t, msg, messages[0])
}

// TestConnectionManager_SendToNonExistent 测试向不在线玩家推送返回错误。
func TestConnectionManager_SendToNonExistent(t *testing.T) {
	logger := zap.NewNop()
	mgr := NewConnectionManager(logger)
	err := mgr.Send(context.Background(), 9999, []byte("msg"))
	require.Error(t, err)
}

// TestConnectionManager_SendWithNilSender 测试sender未设置时推送返回错误。
func TestConnectionManager_SendWithNilSender(t *testing.T) {
	logger := zap.NewNop()
	mgr := NewConnectionManager(logger)
	ctx := context.Background()

	// 传入nil sender
	require.NoError(t, mgr.AddConnection(ctx, 1001, "conn-001", nil))
	err := mgr.Send(ctx, 1001, []byte("msg"))
	require.Error(t, err)
}

// TestConnectionManager_SendSenderError 测试sender返回错误时推送返回错误。
func TestConnectionManager_SendSenderError(t *testing.T) {
	logger := zap.NewNop()
	mgr := NewConnectionManager(logger)
	ctx := context.Background()

	sender := &mockSender{sendErr: errors.New("connection closed")}
	require.NoError(t, mgr.AddConnection(ctx, 1001, "conn-001", sender))

	err := mgr.Send(ctx, 1001, []byte("msg"))
	require.Error(t, err)
}

// TestConnectionManager_OverwriteConnection 测试同一玩家重复添加连接覆盖旧连接。
func TestConnectionManager_OverwriteConnection(t *testing.T) {
	logger := zap.NewNop()
	mgr := NewConnectionManager(logger)
	ctx := context.Background()

	sender1 := &mockSender{}
	sender2 := &mockSender{}
	require.NoError(t, mgr.AddConnection(ctx, 1001, "conn-old", sender1))
	require.NoError(t, mgr.AddConnection(ctx, 1001, "conn-new", sender2))

	assert.Equal(t, 1, mgr.OnlineCount(), "同一玩家应只占一个连接槽")

	// 推送应走新sender
	require.NoError(t, mgr.Send(ctx, 1001, []byte("msg")))
	assert.Len(t, sender1.getMessages(), 0, "旧sender不应收到消息")
	assert.Len(t, sender2.getMessages(), 1, "新sender应收到消息")
}

// TestConnectionManager_UpdateHeartbeat 测试连接心跳更新。
func TestConnectionManager_UpdateHeartbeat(t *testing.T) {
	logger := zap.NewNop()
	mgr := NewConnectionManager(logger)
	ctx := context.Background()

	require.NoError(t, mgr.AddConnection(ctx, 1001, "conn-001", &mockSender{}))
	// 更新心跳不应panic，且对不存在玩家也不报错
	mgr.UpdateHeartbeat(1001)
	mgr.UpdateHeartbeat(9999) // 不存在玩家，无操作
	assert.True(t, mgr.IsOnline(1001))
}

// TestConnectionManager_MultiplePlayers 测试多玩家连接管理。
func TestConnectionManager_MultiplePlayers(t *testing.T) {
	logger := zap.NewNop()
	mgr := NewConnectionManager(logger)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		require.NoError(t, mgr.AddConnection(ctx, int64(1000+i), "conn", &mockSender{}))
	}
	assert.Equal(t, 10, mgr.OnlineCount())

	// 移除部分
	for i := 0; i < 5; i++ {
		require.NoError(t, mgr.RemoveConnection(ctx, int64(1000+i)))
	}
	assert.Equal(t, 5, mgr.OnlineCount())

	// 剩余玩家仍在线
	for i := 5; i < 10; i++ {
		assert.True(t, mgr.IsOnline(int64(1000+i)))
	}
}
