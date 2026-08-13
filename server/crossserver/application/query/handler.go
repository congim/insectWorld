// Package query CrossServer服务application层读侧查询，CQRS读模型查询handler。
package query

import (
	"context"

	"go.uber.org/zap"
)

// NodeListQuery 节点列表查询DTO。
type NodeListQuery struct {
	ZoneID int64 // 区ID，0表示查询所有区
	Role   int   // 节点角色，0表示查询所有角色
}

// NodeListResult 节点列表查询结果DTO。
type NodeListResult struct {
	Nodes []NodeInfo // 节点信息列表
}

// NodeInfo 节点信息DTO。
type NodeInfo struct {
	NodeID      int64  // 节点ID
	ZoneID      int64  // 区ID
	Role        int    // 节点角色
	Status      int    // 节点状态
	Version     string // 版本号
	CurrentLoad int64  // 当前负载
	MaxLoad     int64  // 最大负载
}

// NodeReadModel 节点读模型查询接口，在application层声明，infrastructure层实现。
// CQRS读侧通过读模型表t_server_node查询，不经过聚合根。
type NodeReadModel interface {
	// QueryNodeList 查询节点列表
	QueryNodeList(ctx context.Context, zoneID int64, role int) ([]NodeInfo, error)
}

// NodeListQueryHandler 节点列表查询handler，CQRS读侧。
type NodeListQueryHandler struct {
	nodeReadModel NodeReadModel // 节点读模型查询接口，infrastructure层注入
	logger        *zap.Logger   // 结构化日志器（规范7）
}

// NewNodeListQueryHandler 创建节点列表查询handler实例。
// nodeReadModel由infrastructure层实现，cmd/main.go组装时注入。
func NewNodeListQueryHandler(nodeReadModel NodeReadModel, logger *zap.Logger) *NodeListQueryHandler {
	return &NodeListQueryHandler{nodeReadModel: nodeReadModel, logger: logger}
}

// Handle 处理节点列表查询。
func (h *NodeListQueryHandler) Handle(ctx context.Context, q NodeListQuery) (*NodeListResult, error) {
	nodes, err := h.nodeReadModel.QueryNodeList(ctx, q.ZoneID, q.Role)
	if err != nil {
		return nil, err
	}

	h.logger.Debug("查询节点列表",
		zap.Int64("zone_id", q.ZoneID),
		zap.Int("role", q.Role),
		zap.Int("count", len(nodes)),
	)
	return &NodeListResult{Nodes: nodes}, nil
}
