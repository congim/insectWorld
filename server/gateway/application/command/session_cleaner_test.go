package command

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainconfig "insectworld/server/gateway/domain/config"
	domainsession "insectworld/server/gateway/domain/session"

	"go.uber.org/zap"
)

// newTestSessionCleaner 构造测试用SessionTimeoutCleaner实例。
func newTestSessionCleaner(
	sessionRepo *mockSessionRepo,
	eventBus *mockEventBus,
	cfg domainconfig.AuthConfig,
) *SessionTimeoutCleaner {
	logger := zap.NewNop()
	return NewSessionTimeoutCleaner(sessionRepo, eventBus, cfg, logger)
}

// TestSessionTimeoutCleaner_RunAndClean 测试清理器运行后触发超时会话清理。
func TestSessionTimeoutCleaner_RunAndClean(t *testing.T) {
	expiredSession := domainsession.NewOnlineSession(1001, "conn", 1700000000000, 1, "device")
	sessionRepo := &mockSessionRepo{
		findExpiredResult: []*domainsession.OnlineSession{expiredSession},
	}
	eventBus := &mockEventBus{}
	// SessionTimeoutMs=4000 → cleanInterval=1000ms=1s（最小1秒）
	cfg := domainconfig.DefaultAuthConfig()
	cfg.SessionTimeoutMs = 4000

	cleaner := newTestSessionCleaner(sessionRepo, eventBus, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	go cleaner.Run(ctx)

	// 等待ticker触发（1秒间隔+缓冲）
	time.Sleep(1500 * time.Millisecond)
	cancel()

	// 应查询超时会话
	assert.Greater(t, sessionRepo.findExpiredCallCount, 0, "应调用FindExpired查询超时会话")
	// 应删除超时会话
	assert.Greater(t, sessionRepo.deleteCallCount, 0, "应删除超时会话")
	// 应发布下线事件
	assert.Greater(t, eventBus.publishCount, 0, "应发布下线事件")
}

// TestSessionTimeoutCleaner_NoExpired 测试无超时会话时不执行删除。
func TestSessionTimeoutCleaner_NoExpired(t *testing.T) {
	sessionRepo := &mockSessionRepo{findExpiredResult: nil} // 无超时会话
	eventBus := &mockEventBus{}
	cfg := domainconfig.DefaultAuthConfig()
	cfg.SessionTimeoutMs = 4000

	cleaner := newTestSessionCleaner(sessionRepo, eventBus, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	go cleaner.Run(ctx)

	time.Sleep(1500 * time.Millisecond)
	cancel()

	assert.Greater(t, sessionRepo.findExpiredCallCount, 0, "应调用FindExpired")
	assert.Equal(t, 0, sessionRepo.deleteCallCount, "无超时会话不应删除")
	assert.Equal(t, 0, eventBus.publishCount, "无超时会话不应发布事件")
}

// TestSessionTimeoutCleaner_FindExpiredError 测试FindExpired故障时不阻塞下一周期。
func TestSessionTimeoutCleaner_FindExpiredError(t *testing.T) {
	sessionRepo := &mockSessionRepo{findExpiredErr: assert.AnError}
	eventBus := &mockEventBus{}
	cfg := domainconfig.DefaultAuthConfig()
	cfg.SessionTimeoutMs = 4000

	cleaner := newTestSessionCleaner(sessionRepo, eventBus, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	go cleaner.Run(ctx)

	time.Sleep(1500 * time.Millisecond)
	cancel()

	// FindExpired故障不应导致panic，且不应删除或发布
	assert.Greater(t, sessionRepo.findExpiredCallCount, 0)
	assert.Equal(t, 0, sessionRepo.deleteCallCount)
}

// TestSessionTimeoutCleaner_GracefulExit 测试ctx cancel后清理器优雅退出。
func TestSessionTimeoutCleaner_GracefulExit(t *testing.T) {
	sessionRepo := &mockSessionRepo{}
	eventBus := &mockEventBus{}
	cfg := domainconfig.DefaultAuthConfig()

	cleaner := newTestSessionCleaner(sessionRepo, eventBus, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		cleaner.Run(ctx)
		close(done)
	}()

	// 立即cancel，应在1秒内退出
	cancel()
	select {
	case <-done:
		// 成功退出
	case <-time.After(3 * time.Second):
		t.Fatal("清理器未在3秒内退出")
	}
}

// TestSessionTimeoutCleaner_DeleteError 测试删除超时会话故障时跳过该会话继续下一个。
func TestSessionTimeoutCleaner_DeleteError(t *testing.T) {
	expiredSession := domainsession.NewOnlineSession(1001, "conn", 1700000000000, 1, "device")
	sessionRepo := &mockSessionRepo{
		findExpiredResult: []*domainsession.OnlineSession{expiredSession},
		deleteErr:         assert.AnError,
	}
	eventBus := &mockEventBus{}
	cfg := domainconfig.DefaultAuthConfig()
	cfg.SessionTimeoutMs = 4000

	cleaner := newTestSessionCleaner(sessionRepo, eventBus, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	go cleaner.Run(ctx)

	time.Sleep(1500 * time.Millisecond)
	cancel()

	// 删除失败不应发布下线事件
	require.Greater(t, sessionRepo.deleteCallCount, 0)
	assert.Equal(t, 0, eventBus.publishCount, "删除失败不应发布下线事件")
}
