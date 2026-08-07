// Package config 共享内核配置模块，提供配置加载/校验/查询统一API。
// 本文件定义ConfigConsistencyChecker跨服务配置一致性检查器，
// 收集各服务config.reloaded确认，未确认服务告警。
package config

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// 服务确认状态常量（规范1）。
const (
	ReloadStatusPending   = 1 // 待确认
	ReloadStatusConfirmed = 2 // 已确认
)

// ConfigConsistencyChecker 跨服务配置一致性检查器。
// 收集各服务config.reloaded确认，未确认服务告警（config_reload_pending_count指标）。
type ConfigConsistencyChecker struct {
	currentVersion int64          // 当前配置版本号
	serviceStatus  map[string]int // 各服务确认状态，key=服务名，value=确认状态
	mu             sync.RWMutex   // 读写锁，保证并发安全
	logger         *zap.Logger    // 结构化日志器（规范7）
	checkInterval  time.Duration  // 检查间隔
}

// NewConfigConsistencyChecker 创建跨服务配置一致性检查器实例。
func NewConfigConsistencyChecker(logger *zap.Logger, checkInterval time.Duration) *ConfigConsistencyChecker {
	return &ConfigConsistencyChecker{
		serviceStatus: make(map[string]int),
		logger:        logger,
		checkInterval: checkInterval,
	}
}

// StartReload 开始新一轮配置热更，重置所有服务确认状态为待确认。
func (c *ConfigConsistencyChecker) StartReload(ctx context.Context, version int64, services []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.currentVersion = version
	for _, svc := range services {
		c.serviceStatus[svc] = ReloadStatusPending
	}

	c.logger.Info("配置热更开始，等待各服务确认",
		zap.Int64("config_version", version),
		zap.Int("service_count", len(services)),
	)
}

// ConfirmReload 服务确认配置热更已生效。
func (c *ConfigConsistencyChecker) ConfirmReload(ctx context.Context, serviceName string, version int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if version != c.currentVersion {
		c.logger.Warn("服务确认的配置版本与当前版本不匹配",
			zap.String("service", serviceName),
			zap.Int64("confirmed_version", version),
			zap.Int64("current_version", c.currentVersion),
		)
		return
	}

	c.serviceStatus[serviceName] = ReloadStatusConfirmed
	c.logger.Info("服务确认配置热更",
		zap.String("service", serviceName),
		zap.Int64("config_version", version),
	)
}

// Run 启动后台检查goroutine，通过context控制生命周期（规范9goroutine安全）。
func (c *ConfigConsistencyChecker) Run(ctx context.Context) {
	ticker := time.NewTicker(c.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("配置一致性检查器停止")
			return
		case <-ticker.C:
			c.check(ctx)
		}
	}
}

// check 执行一次一致性检查，告警未确认服务。
func (c *ConfigConsistencyChecker) check(ctx context.Context) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var pendingServices []string
	for svc, status := range c.serviceStatus {
		if status == ReloadStatusPending {
			pendingServices = append(pendingServices, svc)
		}
	}

	pendingCount := len(pendingServices)
	if pendingCount > 0 {
		c.logger.Warn("配置热更未全部确认，存在待确认服务",
			zap.Int64("config_version", c.currentVersion),
			zap.Int("pending_count", pendingCount),
			zap.Strings("pending_services", pendingServices),
		)
	} else {
		c.logger.Info("配置热更全部确认",
			zap.Int64("config_version", c.currentVersion),
		)
	}
}

// PendingCount 返回待确认服务数量（用于指标采集）。
func (c *ConfigConsistencyChecker) PendingCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	for _, status := range c.serviceStatus {
		if status == ReloadStatusPending {
			count++
		}
	}
	return count
}

// CurrentVersion 返回当前配置版本号。
func (c *ConfigConsistencyChecker) CurrentVersion() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentVersion
}
