// Package errors Gateway服务错误码定义。
// 错误码区间 17000-17999 为 Gateway 服务保留。
package errors

import "fmt"

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

// GatewayError Gateway服务错误。
type GatewayError struct {
	Code int    // 错误码
	Msg  string // 错误消息，中文文案（规范5）
}

// Error 实现error接口。
func (e *GatewayError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

var (
	ErrTokenInvalid     = &GatewayError{ErrCodeTokenInvalid, "Token无效"}
	ErrRateLimited      = &GatewayError{ErrCodeRateLimited, "请求频率受限"}
	ErrConnectionClosed = &GatewayError{ErrCodeConnectionClosed, "连接已关闭"}

	// 注册相关错误变量
	ErrInvalidUsernameFormat   = &GatewayError{ErrCodeInvalidUsernameFormat, "用户名格式不合法"}
	ErrInvalidPasswordStrength = &GatewayError{ErrCodeInvalidPasswordStrength, "密码强度不足"}
	ErrUsernameAlreadyExists   = &GatewayError{ErrCodeUsernameAlreadyExists, "用户名已存在"}
	ErrRegisterRateLimited     = &GatewayError{ErrCodeRegisterRateLimited, "注册频率超限"}
	ErrRegisterInternalError   = &GatewayError{ErrCodeRegisterInternalError, "注册内部错误"}

	// 登录相关错误变量
	ErrAccountNotFound    = &GatewayError{ErrCodeAccountNotFound, "账号不存在"}
	ErrPasswordIncorrect  = &GatewayError{ErrCodePasswordIncorrect, "密码错误"}
	ErrAccountBanned      = &GatewayError{ErrCodeAccountBanned, "账号已被封禁"}
	ErrAccountLocked      = &GatewayError{ErrCodeAccountLocked, "账号已被锁定，请稍后再试"}
	ErrLoginRateLimited   = &GatewayError{ErrCodeLoginRateLimited, "登录频率超限"}
	ErrLoginInternalError = &GatewayError{ErrCodeLoginInternalError, "登录内部错误"}

	// 登出相关错误变量
	ErrLogoutInternalError = &GatewayError{ErrCodeLogoutInternalError, "登出内部错误"}

	// 鉴权相关错误变量
	ErrTokenMissing = &GatewayError{ErrCodeTokenMissing, "令牌缺失"}
	ErrTokenExpired = &GatewayError{ErrCodeTokenExpired, "令牌已过期"}
	ErrNotLoggedIn  = &GatewayError{ErrCodeNotLoggedIn, "未登录"}

	// 基础设施故障错误变量
	ErrAccountRepoUnavailable    = &GatewayError{ErrCodeAccountRepoUnavailable, "账号仓储不可用"}
	ErrSessionRepoUnavailable    = &GatewayError{ErrCodeSessionRepoUnavailable, "会话仓储不可用"}
	ErrSessionNotFound           = &GatewayError{ErrCodeSessionNotFound, "会话不存在"}
	ErrAccountNotFoundSentinel   = &GatewayError{ErrCodeAccountNotFoundSentinel, "账号不存在"}
	ErrTokenSignerUnavailable    = &GatewayError{ErrCodeTokenSignerUnavailable, "令牌签发器不可用"}
	ErrTokenBlacklistUnavailable = &GatewayError{ErrCodeTokenBlacklistUnavailable, "令牌黑名单不可用"}
	ErrFailureTrackerUnavailable = &GatewayError{ErrCodeFailureTrackerUnavailable, "失败计数器不可用"}
	ErrAuditLogUnavailable       = &GatewayError{ErrCodeAuditLogUnavailable, "审计日志不可用"}
	ErrIDGenClockBack            = &GatewayError{ErrCodeIDGenClockBack, "雪花ID时钟回拨"}
	ErrPasswordHashFailed        = &GatewayError{ErrCodePasswordHashFailed, "密码哈希失败"}
)
