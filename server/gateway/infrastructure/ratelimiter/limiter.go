// Package ratelimiter Gateway服务频率限制器，基于令牌桶算法实现请求限流。
//
// infrastructure层技术适配，实现domain层RateLimiter接口。
package ratelimiter

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// Limiter 令牌桶限流器，按玩家维度限流。
type Limiter struct {
	buckets map[int64]*tokenBucket // 限流桶池，key=玩家ID
	rate    int                    // 令牌生成速率，每秒生成令牌数
	burst   int                    // 桶容量，最大突发请求数
	mu      sync.RWMutex           // 读写锁，保护桶池并发访问
	logger  *zap.Logger            // 结构化日志
}

// tokenBucket 令牌桶。
type tokenBucket struct {
	tokens   int       // 当前令牌数
	lastTime time.Time // 上次令牌生成时间
}

// NewLimiter 创建限流器实例。
func NewLimiter(rate, burst int, logger *zap.Logger) *Limiter {
	return &Limiter{
		buckets: make(map[int64]*tokenBucket),
		rate:    rate,
		burst:   burst,
		logger:  logger,
	}
}

// Allow 判定玩家请求是否允许通过，消耗一个令牌。
func (l *Limiter) Allow(playerID int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[playerID]
	if !ok {
		bucket = &tokenBucket{
			tokens:   l.burst,
			lastTime: time.Now(),
		}
		l.buckets[playerID] = bucket
	}

	now := time.Now()
	elapsed := now.Sub(bucket.lastTime).Seconds()
	bucket.tokens += int(elapsed * float64(l.rate))
	if bucket.tokens > l.burst {
		bucket.tokens = l.burst
	}
	bucket.lastTime = now

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	return false
}
