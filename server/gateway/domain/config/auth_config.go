// Package config 用户认证配置值对象，domain层声明，infrastructure层负责加载。
//
// AuthConfig为纯数据值对象，无行为方法，从infrastructure/config加载后注入application层。
// 所有数值字段用整型（规范8），时间字段用int64毫秒，TokenSigningKey不入日志（规范7脱敏）。
package config

// AuthConfig 用户认证配置值对象，覆盖注册/登录/登出/心跳/鉴权全部可调阈值。
//
// 从infrastructure/config.AuthConfig提升到domain层，避免application层直接依赖infrastructure（规范3 DDD依赖方向）。
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
