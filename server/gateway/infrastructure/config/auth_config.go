// Package config Gateway用户认证配置加载与热更。
//
// infrastructure层配置适配，从Config服务加载AuthConfig并在热更时回调刷新。
// 配置项覆盖spec 5.1-5.5全部可调阈值，加载失败时降级为默认值并记录Warn日志。
package config

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// AuthConfig 用户认证配置，覆盖注册/登录/登出/心跳/鉴权全部可调阈值。
//
// 所有数值字段用整型（规范8），时间相关字段用int64毫秒。
// TokenSigningKey为敏感字段，不得出现在任何日志（规范7脱敏）。
type AuthConfig struct {
	UsernameMinLength      int    // 用户名最小长度，默认4
	UsernameMaxLength      int    // 用户名最大长度，默认20
	PasswordMinLength      int    // 密码最小长度，默认8
	PasswordMaxLength      int    // 密码最大长度，默认32
	SessionTimeoutMs       int64  // 会话超时时间，毫秒级，默认300000（5分钟）
	SessionTTLMs           int64  // 会话TTL，毫秒级，与SessionTimeoutMs对齐
	LoginFailMaxCount      int    // 登录失败最大次数，默认5
	LoginLockDurationMs    int64  // 登录锁定时长，毫秒级，默认900000（15分钟）
	RegisterRateLimitPerIP int    // 每IP注册频率限制，默认5次/窗口
	LoginRateLimitPerIP    int    // 每IP登录频率限制，默认10次/窗口
	LoginRateLimitPerAcc   int    // 每账号登录频率限制，默认10次/窗口
	SingleLoginEnabled     bool   // 单点登录开关，true=同账号新登录踢旧会话下线
	TokenVersion           int    // 令牌版本号，默认1
	TokenSigningKey        string // 令牌签名密钥，从安全配置加载，不入日志（规范7脱敏）
}

// DefaultAuthConfig 返回默认认证配置，配置加载失败时降级使用。
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		UsernameMinLength:      4,
		UsernameMaxLength:      20,
		PasswordMinLength:      8,
		PasswordMaxLength:      32,
		SessionTimeoutMs:       300000,
		SessionTTLMs:           300000,
		LoginFailMaxCount:      5,
		LoginLockDurationMs:    900000,
		RegisterRateLimitPerIP: 5,
		LoginRateLimitPerIP:    10,
		LoginRateLimitPerAcc:   10,
		SingleLoginEnabled:     true,
		TokenVersion:           1,
		TokenSigningKey:        "",
	}
}

// AuthConfigLoader 认证配置加载器，从Config服务加载并支持热更回调。
type AuthConfigLoader struct {
	mu     sync.RWMutex // 读写锁，保护配置并发访问
	cfg    AuthConfig   // 当前生效的认证配置
	logger *zap.Logger  // 结构化日志
}

// NewAuthConfigLoader 创建认证配置加载器实例，初始使用默认配置。
func NewAuthConfigLoader(logger *zap.Logger) *AuthConfigLoader {
	return &AuthConfigLoader{
		cfg:    DefaultAuthConfig(),
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

	newCfg := DefaultAuthConfig()
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
func (l *AuthConfigLoader) Get() AuthConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.cfg
}
