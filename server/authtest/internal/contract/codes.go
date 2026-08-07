// Package contract 测试端契约定义，与Gateway服务消息格式对齐，独立维护避免跨服务依赖。
//
// 对齐：server/gateway/domain/errors/codes.go
// 错误码变更时需人工同步本文件与Gateway源文件。
package contract

// 错误码常量，与Gateway domain/errors/codes.go完全对齐。
// 用于端到端断言时比较AuthResponse.ErrorCode。
const (
	ErrCodeTokenInvalid     = 17001 // Token无效
	ErrCodeRateLimited      = 17002 // 请求频率受限
	ErrCodeConnectionClosed = 17003 // 连接已关闭

	// 注册相关错误码 17010-17014
	ErrCodeInvalidUsernameFormat   = 17010 // 用户名格式不合法
	ErrCodeInvalidPasswordStrength = 17011 // 密码强度不足
	ErrCodeUsernameAlreadyExists   = 17012 // 用户名已存在
	ErrCodeRegisterRateLimited     = 17013 // 注册频率超限
	ErrCodeRegisterInternalError   = 17014 // 注册内部错误

	// 登录相关错误码 17020-17025
	ErrCodeAccountNotFound    = 17020 // 账号不存在
	ErrCodePasswordIncorrect  = 17021 // 密码错误
	ErrCodeAccountBanned      = 17022 // 账号已被封禁
	ErrCodeAccountLocked      = 17023 // 账号已被锁定
	ErrCodeLoginRateLimited   = 17024 // 登录频率超限
	ErrCodeLoginInternalError = 17025 // 登录内部错误

	// 登出相关错误码 17030
	ErrCodeLogoutInternalError = 17030 // 登出内部错误

	// 鉴权相关错误码 17040-17042
	ErrCodeTokenMissing = 17040 // 令牌缺失
	ErrCodeTokenExpired = 17041 // 令牌已过期
	ErrCodeNotLoggedIn  = 17042 // 未登录

	// 基础设施故障错误码 17050-17059
	ErrCodeAccountRepoUnavailable    = 17050 // 账号仓储不可用
	ErrCodeSessionRepoUnavailable    = 17051 // 会话仓储不可用
	ErrCodeSessionNotFound           = 17052 // 会话不存在
	ErrCodeAccountNotFoundSentinel   = 17053 // 账号不存在哨兵
	ErrCodeTokenSignerUnavailable    = 17054 // 令牌签发器不可用
	ErrCodeTokenBlacklistUnavailable = 17055 // 令牌黑名单不可用
	ErrCodeFailureTrackerUnavailable = 17056 // 失败计数器不可用
	ErrCodeAuditLogUnavailable       = 17057 // 审计日志不可用
	ErrCodeIDGenClockBack            = 17058 // 雪花ID时钟回拨
	ErrCodePasswordHashFailed        = 17059 // 密码哈希失败
)
