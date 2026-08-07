// Package router Gateway服务路由管理，实现路由表持久化与动态加载。
//
// infrastructure层技术适配，实现domain层RouterRepository接口。
package router

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// Route 路由规则，定义请求路径到目标服务的映射。
type Route struct {
	Path          string // 路由路径
	TargetService string // 目标服务
	Method        string // 请求方法
	RateLimit     int    // 限流阈值，0表示不限流
}

// Router 路由管理器，维护路由表并支持动态加载。
type Router struct {
	routes map[string]*Route // 路由表，key=路由路径
	mu     sync.RWMutex      // 读写锁，保护路由表并发访问
	logger *zap.Logger       // 结构化日志
}

// NewRouter 创建路由管理器实例。
func NewRouter(logger *zap.Logger) *Router {
	return &Router{
		routes: make(map[string]*Route),
		logger: logger,
	}
}

// LoadRoutes 加载路由表。
func (r *Router) LoadRoutes(ctx context.Context, routes []*Route) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, route := range routes {
		r.routes[route.Path] = route
	}
	r.logger.Info("路由表加载成功", zap.Int("route_count", len(routes)))
	return nil
}

// Match 匹配路由，根据请求路径查找目标服务。
func (r *Router) Match(path string) (*Route, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	route, ok := r.routes[path]
	return route, ok
}
