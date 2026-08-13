// Package servernode 游戏服节点聚合根，维护节点注册、状态与负载信息。
// ServerNode聚合根是跨服架构的基础，管理游戏服节点的上下线与路由。
// 对应spec.md 5.1.7.5节CrossServer上下文功能1"节点管理"。
package servernode

import (
	"context"
	"fmt"

	cserr "insectworld/server/crossserver/domain/errors"
)

// 节点状态常量（规范1）。
const (
	StatusOnline   = 1 // 在线
	StatusOffline  = 2 // 离线
	StatusDraining = 3 // 排水中（不接受新请求，等待存量请求完成）
)

// 节点角色常量（规范1）。
const (
	RoleWorld      = 1 // World服务节点
	RoleMatch      = 2 // Match服务节点
	RoleCrossProxy = 3 // 跨服代理节点
)

// ServerNode 游戏服节点聚合根，维护节点注册与状态。
type ServerNode struct {
	nodeID        int64  // 节点ID，全局唯一
	zoneID        int64  // 区ID，节点所属的游戏区
	role          int    // 节点角色：1=World 2=Match 3=跨服代理
	status        int    // 节点状态：1=在线 2=离线 3=排水中
	version       string // 节点版本号，用于灰度发布与版本兼容校验
	host          string // 节点主机地址
	port          int32  // 节点监听端口
	maxLoad       int64  // 最大负载容量（最大在线玩家数）
	currentLoad   int64  // 当前负载（当前在线玩家数）
	registerTime  int64  // 注册时间戳（毫秒）
	heartbeatTime int64  // 最后心跳时间戳（毫秒）
}

// NewServerNode 创建游戏服节点聚合根实例。
func NewServerNode(nodeID int64, zoneID int64, role int, version string, host string, port int32, maxLoad int64, registerTime int64) *ServerNode {
	return &ServerNode{
		nodeID:        nodeID,
		zoneID:        zoneID,
		role:          role,
		status:        StatusOnline,
		version:       version,
		host:          host,
		port:          port,
		maxLoad:       maxLoad,
		registerTime:  registerTime,
		heartbeatTime: registerTime,
	}
}

// NodeID 返回节点ID。
func (n *ServerNode) NodeID() int64 { return n.nodeID }

// ZoneID 返回区ID。
func (n *ServerNode) ZoneID() int64 { return n.zoneID }

// Status 返回节点状态。
func (n *ServerNode) Status() int { return n.status }

// IsOnline 判断节点是否在线。
func (n *ServerNode) IsOnline() bool { return n.status == StatusOnline }

// Heartbeat 更新心跳时间与负载。
func (n *ServerNode) Heartbeat(currentLoad int64, now int64) {
	n.heartbeatTime = now
	n.currentLoad = currentLoad
}

// Drain 进入排水状态，不再接受新请求。
func (n *ServerNode) Drain() error {
	if n.status != StatusOnline {
		return fmt.Errorf("节点非在线状态，nodeID=%d，当前状态=%d: %w",
			n.nodeID, n.status, cserr.ErrNodeOffline)
	}
	n.status = StatusDraining
	return nil
}

// Offline 节点下线。
func (n *ServerNode) Offline() {
	n.status = StatusOffline
	n.currentLoad = 0
}

// LoadRate 返回负载比率（当前负载/最大负载），用于路由权重计算。
func (n *ServerNode) LoadRate() float64 {
	if n.maxLoad == 0 {
		return 1.0
	}
	return float64(n.currentLoad) / float64(n.maxLoad)
}

// ServerNodeRepository 游戏服节点仓储接口，在domain层声明（规范3）。
type ServerNodeRepository interface {
	// LoadServerNode 加载节点聚合根
	LoadServerNode(ctx context.Context, nodeID int64) (*ServerNode, error)
	// SaveServerNode 保存节点聚合根
	SaveServerNode(ctx context.Context, n *ServerNode) error
}
