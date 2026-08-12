// Package config Gateway用户认证配置加载与热更。
package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainconfig "insectworld/server/gateway/domain/config"

	"go.uber.org/zap"
)

// TestAuthConfigLoader_DefaultOnNew 测试新建加载器初始使用默认配置。
func TestAuthConfigLoader_DefaultOnNew(t *testing.T) {
	logger := zap.NewNop()
	loader := NewAuthConfigLoader(logger)
	cfg := loader.Get()
	defaultCfg := domainconfig.DefaultAuthConfig()
	assert.Equal(t, defaultCfg, cfg, "新建加载器应返回默认配置")
}

// TestAuthConfigLoader_LoadNil 测试加载nil配置降级为默认值。
func TestAuthConfigLoader_LoadNil(t *testing.T) {
	logger := zap.NewNop()
	loader := NewAuthConfigLoader(logger)
	require.NoError(t, loader.Load(context.Background(), nil))

	cfg := loader.Get()
	assert.Equal(t, domainconfig.DefaultAuthConfig(), cfg)
}

// TestAuthConfigLoader_LoadFull 测试加载完整配置覆盖全部字段。
func TestAuthConfigLoader_LoadFull(t *testing.T) {
	logger := zap.NewNop()
	loader := NewAuthConfigLoader(logger)

	rawCfg := map[string]any{
		"username_min_length":          6,
		"username_max_length":          24,
		"password_min_length":          10,
		"password_max_length":          40,
		"session_timeout_ms":           int64(600000),
		"session_ttl_ms":               int64(600000),
		"login_fail_max_count":         3,
		"login_lock_duration_ms":       int64(1800000),
		"register_rate_limit_per_ip":   8,
		"login_rate_limit_per_ip":      20,
		"login_rate_limit_per_account": 15,
		"single_login_enabled":         false,
		"token_version":                2,
		"token_signing_key":            "my-secret-key",
	}
	require.NoError(t, loader.Load(context.Background(), rawCfg))

	cfg := loader.Get()
	assert.Equal(t, 6, cfg.UsernameMinLength)
	assert.Equal(t, 24, cfg.UsernameMaxLength)
	assert.Equal(t, 10, cfg.PasswordMinLength)
	assert.Equal(t, 40, cfg.PasswordMaxLength)
	assert.Equal(t, int64(600000), cfg.SessionTimeoutMs)
	assert.Equal(t, int64(600000), cfg.SessionTTLMs)
	assert.Equal(t, 3, cfg.LoginFailMaxCount)
	assert.Equal(t, int64(1800000), cfg.LoginLockDurationMs)
	assert.Equal(t, 8, cfg.RegisterRateLimitPerIP)
	assert.Equal(t, 20, cfg.LoginRateLimitPerIP)
	assert.Equal(t, 15, cfg.LoginRateLimitPerAcc)
	assert.False(t, cfg.SingleLoginEnabled)
	assert.Equal(t, 2, cfg.TokenVersion)
	assert.Equal(t, "my-secret-key", cfg.TokenSigningKey)
}

// TestAuthConfigLoader_LoadPartial 测试加载部分配置，未提供字段保留默认值。
func TestAuthConfigLoader_LoadPartial(t *testing.T) {
	logger := zap.NewNop()
	loader := NewAuthConfigLoader(logger)

	rawCfg := map[string]any{
		"username_min_length":  8,
		"single_login_enabled": false,
	}
	require.NoError(t, loader.Load(context.Background(), rawCfg))

	cfg := loader.Get()
	assert.Equal(t, 8, cfg.UsernameMinLength, "提供的字段应覆盖默认值")
	assert.False(t, cfg.SingleLoginEnabled, "提供的字段应覆盖默认值")
	// 未提供字段应保留默认值
	assert.Equal(t, 20, cfg.UsernameMaxLength, "未提供字段应保留默认值")
	assert.Equal(t, int64(300000), cfg.SessionTimeoutMs)
	assert.Equal(t, 1, cfg.TokenVersion)
}

// TestAuthConfigLoader_LoadWrongType 测试配置值类型不匹配时保留默认值（类型断言失败降级）。
func TestAuthConfigLoader_LoadWrongType(t *testing.T) {
	logger := zap.NewNop()
	loader := NewAuthConfigLoader(logger)

	// 提供错误类型的值（string而非int）
	rawCfg := map[string]any{
		"username_min_length": "not-an-int",
		"session_timeout_ms":  "not-int64",
	}
	require.NoError(t, loader.Load(context.Background(), rawCfg))

	cfg := loader.Get()
	// 类型不匹配应保留默认值
	assert.Equal(t, 4, cfg.UsernameMinLength)
	assert.Equal(t, int64(300000), cfg.SessionTimeoutMs)
}

// TestAuthConfigLoader_ReLoad 测试多次加载原子替换配置。
func TestAuthConfigLoader_ReLoad(t *testing.T) {
	logger := zap.NewNop()
	loader := NewAuthConfigLoader(logger)

	// 第一次加载
	require.NoError(t, loader.Load(context.Background(), map[string]any{
		"username_min_length": 6,
	}))
	assert.Equal(t, 6, loader.Get().UsernameMinLength)

	// 第二次加载，应替换而非合并
	require.NoError(t, loader.Load(context.Background(), map[string]any{
		"username_max_length": 30,
	}))
	cfg := loader.Get()
	assert.Equal(t, 4, cfg.UsernameMinLength, "第二次加载未提供字段应恢复默认值")
	assert.Equal(t, 30, cfg.UsernameMaxLength)
}
