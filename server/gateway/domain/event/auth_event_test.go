// Package event 用户认证领域事件，发布玩家上线/下线通知供其他服务订阅。
package event

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"insectworld/server/shared/pkg/eventbus"
)

// TestPlayerOnlineEvent_EventType 测试玩家上线事件类型常量。
func TestPlayerOnlineEvent_EventType(t *testing.T) {
	e := &PlayerOnlineEvent{PlayerID: 1001, LoginTime: 1700000000000, SourceIP: "127.0.0.1"}
	assert.Equal(t, EventTypePlayerOnline, e.EventType())
	assert.Equal(t, "auth.player_online", e.EventType())
}

// TestPlayerOnlineEvent_ToDomainEvent 测试上线事件转换为共享内核DomainEvent。
func TestPlayerOnlineEvent_ToDomainEvent(t *testing.T) {
	t.Run("正常转换", func(t *testing.T) {
		e := &PlayerOnlineEvent{
			PlayerID:  1001,
			LoginTime: 1700000000000,
			SourceIP:  "127.0.0.1",
		}
		evt, err := e.ToDomainEvent("evt-001", 1)
		require.NoError(t, err)
		assert.Equal(t, "evt-001", evt.EventID)
		assert.Equal(t, EventTypePlayerOnline, evt.EventType)
		assert.Equal(t, int64(1001), evt.AggregateID)
		assert.Equal(t, 1, evt.Version)
		assert.Equal(t, int64(1700000000000), evt.Timestamp)

		// 校验Payload反序列化还原原事件
		var restored PlayerOnlineEvent
		require.NoError(t, json.Unmarshal(evt.Payload, &restored))
		assert.Equal(t, int64(1001), restored.PlayerID)
		assert.Equal(t, int64(1700000000000), restored.LoginTime)
		assert.Equal(t, "127.0.0.1", restored.SourceIP)
	})

	t.Run("事件ID与版本号透传", func(t *testing.T) {
		e := &PlayerOnlineEvent{PlayerID: 2002, LoginTime: 1700000001000, SourceIP: "10.0.0.1"}
		evt, err := e.ToDomainEvent("online-unique-id", 5)
		require.NoError(t, err)
		assert.Equal(t, "online-unique-id", evt.EventID)
		assert.Equal(t, 5, evt.Version)
	})
}

// TestPlayerOfflineEvent_EventType 测试玩家下线事件类型常量。
func TestPlayerOfflineEvent_EventType(t *testing.T) {
	e := &PlayerOfflineEvent{PlayerID: 1001, OfflineTime: 1700000000000, Reason: OfflineReasonLogout}
	assert.Equal(t, EventTypePlayerOffline, e.EventType())
	assert.Equal(t, "auth.player_offline", e.EventType())
}

// TestPlayerOfflineEvent_ToDomainEvent 测试下线事件转换为共享内核DomainEvent，覆盖全部下线原因枚举。
func TestPlayerOfflineEvent_ToDomainEvent(t *testing.T) {
	tests := []struct {
		name   string
		reason int
	}{
		{"主动登出", OfflineReasonLogout},
		{"会话超时", OfflineReasonSessionTimeout},
		{"被踢下线", OfflineReasonKicked},
		{"封禁踢下线", OfflineReasonBanned},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &PlayerOfflineEvent{
				PlayerID:    1001,
				OfflineTime: 1700000000000,
				Reason:      tt.reason,
			}
			evt, err := e.ToDomainEvent("offline-001", 1)
			require.NoError(t, err)
			assert.Equal(t, "offline-001", evt.EventID)
			assert.Equal(t, EventTypePlayerOffline, evt.EventType)
			assert.Equal(t, int64(1001), evt.AggregateID)
			assert.Equal(t, int64(1700000000000), evt.Timestamp)

			var restored PlayerOfflineEvent
			require.NoError(t, json.Unmarshal(evt.Payload, &restored))
			assert.Equal(t, tt.reason, restored.Reason)
		})
	}
}

// TestOfflineReasonConstants 测试下线原因枚举常量取值唯一性，防止常量重复定义导致语义歧义。
func TestOfflineReasonConstants(t *testing.T) {
	reasons := map[int]string{
		OfflineReasonLogout:         "主动登出",
		OfflineReasonSessionTimeout: "会话超时",
		OfflineReasonKicked:         "被踢下线",
		OfflineReasonBanned:         "封禁踢下线",
	}
	// 枚举值应为 1-4 连续递增
	assert.Equal(t, 1, OfflineReasonLogout)
	assert.Equal(t, 2, OfflineReasonSessionTimeout)
	assert.Equal(t, 3, OfflineReasonKicked)
	assert.Equal(t, 4, OfflineReasonBanned)
	assert.Len(t, reasons, 4)
}

// TestEventTypeConstants 测试事件类型常量字符串值，防止外部订阅方依赖的字符串被意外修改。
func TestEventTypeConstants(t *testing.T) {
	assert.Equal(t, "auth.player_online", EventTypePlayerOnline)
	assert.Equal(t, "auth.player_offline", EventTypePlayerOffline)
}

// 确保 eventbus.DomainEvent 类型可用（编译期检查，避免未使用import告警）。
var _ eventbus.DomainEvent
