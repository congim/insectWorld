// Package ratelimit 限流能力接口，domain层声明，infrastructure层实现令牌桶适配。
//
// domain层零外部依赖（规范3），支持IP/账号名/playerID三维度限流（spec 4.3 安全性6-7）。
package ratelimit

import "context"

// RateLimiter 限流能力接口，infrastructure层实现令牌桶适配。
//
// 接口在domain层声明（规范3 DDD），保证domain层不依赖第三方限流包。
// dimension为限流维度（如"register:ip"/"login:ip"/"login:account"），key为维度下的具体键值。
type RateLimiter interface {
	// Allow 判断请求是否允许通过。
	// dimension为限流维度，key为维度下的具体键值（如IP地址、账号名）。
	// 返回true表示允许，false表示超限拒绝。限流器故障降级为"允许"（避免故障导致全部请求被拒）。
	Allow(ctx context.Context, dimension string, key string) bool
}
