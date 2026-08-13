// Package servernode 游戏服节点聚合根单元测试。
package servernode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewServerNode 测试节点创建。
func TestNewServerNode(t *testing.T) {
	n := NewServerNode(1, 100, RoleWorld, "1.0.0", "127.0.0.1", 50051, 10000, 1000)
	assert.Equal(t, int64(1), n.NodeID())
	assert.Equal(t, int64(100), n.ZoneID())
	assert.Equal(t, StatusOnline, n.Status())
	assert.True(t, n.IsOnline())
}

// TestServerNode_Heartbeat 测试心跳更新。
func TestServerNode_Heartbeat(t *testing.T) {
	n := NewServerNode(1, 100, RoleWorld, "1.0.0", "127.0.0.1", 50051, 10000, 1000)
	n.Heartbeat(5000, 2000)
	assert.Equal(t, 0.5, n.LoadRate())
}

// TestServerNode_Drain 测试进入排水状态。
func TestServerNode_Drain(t *testing.T) {
	n := NewServerNode(1, 100, RoleWorld, "1.0.0", "127.0.0.1", 50051, 10000, 1000)
	err := n.Drain()
	require.NoError(t, err)
	assert.Equal(t, StatusDraining, n.Status())
	assert.False(t, n.IsOnline())
}

// TestServerNode_Drain_NotOnline 测试非在线状态排水失败。
func TestServerNode_Drain_NotOnline(t *testing.T) {
	n := NewServerNode(1, 100, RoleWorld, "1.0.0", "127.0.0.1", 50051, 10000, 1000)
	n.Offline()
	err := n.Drain()
	assert.Error(t, err)
}

// TestServerNode_Offline 测试节点下线。
func TestServerNode_Offline(t *testing.T) {
	n := NewServerNode(1, 100, RoleWorld, "1.0.0", "127.0.0.1", 50051, 10000, 1000)
	n.Heartbeat(5000, 2000)
	n.Offline()
	assert.Equal(t, StatusOffline, n.Status())
	assert.Equal(t, 0.0, n.LoadRate())
}

// TestServerNode_LoadRate 测试负载比率计算。
func TestServerNode_LoadRate(t *testing.T) {
	n := NewServerNode(1, 100, RoleWorld, "1.0.0", "127.0.0.1", 50051, 10000, 1000)
	n.Heartbeat(8000, 2000)
	assert.Equal(t, 0.8, n.LoadRate())
}
