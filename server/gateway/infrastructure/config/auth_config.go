// Package config Gateway用户认证配置加载与热更。
//
// infrastructure层配置适配，从Config服务加载AuthConfig并在热更时回调刷新。
// AuthConfig值对象定义在domain/config层，本包负责加载与热更逻辑。
// 配置项覆盖spec 5.1-5.5全部可调阈值，加载失败时降级为默认值并记录Warn日志。
package config

import (
	"context"
	"sync"

	"go.uber.org/zap"

	domainconfig "insectworld/server/gateway/domain/config"
)

// AuthConfigLoader 认证配置加载器，从Config服务加载并支持热更回调。
type AuthConfigLoader struct {
	mu     sync.RWMutex            // 读写锁，保护配置并发访问
	cfg    domainconfig.AuthConfig // 当前生效的认证配置
	logger *zap.Logger             // 结构化日志
}

// NewAuthConfigLoader 创建认证配置加载器实例，初始使用默认配置。
func NewAuthConfigLoader(logger *zap.Logger) *AuthConfigLoader {
	return &AuthConfigLoader{
		cfg:    domainconfig.DefaultAuthConfig(),
		logger: logger,
	}
}

// Load 从Config服务加载认证配置，加载失败时降级为默认值并记录Warn日志。
//
// ctx用于控制加载超时，configQuery为Config服务查询接口。
// 加载成功后原子替换本地配置，加载失败保留旧配置不变更。
func (l *AuthConfigLoader) Load(ctx context.Context, rawCfg map[string]any) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	newCfg := domainconfig.DefaultAuthConfig()
	if rawCfg == nil {
		l.logger.Warn("认证配置加载为空，降级使用默认配置")
		l.cfg = newCfg
		return nil
	}

	if v, ok := rawCfg["username_min_length"].(int); ok {
		newCfg.UsernameMinLength = v
	}
	if v, ok := rawCfg["username_max_length"].(int); ok {
		newCfg.UsernameMaxLength = v
	}
	if v, ok := rawCfg["password_min_length"].(int); ok {
		newCfg.PasswordMinLength = v
	}
	if v, ok := rawCfg["password_max_length"].(int); ok {
		newCfg.PasswordMaxLength = v
	}
	if v, ok := rawCfg["session_timeout_ms"].(int64); ok {
		newCfg.SessionTimeoutMs = v
	}
	if v, ok := rawCfg["session_ttl_ms"].(int64); ok {
		newCfg.SessionTTLMs = v
	}
	if v, ok := rawCfg["login_fail_max_count"].(int); ok {
		newCfg.LoginFailMaxCount = v
	}
	if v, ok := rawCfg["login_lock_duration_ms"].(int64); ok {
		newCfg.LoginLockDurationMs = v
	}
	if v, ok := rawCfg["register_rate_limit_per_ip"].(int); ok {
		newCfg.RegisterRateLimitPerIP = v
	}
	if v, ok := rawCfg["login_rate_limit_per_ip"].(int); ok {
		newCfg.LoginRateLimitPerIP = v
	}
	if v, ok := rawCfg["login_rate_limit_per_account"].(int); ok {
		newCfg.LoginRateLimitPerAcc = v
	}
	if v, ok := rawCfg["single_login_enabled"].(bool); ok {
		newCfg.SingleLoginEnabled = v
	}
	if v, ok := rawCfg["token_version"].(int); ok {
		newCfg.TokenVersion = v
	}
	if v, ok := rawCfg["token_signing_key"].(string); ok {
		newCfg.TokenSigningKey = v
	}

	l.cfg = newCfg
	l.logger.Info("认证配置加载成功",
		zap.Int("username_min_length", newCfg.UsernameMinLength),
		zap.Int("username_max_length", newCfg.UsernameMaxLength),
		zap.Int("password_min_length", newCfg.PasswordMinLength),
		zap.Int("password_max_length", newCfg.PasswordMaxLength),
		zap.Int64("session_timeout_ms", newCfg.SessionTimeoutMs),
		zap.Int("login_fail_max_count", newCfg.LoginFailMaxCount),
		zap.Int64("login_lock_duration_ms", newCfg.LoginLockDurationMs),
		zap.Bool("single_login_enabled", newCfg.SingleLoginEnabled),
	)
	return nil
}

// Get 获取当前生效的认证配置（返回值拷贝，避免外部修改）。
func (l *AuthConfigLoader) Get() domainconfig.AuthConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.cfg
}
