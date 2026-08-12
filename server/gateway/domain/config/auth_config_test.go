// Package config 用户认证配置值对象，domain层声明，infrastructure层负责加载。
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDefaultAuthConfig 测试默认认证配置值，覆盖spec全部可调阈值的默认值。
func TestDefaultAuthConfig(t *testing.T) {
	cfg := DefaultAuthConfig()

	assert.Equal(t, 4, cfg.UsernameMinLength, "用户名最小长度默认4")
	assert.Equal(t, 20, cfg.UsernameMaxLength, "用户名最大长度默认20")
	assert.Equal(t, 8, cfg.PasswordMinLength, "密码最小长度默认8")
	assert.Equal(t, 32, cfg.PasswordMaxLength, "密码最大长度默认32")
	assert.Equal(t, int64(300000), cfg.SessionTimeoutMs, "会话超时默认5分钟")
	assert.Equal(t, int64(300000), cfg.SessionTTLMs, "会话TTL默认5分钟")
	assert.Equal(t, 5, cfg.LoginFailMaxCount, "登录失败最大次数默认5")
	assert.Equal(t, int64(900000), cfg.LoginLockDurationMs, "登录锁定时长默认15分钟")
	assert.Equal(t, 5, cfg.RegisterRateLimitPerIP, "每IP注册频率默认5")
	assert.Equal(t, 10, cfg.LoginRateLimitPerIP, "每IP登录频率默认10")
	assert.Equal(t, 10, cfg.LoginRateLimitPerAcc, "每账号登录频率默认10")
	assert.True(t, cfg.SingleLoginEnabled, "单点登录默认开启")
	assert.Equal(t, 1, cfg.TokenVersion, "令牌版本号默认1")
	assert.Equal(t, "", cfg.TokenSigningKey, "默认签名密钥为空")
}

// TestDefaultAuthConfig_Immutability 测试每次调用DefaultAuthConfig返回独立副本，防止全局状态污染。
func TestDefaultAuthConfig_Immutability(t *testing.T) {
	cfg1 := DefaultAuthConfig()
	cfg1.UsernameMinLength = 99
	cfg1.TokenSigningKey = "modified"

	cfg2 := DefaultAuthConfig()
	assert.Equal(t, 4, cfg2.UsernameMinLength, "修改cfg1不应影响后续返回的默认配置")
	assert.Equal(t, "", cfg2.TokenSigningKey, "修改cfg1不应影响后续返回的默认配置")
}
