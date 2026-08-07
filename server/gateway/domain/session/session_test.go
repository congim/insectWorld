// Package session 在线会话聚合根，维护玩家在线会话的一致性边界。
package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gatewayerr "insectworld/server/gateway/domain/errors"
)

// TestNewOnlineSession 测试创建在线会话。
func TestNewOnlineSession(t *testing.T) {
	t.Run("正常创建", func(t *testing.T) {
		sess := NewOnlineSession(1001, "conn-001", 1700000000000, 1, "device-001")
		assert.Equal(t, int64(1001), sess.PlayerID())
		assert.Equal(t, "conn-001", sess.ConnID())
		assert.Equal(t, int64(1700000000000), sess.LoginTime())
		assert.Equal(t, int64(1700000000000), sess.HeartbeatTime())
		assert.Equal(t, SessionStatusActive, sess.Status())
		assert.Equal(t, 1, sess.TokenVersion())
		assert.Equal(t, "device-001", sess.DeviceID())
	})
}

// TestOnlineSession_IsExpired 测试会话超时判定。
func TestOnlineSession_IsExpired(t *testing.T) {
	sess := NewOnlineSession(1001, "conn-001", 1700000000000, 1, "device-001")
	t.Run("未超时", func(t *testing.T) {
		assert.False(t, sess.IsExpired(300000, 1700000000000+100000))
	})
	t.Run("已超时", func(t *testing.T) {
		assert.True(t, sess.IsExpired(300000, 1700000000000+400000))
	})
}

// TestOnlineSession_UpdateHeartbeat 测试心跳更新。
func TestOnlineSession_UpdateHeartbeat(t *testing.T) {
	t.Run("活跃状态可更新", func(t *testing.T) {
		sess := NewOnlineSession(1001, "conn-001", 1700000000000, 1, "device-001")
		err := sess.UpdateHeartbeat(1700000000000 + 60000)
		require.NoError(t, err)
		assert.Equal(t, int64(1700000000000+60000), sess.HeartbeatTime())
	})

	t.Run("待销毁状态拒绝更新", func(t *testing.T) {
		sess := NewOnlineSession(1001, "conn-001", 1700000000000, 1, "device-001")
		_ = sess.Destroy()
		err := sess.UpdateHeartbeat(1700000000000 + 60000)
		assert.Error(t, err)
	})
}

// TestOnlineSession_Destroy 测试销毁状态机流转。
func TestOnlineSession_Destroy(t *testing.T) {
	t.Run("活跃→待销毁", func(t *testing.T) {
		sess := NewOnlineSession(1001, "conn-001", 1700000000000, 1, "device-001")
		err := sess.Destroy()
		require.NoError(t, err)
		assert.Equal(t, SessionStatusDestroying, sess.Status())
	})

	t.Run("幂等销毁", func(t *testing.T) {
		sess := NewOnlineSession(1001, "conn-001", 1700000000000, 1, "device-001")
		_ = sess.Destroy()
		err := sess.Destroy()
		require.NoError(t, err)
		assert.Equal(t, SessionStatusDestroying, sess.Status())
	})
}

// 确保错误码变量可用（避免未使用import告警）。
var _ = gatewayerr.ErrSessionNotFound
