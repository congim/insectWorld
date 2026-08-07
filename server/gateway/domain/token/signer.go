// Package token 访问令牌值对象与签发/校验能力接口。
package token

import "context"

// TokenSigner 令牌签发与校验能力接口，infrastructure层实现HMAC-SHA256适配。
//
// 接口在domain层声明（规范3 DDD），保证domain层不依赖第三方加密包。
// HMAC-SHA256签名防伪造（spec 4.3 安全性3），签名密钥不入日志（规范7脱敏）。
type TokenSigner interface {
	// Sign 对令牌负载计算签名，返回"payload.signature"格式的令牌字符串。
	// 签名密钥未配置时返回ErrTokenSignerUnavailable。
	Sign(ctx context.Context, payload TokenPayload) (string, error)

	// Verify 校验令牌字符串的签名与格式，返回令牌负载。
	// 签名不匹配或格式错误返回ErrTokenInvalid。
	Verify(ctx context.Context, tokenStr string) (TokenPayload, error)
}

// TokenBlacklist 令牌黑名单能力接口，infrastructure层实现Redis适配。
//
// 用于登出/踢下线时使令牌即刻失效（spec 5.3.1 规则3）。
// Redis故障时IsInvalid返回(false, ErrTokenBlacklistUnavailable)，鉴权层降级为"不拒绝"。
type TokenBlacklist interface {
	// Invalidate 将令牌加入黑名单，设置TTL为令牌剩余有效期，自动清理。
	Invalidate(ctx context.Context, playerID int64, tokenVersion int, remainingTTLSeconds int64) error

	// IsInvalid 查询令牌是否在黑名单中。
	// 返回true表示已失效，Redis故障返回(false, ErrTokenBlacklistUnavailable)降级为"不拒绝"。
	IsInvalid(ctx context.Context, playerID int64, tokenVersion int) (bool, error)
}
