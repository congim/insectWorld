// Package publisher 提供Outbox租约轮询、发布和失败退避编排。
package publisher

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"insectworld/server/shared/pkg/eventbus"
)

const maxFailureLength = 512

// Config 是Outbox发布器运行参数。
type Config struct {
	EventTypes     []string      // EventTypes 是当前发布器负责投递的事件类型白名单
	BatchSize      int           // BatchSize 是每轮最大领取数量
	PollInterval   time.Duration // PollInterval 是空闲轮询间隔
	LeaseDuration  time.Duration // LeaseDuration 是单批投递租约时长
	BaseRetryDelay time.Duration // BaseRetryDelay 是首次失败退避时长
	MaxRetryDelay  time.Duration // MaxRetryDelay 是失败退避上限
}

// Publisher 是支持多实例租约和本实例防重入的Outbox发布器。
type Publisher struct {
	repository eventbus.OutboxRepository // Outbox事务仓储
	bus        eventbus.EventBus         // 事件发布目标
	config     Config                    // 发布器运行参数
	logger     *zap.Logger               // 结构化日志器
	now        func() time.Time          // 可替换时钟，用于确定性测试
	mu         sync.Mutex                // 防止同一实例PollOnce并发重入
	metrics    runtimeMetrics            // 进程内投递健康指标
}

type runtimeMetrics struct {
	claimedTotal       atomic.Int64 // 累计领取事件数量
	publishedTotal     atomic.Int64 // 累计成功发布事件数量
	failedTotal        atomic.Int64 // 累计发布失败事件数量
	inFlight           atomic.Int64 // 当前已领取但尚未完成状态更新的事件数量
	lastPollDurationMs atomic.Int64 // 最近一轮轮询耗时，单位毫秒
}

// Metrics 是Outbox发布器的只读健康指标快照。
type Metrics struct {
	ClaimedTotal       int64 // 累计领取事件数量
	PublishedTotal     int64 // 累计成功发布事件数量，可与失败数计算成功率
	FailedTotal        int64 // 累计发布失败事件数量，反映一致性重试压力
	InFlight           int64 // 当前投递中数量，用于发现卡住或租约积压
	LastPollDurationMs int64 // 最近一轮轮询耗时，单位毫秒
}

// New 创建Outbox发布器并校验运行参数。
func New(repository eventbus.OutboxRepository, bus eventbus.EventBus, config Config, logger *zap.Logger) (*Publisher, error) {
	if repository == nil || bus == nil || len(config.EventTypes) == 0 || config.BatchSize <= 0 || config.PollInterval <= 0 || config.LeaseDuration <= 0 || config.BaseRetryDelay <= 0 || config.MaxRetryDelay < config.BaseRetryDelay {
		return nil, fmt.Errorf("Outbox发布器参数非法")
	}
	for _, eventType := range config.EventTypes {
		if eventType == "" {
			return nil, fmt.Errorf("Outbox发布器事件类型不能为空")
		}
	}
	config.EventTypes = append([]string(nil), config.EventTypes...)
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Publisher{repository: repository, bus: bus, config: config, logger: logger, now: time.Now}, nil
}

// Run 持续轮询直到context取消；单轮失败会记录日志并等待下一轮恢复。
func (p *Publisher) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()
	for {
		if err := p.PollOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			p.logger.Error("Outbox轮询发布失败", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// PollOnce 领取并尝试发布一批事件，返回本轮全部错误的聚合结果。
func (p *Publisher) PollOnce(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	startedAt := time.Now()
	defer func() { p.metrics.lastPollDurationMs.Store(time.Since(startedAt).Milliseconds()) }()
	nowMs := p.now().UnixMilli()
	leaseUntil := nowMs + p.config.LeaseDuration.Milliseconds()
	records, err := p.repository.ClaimPending(ctx, nowMs, p.config.EventTypes, p.config.BatchSize, leaseUntil)
	if err != nil {
		return err
	}
	p.metrics.claimedTotal.Add(int64(len(records)))
	p.metrics.inFlight.Add(int64(len(records)))
	var publishErrors []error
	for _, record := range records {
		event := eventbus.DomainEvent{EventID: record.EventID, EventType: record.EventType, AggregateID: record.AggregateID, Version: record.Version, Timestamp: record.CreateTime, Payload: record.Payload}
		if err := p.bus.Publish(ctx, event); err != nil {
			p.metrics.failedTotal.Add(1)
			nextAttempt := nowMs + retryDelay(p.config.BaseRetryDelay, p.config.MaxRetryDelay, record.RetryCount).Milliseconds()
			failure := err.Error()
			if len(failure) > maxFailureLength {
				failure = failure[:maxFailureLength]
			}
			if markErr := p.repository.MarkFailed(ctx, record.EventID, nextAttempt, failure); markErr != nil {
				publishErrors = append(publishErrors, errors.Join(err, markErr))
			} else {
				publishErrors = append(publishErrors, err)
			}
			p.metrics.inFlight.Add(-1)
			continue
		}
		if err := p.repository.MarkPublished(ctx, record.EventID, p.now().UnixMilli()); err != nil {
			p.metrics.failedTotal.Add(1)
			publishErrors = append(publishErrors, err)
		} else {
			p.metrics.publishedTotal.Add(1)
		}
		p.metrics.inFlight.Add(-1)
	}
	return errors.Join(publishErrors...)
}

// Metrics 返回无锁一致读取的投递指标快照。
func (p *Publisher) Metrics() Metrics {
	return Metrics{
		ClaimedTotal:       p.metrics.claimedTotal.Load(),
		PublishedTotal:     p.metrics.publishedTotal.Load(),
		FailedTotal:        p.metrics.failedTotal.Load(),
		InFlight:           p.metrics.inFlight.Load(),
		LastPollDurationMs: p.metrics.lastPollDurationMs.Load(),
	}
}

func retryDelay(base time.Duration, maximum time.Duration, retryCount int) time.Duration {
	if retryCount <= 0 {
		return base
	}
	if retryCount >= 62 || int64(base) > math.MaxInt64>>retryCount {
		return maximum
	}
	delay := base << retryCount
	if delay > maximum {
		return maximum
	}
	return delay
}
