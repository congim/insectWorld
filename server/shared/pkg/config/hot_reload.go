// Package config 共享内核配置模块，提供配置加载/校验/查询统一API。
// 本文件定义配置热更watcher，通过etcd watch被动感知配置变更，
// 校验后原子替换本地缓存并触发业务层reload回调。
// 对应design.md 2.3.2.2节换皮配置驱动5步链路第四步。
package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// 业务切换边界常量（规范1），对应design.md配置热更业务切换边界规则。
const (
	SwitchBoundaryCombat     = 1 // 进行中战斗用开战时配置快照
	SwitchBoundaryMovement   = 2 // 进行中移动用当前最新配置
	SwitchBoundaryProduction = 3 // 生产/采集下个Tick用新配置
	SwitchBoundaryDiplomacy  = 4 // 联盟外交用当前最新配置
	SwitchBoundarySeason     = 5 // 赛季阶段下个赛季用新配置
)

// ReloadCallback 业务层reload回调函数类型。
// 业务服务注册回调，在配置热更生效时被调用。
type ReloadCallback func(ctx context.Context, configVersion int64) error

// ConfigHotReloader 配置热更watcher，通过etcd watch被动感知配置变更。
// 校验通过后原子替换本地缓存，触发业务层reload回调。
type ConfigHotReloader struct {
	compiler       *ConfigCompiler        // 配置编译器
	validator      *Validator             // 配置校验器
	callbacks      map[int]ReloadCallback // 业务层reload回调，key=切换边界类型
	mu             sync.RWMutex           // 读写锁，保证回调注册的并发安全
	logger         *zap.Logger            // 结构化日志器（规范7）
	currentVersion int64                  // 当前配置版本号
	reloading      bool                   // 是否正在热更中
	reloadMu       sync.Mutex             // 热更互斥锁，防止并发热更
}

// NewConfigHotReloader 创建配置热更watcher实例。
func NewConfigHotReloader(compiler *ConfigCompiler, validator *Validator, logger *zap.Logger) *ConfigHotReloader {
	return &ConfigHotReloader{
		compiler:  compiler,
		validator: validator,
		callbacks: make(map[int]ReloadCallback),
		logger:    logger,
	}
}

// RegisterCallback 注册业务层reload回调。
// boundaryType为业务切换边界类型，callback为回调函数。
func (h *ConfigHotReloader) RegisterCallback(boundaryType int, callback ReloadCallback) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.callbacks[boundaryType] = callback
}

// HandleReload 处理配置热更，校验后原子替换并触发回调。
// configPack为新的配置包数据，configVersion为新配置版本号。
func (h *ConfigHotReloader) HandleReload(ctx context.Context, configPack map[string]any, configVersion int64) error {
	// 防止并发热更（规范1错误码ErrHotReloadInProgress）
	h.reloadMu.Lock()
	if h.reloading {
		h.reloadMu.Unlock()
		return fmt.Errorf("配置热更失败，已有热更进行中: %w", ErrHotReloadInProgress)
	}
	h.reloading = true
	h.reloadMu.Unlock()
	defer func() {
		h.reloadMu.Lock()
		h.reloading = false
		h.reloadMu.Unlock()
	}()

	startTime := time.Now()

	h.logger.Info("配置热更开始",
		zap.Int64("config_version", configVersion),
		zap.Int64("current_version", h.currentVersion),
	)

	// 1. 编译+校验新配置
	if err := h.compiler.CompileConfigPack(ctx, configPack, configVersion); err != nil {
		h.logger.Error("配置热更失败，编译校验未通过",
			zap.Int64("config_version", configVersion),
			zap.Error(err),
		)
		return fmt.Errorf("配置热更失败，编译校验未通过: %w", err)
	}

	// 2. 原子替换本地缓存（编译器在注册时已原子替换ExtensionRegistry）

	// 3. 触发业务层reload回调，按业务切换边界应用新配置
	h.mu.RLock()
	callbacks := make(map[int]ReloadCallback, len(h.callbacks))
	for k, v := range h.callbacks {
		callbacks[k] = v
	}
	h.mu.RUnlock()

	for boundaryType, callback := range callbacks {
		if err := callback(ctx, configVersion); err != nil {
			h.logger.Warn("业务层reload回调失败",
				zap.Int("boundary_type", boundaryType),
				zap.Int64("config_version", configVersion),
				zap.Error(err),
			)
		}
	}

	// 4. 更新当前版本号
	h.currentVersion = configVersion

	h.logger.Info("配置热更完成",
		zap.Int64("config_version", configVersion),
		zap.Duration("reload_duration", time.Since(startTime)),
	)

	return nil
}

// Run 启动etcd watch goroutine，通过context控制生命周期（规范9goroutine安全）。
// watchCh为etcd配置变更事件channel，实际由infrastructure层注入。
func (h *ConfigHotReloader) Run(ctx context.Context, watchCh <-chan ConfigChangeEvent) {
	for {
		select {
		case <-ctx.Done():
			h.logger.Info("配置热更watcher停止")
			return
		case event := <-watchCh:
			if err := h.HandleReload(ctx, event.ConfigPack, event.ConfigVersion); err != nil {
				h.logger.Error("配置热更处理失败",
					zap.Int64("config_version", event.ConfigVersion),
					zap.Error(err),
				)
			}
		}
	}
}

// CurrentVersion 返回当前配置版本号。
func (h *ConfigHotReloader) CurrentVersion() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentVersion
}

// ConfigChangeEvent 配置变更事件，由etcd watch产生。
type ConfigChangeEvent struct {
	ConfigPack    map[string]any // 配置包数据
	ConfigVersion int64          // 配置版本号
}
