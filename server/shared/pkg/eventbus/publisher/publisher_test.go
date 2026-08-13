// Package publisher 提供Outbox租约轮询、发布和失败退避编排。
package publisher

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"insectworld/server/shared/pkg/eventbus"
)

type repositoryStub struct {
	records       []eventbus.OutboxRecord // 待领取记录
	published     []string                // 已标记发布事件ID
	failed        []string                // 已标记失败事件ID
	nextAttemptMs int64                   // 最近一次失败重试时间
	activeClaims  int                     // 当前领取调用数量
	maxActive     int                     // 最大并发领取调用数量
	mu            sync.Mutex              // 保护测试状态
}

func (r *repositoryStub) Save(_ context.Context, _ eventbus.OutboxRecord) error { return nil }
func (r *repositoryStub) MarkPublished(_ context.Context, eventID string, _ int64) error {
	r.published = append(r.published, eventID)
	return nil
}
func (r *repositoryStub) GetPending(_ context.Context, _ int) ([]eventbus.OutboxRecord, error) {
	return nil, nil
}
func (r *repositoryStub) MarkFailed(_ context.Context, eventID string, nextAttemptMs int64, _ string) error {
	r.failed = append(r.failed, eventID)
	r.nextAttemptMs = nextAttemptMs
	return nil
}
func (r *repositoryStub) ClaimPending(_ context.Context, _ int64, _ []string, _ int, _ int64) ([]eventbus.OutboxRecord, error) {
	r.mu.Lock()
	r.activeClaims++
	if r.activeClaims > r.maxActive {
		r.maxActive = r.activeClaims
	}
	r.mu.Unlock()
	time.Sleep(time.Millisecond)
	r.mu.Lock()
	r.activeClaims--
	r.mu.Unlock()
	return r.records, nil
}

type busStub struct {
	events []eventbus.DomainEvent // 已收到事件
	err    error                  // 发布返回错误
}

func (b *busStub) Publish(_ context.Context, event eventbus.DomainEvent) error {
	b.events = append(b.events, event)
	return b.err
}
func (b *busStub) Subscribe(_ context.Context, _ string, _ eventbus.EventHandler) error { return nil }

func testConfig() Config {
	return Config{EventTypes: []string{"test"}, BatchSize: 10, PollInterval: time.Second, LeaseDuration: time.Minute, BaseRetryDelay: time.Second, MaxRetryDelay: time.Minute}
}

// TestPollOnceMarksPublished 验证成功交付后Outbox才标记已发布。
func TestPollOnceMarksPublished(t *testing.T) {
	t.Parallel()
	repository := &repositoryStub{records: []eventbus.OutboxRecord{{EventID: "event-1", EventType: "test", AggregateID: 1, Version: 2, CreateTime: 1000}}}
	bus := &busStub{}
	publisher, err := New(repository, bus, testConfig(), nil)
	require.NoError(t, err)
	publisher.now = func() time.Time { return time.UnixMilli(5000) }
	require.NoError(t, publisher.PollOnce(context.Background()))
	assert.Equal(t, []string{"event-1"}, repository.published)
	require.Len(t, bus.events, 1)
	assert.Equal(t, 2, bus.events[0].Version)
	metrics := publisher.Metrics()
	assert.Equal(t, int64(1), metrics.ClaimedTotal)
	assert.Equal(t, int64(1), metrics.PublishedTotal)
	assert.Zero(t, metrics.FailedTotal)
	assert.Zero(t, metrics.InFlight)
}

// TestPollOnceMarksFailedWithBackoff 验证消费者失败时保留事件并设置指数退避。
func TestPollOnceMarksFailedWithBackoff(t *testing.T) {
	t.Parallel()
	repository := &repositoryStub{records: []eventbus.OutboxRecord{{EventID: "event-1", EventType: "test", RetryCount: 2}}}
	bus := &busStub{err: errors.New("consumer failed")}
	publisher, err := New(repository, bus, testConfig(), nil)
	require.NoError(t, err)
	publisher.now = func() time.Time { return time.UnixMilli(5000) }
	err = publisher.PollOnce(context.Background())
	require.Error(t, err)
	assert.Equal(t, []string{"event-1"}, repository.failed)
	assert.Equal(t, int64(9000), repository.nextAttemptMs)
	assert.Empty(t, repository.published)
	metrics := publisher.Metrics()
	assert.Equal(t, int64(1), metrics.FailedTotal)
	assert.Zero(t, metrics.InFlight)
}

// TestPollOncePreventsLocalReentry 验证同一发布器并发触发时领取过程不会重入。
func TestPollOncePreventsLocalReentry(t *testing.T) {
	t.Parallel()
	repository := &repositoryStub{}
	publisher, err := New(repository, &busStub{}, testConfig(), nil)
	require.NoError(t, err)
	var waitGroup sync.WaitGroup
	for range 8 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			require.NoError(t, publisher.PollOnce(context.Background()))
		}()
	}
	waitGroup.Wait()
	assert.Equal(t, 1, repository.maxActive)
}

// TestRetryDelayCapsAtMaximum 验证指数退避不会发生溢出或超过上限。
func TestRetryDelayCapsAtMaximum(t *testing.T) {
	t.Parallel()
	assert.Equal(t, time.Second, retryDelay(time.Second, time.Minute, 0))
	assert.Equal(t, 4*time.Second, retryDelay(time.Second, time.Minute, 2))
	assert.Equal(t, time.Minute, retryDelay(time.Second, time.Minute, 62))
}
