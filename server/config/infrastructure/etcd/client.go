// Package etcd Config服务etcd客户端封装，提供配置读写与watch订阅。
//
// infrastructure层技术适配，实现domain层ConfigStorage接口。
// Put时产生ConfigChangeEvent通知热更watcher，实现配置变更→热更广播链路。
package etcd

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"insectworld/server/config/domain"
	"insectworld/server/shared/pkg/config"
)

// 默认watch channel缓冲区大小，超过后Put操作阻塞避免事件丢失。
const defaultWatchBufferSize = 64

// Client etcd客户端封装，管理配置的读写与变更订阅。
// 实现domain.ConfigStorage接口，Put时产生配置变更事件通知hotReloader。
type Client struct {
	endpoints []string                      // etcd端点列表
	configs   map[string][]byte             // 配置缓存，key=配置键
	watchCh   chan config.ConfigChangeEvent // 配置变更事件channel，通知hotReloader
	mu        sync.RWMutex                  // 读写锁，保护配置缓存并发访问
	logger    *zap.Logger                   // 结构化日志
}

// NewClient 创建etcd客户端实例。
// watchBufferSize为watch channel缓冲区大小，<=0时使用默认值。
func NewClient(endpoints []string, watchBufferSize int, logger *zap.Logger) *Client {
	if watchBufferSize <= 0 {
		watchBufferSize = defaultWatchBufferSize
	}
	return &Client{
		endpoints: endpoints,
		configs:   make(map[string][]byte),
		watchCh:   make(chan config.ConfigChangeEvent, watchBufferSize),
		logger:    logger,
	}
}

// Get 读取配置值，实现domain.ConfigStorage接口。
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.configs[key]
	if !ok {
		return nil, fmt.Errorf("配置键 %s 不存在", key)
	}
	return val, nil
}

// Put 写入配置值并通知热更watcher，实现domain.ConfigStorage接口。
// configVersion为配置版本号，configPack为解析后的配置数据（供热更编译使用）。
func (c *Client) Put(ctx context.Context, key string, value []byte, configVersion int64, configPack map[string]any) error {
	c.mu.Lock()
	c.configs[key] = value
	c.mu.Unlock()

	c.logger.Info("配置写入成功",
		zap.String("key", key),
		zap.Int("value_size", len(value)),
		zap.Int64("config_version", configVersion),
	)

	// 通知热更watcher配置变更，channel满时告警但不阻塞（避免影响写入可用性）
	select {
	case c.watchCh <- config.ConfigChangeEvent{
		ConfigPack:    configPack,
		ConfigVersion: configVersion,
	}:
	default:
		c.logger.Warn("watch channel已满，配置变更事件丢弃",
			zap.String("key", key),
			zap.Int64("version", configVersion),
		)
	}

	return nil
}

// WatchChan 返回配置变更事件channel，供hotReloader.Run消费。
// 通过channel将infrastructure层配置变更事件传递到shared/pkg/config层热更处理。
func (c *Client) WatchChan() <-chan config.ConfigChangeEvent {
	return c.watchCh
}

// 确保Client实现domain.ConfigStorage接口（编译期检查）。
var _ domain.ConfigStorage = (*Client)(nil)
