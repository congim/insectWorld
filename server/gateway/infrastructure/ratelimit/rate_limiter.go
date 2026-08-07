// Package ratelimit 多维度限流器infrastructure层实现，整合现有令牌桶算法。
//
// RateLimiterImpl支持IP/账号名/playerID三维度限流（spec 4.3 安全性6-7），
// 复用现有Limiter的令牌桶算法（design整合步骤6），按复合键"dimension:key"索引桶。
package ratelimit

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RateConfig 限流配置，按维度配置不同的速率与桶容量。
type RateConfig struct {
	Rate  int // 令牌生成速率，每秒生成令牌数
	Burst int // 桶容量，最大突发请求数
}

// RateLimiterImpl 多维度令牌桶限流器，实现domain层RateLimiter接口。
//
// 按复合键"dimension:key"索引令牌桶，从dimConfigs读取对应dimension的rate/burst。
// 故障时降级为"允许"（避免限流器故障导致全部请求被拒），无error返回。
type RateLimiterImpl struct {
	buckets    map[string]*tokenBucket // 限流桶池，key=复合键"dimension:key"
	dimConfigs map[string]RateConfig   // 各维度限流配置，key=dimension
	mu         sync.RWMutex            // 读写锁，保护桶池并发访问
	logger     *zap.Logger             // 结构化日志
}

// tokenBucket 令牌桶，复用现有Limiter的算法。
type tokenBucket struct {
	tokens   int       // 当前令牌数
	lastTime time.Time // 上次令牌生成时间
}

// NewRateLimiterImpl 创建多维度限流器实例。
//
// dimConfigs为各维度限流配置，key为dimension（如"register:ip"/"login:ip"/"login:account"）。
func NewRateLimiterImpl(dimConfigs map[string]RateConfig, logger *zap.Logger) *RateLimiterImpl {
	return &RateLimiterImpl{
		buckets:    make(map[string]*tokenBucket),
		dimConfigs: dimConfigs,
		logger:     logger,
	}
}

// Allow 判断请求是否允许通过，消耗一个令牌。
//
// dimension为限流维度，key为维度下的具体键值。
// 返回true表示允许，false表示超限拒绝。限流器故障降级为"允许"。
func (l *RateLimiterImpl) Allow(ctx context.Context, dimension string, key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cfg, ok := l.dimConfigs[dimension]
	if !ok {
		return true
	}

	compositeKey := dimension + ":" + key
	bucket, ok := l.buckets[compositeKey]
	if !ok {
		bucket = &tokenBucket{
			tokens:   cfg.Burst,
			lastTime: time.Now(),
		}
		l.buckets[compositeKey] = bucket
	}

	now := time.Now()
	elapsed := now.Sub(bucket.lastTime).Seconds()
	bucket.tokens += int(elapsed * float64(cfg.Rate))
	if bucket.tokens > cfg.Burst {
		bucket.tokens = cfg.Burst
	}
	bucket.lastTime = now

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	return false
}
