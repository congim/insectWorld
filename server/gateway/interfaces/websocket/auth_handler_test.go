package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gatewayerr "insectworld/server/gateway/domain/errors"

	"go.uber.org/zap"
)

// TestWSAuthHandler_HandleMessageInvalidJSON 测试非法JSON消息返回错误响应。
func TestWSAuthHandler_HandleMessageInvalidJSON(t *testing.T) {
	logger := zap.NewNop()
	handler := &WSAuthHandler{logger: logger} // 零值command字段，非法JSON分支不触达

	resp, err := handler.HandleMessage(context.Background(), []byte("not-json"), "127.0.0.1", "conn-001")
	require.NoError(t, err)

	var authResp AuthResponse
	require.NoError(t, json.Unmarshal(resp, &authResp))
	assert.False(t, authResp.Success)
	assert.Equal(t, "error", authResp.Type)
	assert.Equal(t, 17014, authResp.ErrorCode)
	assert.Equal(t, "消息格式错误", authResp.ErrorMsg)
}

// TestWSAuthHandler_HandleMessageUnknownType 测试未知消息类型返回错误响应。
func TestWSAuthHandler_HandleMessageUnknownType(t *testing.T) {
	logger := zap.NewNop()
	handler := &WSAuthHandler{logger: logger}

	msg, _ := json.Marshal(AuthMessage{Type: "unknown_type"})
	resp, err := handler.HandleMessage(context.Background(), msg, "127.0.0.1", "conn-001")
	require.NoError(t, err)

	var authResp AuthResponse
	require.NoError(t, json.Unmarshal(resp, &authResp))
	assert.False(t, authResp.Success)
	assert.Equal(t, "unknown_type", authResp.Type)
	assert.Equal(t, 17014, authResp.ErrorCode)
	assert.Contains(t, authResp.ErrorMsg, "未知消息类型")
}

// TestExtractErrCode 测试从错误中提取错误码。
func TestExtractErrCode(t *testing.T) {
	t.Run("nil错误返回0", func(t *testing.T) {
		assert.Equal(t, 0, extractErrCode(nil))
	})

	t.Run("GatewayError提取码", func(t *testing.T) {
		assert.Equal(t, 17020, extractErrCode(gatewayerr.ErrAccountNotFound))
		assert.Equal(t, 17022, extractErrCode(gatewayerr.ErrAccountBanned))
		assert.Equal(t, 17010, extractErrCode(gatewayerr.ErrInvalidUsernameFormat))
	})

	t.Run("包裹的GatewayError提取码", func(t *testing.T) {
		wrapped := errors.New("包裹: " + gatewayerr.ErrAccountNotFound.Error())
		// errors.As对包裹的非GatewayError返回0
		assert.Equal(t, 0, extractErrCode(wrapped))
	})

	t.Run("非GatewayError返回0", func(t *testing.T) {
		assert.Equal(t, 0, extractErrCode(errors.New("plain error")))
	})
}

// TestAuthMessageJSONRoundTrip 测试认证消息JSON序列化往返。
func TestAuthMessageJSONRoundTrip(t *testing.T) {
	original := AuthMessage{
		Type:     MsgTypeLogin,
		Username: "testuser",
		Password: "$3cr3t",
		DeviceID: "device-001",
	}
	data, err := json.Marshal(original)
	require.NoError(t, err)

	var restored AuthMessage
	require.NoError(t, json.Unmarshal(data, &restored))
	assert.Equal(t, MsgTypeLogin, restored.Type)
	assert.Equal(t, "testuser", restored.Username)
	assert.Equal(t, "$3cr3t", restored.Password)
	assert.Equal(t, "device-001", restored.DeviceID)
}

// TestAuthResponseJSON 测试认证响应JSON序列化字段名。
func TestAuthResponseJSON(t *testing.T) {
	resp := AuthResponse{
		Type:         MsgTypeLogin,
		Success:      true,
		Token:        "token-str",
		PlayerID:     1001,
		SessionTTLms: 300000,
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)
	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"type":"login"`)
	assert.Contains(t, jsonStr, `"success":true`)
	assert.Contains(t, jsonStr, `"token":"token-str"`)
	assert.Contains(t, jsonStr, `"player_id":1001`)
	assert.Contains(t, jsonStr, `"session_ttl_ms":300000`)
}

// TestMsgTypeConstants 测试消息类型常量字符串值。
func TestMsgTypeConstants(t *testing.T) {
	assert.Equal(t, "register", MsgTypeRegister)
	assert.Equal(t, "login", MsgTypeLogin)
	assert.Equal(t, "logout", MsgTypeLogout)
	assert.Equal(t, "heartbeat", MsgTypeHeartbeat)
	assert.Equal(t, "authenticate", MsgTypeAuthenticate)
}

// TestNewWSAuthHandler 测试创建WSAuthHandler实例。
func TestNewWSAuthHandler(t *testing.T) {
	logger := zap.NewNop()
	handler := NewWSAuthHandler(nil, nil, nil, nil, nil, logger)
	require.NotNil(t, handler)
}

// TestWSServerConstants 测试wsUpgrader配置。
func TestWSServerConstants(t *testing.T) {
	assert.Equal(t, 4096, wsUpgrader.ReadBufferSize)
	assert.Equal(t, 4096, wsUpgrader.WriteBufferSize)
}

// TestNewWSServer 测试创建WSServer实例。
func TestNewWSServer(t *testing.T) {
	logger := zap.NewNop()
	server := NewWSServer(nil, logger)
	require.NotNil(t, server)
}
