// Package etcd Config服务etcd客户端封装，提供配置读写与watch订阅。
//
// infrastructure层技术适配，实现domain层ConfigRepository接口。
package etcd

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// Client etcd客户端封装，管理配置的读写与变更订阅。
type Client struct {
	endpoints []string          // etcd端点列表
	configs   map[string][]byte // 配置缓存，key=配置键
	mu        sync.RWMutex      // 读写锁，保护配置缓存并发访问
	logger    *zap.Logger       // 结构化日志
}

// NewClient 创建etcd客户端实例。
func NewClient(endpoints []string, logger *zap.Logger) *Client {
	return &Client{
		endpoints: endpoints,
		configs:   make(map[string][]byte),
		logger:    logger,
	}
}

// Get 读取配置值。
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.configs[key]
	if !ok {
		return nil, fmt.Errorf("配置键 %s 不存在", key)
	}
	return val, nil
}

// Put 写入配置值。
func (c *Client) Put(ctx context.Context, key string, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.configs[key] = value
	c.logger.Info("配置写入成功", zap.String("key", key), zap.Int("value_size", len(value)))
	return nil
}

// Watch 订阅配置变更。
// TODO 后续接入etcd watch API，当前为占位实现。
func (c *Client) Watch(ctx context.Context, key string) error {
	_ = ctx
	_ = key
	return nil
}
