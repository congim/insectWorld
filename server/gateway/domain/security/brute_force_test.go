// Package security 登录安全领域服务，提供暴力破解防护能力。
package security

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gatewayerr "insectworld/server/gateway/domain/errors"
)

// mockTracker 测试用登录失败计数器mock，可配置各方法返回值。
type mockTracker struct {
	recordFailCount int   // RecordFailure返回的当前失败次数
	recordFailErr   error // RecordFailure返回的错误
	isLockedResult  bool  // IsLocked返回的锁定状态
	isLockedErr     error // IsLocked返回的错误
	remainingResult int64 // RemainingLockSeconds返回的剩余秒数
	remainingErr    error // RemainingLockSeconds返回的错误
	resetErr        error // ResetClear返回的错误

	recordFailCalled bool // RecordFailure被调用标记
	isLockedCalled   bool // IsLocked被调用标记
	resetCalled      bool // ResetClear被调用标记
}

func (m *mockTracker) RecordFailure(ctx context.Context, username string) (int, error) {
	m.recordFailCalled = true
	return m.recordFailCount, m.recordFailErr
}

func (m *mockTracker) IsLocked(ctx context.Context, username string) (bool, error) {
	m.isLockedCalled = true
	return m.isLockedResult, m.isLockedErr
}

func (m *mockTracker) RemainingLockSeconds(ctx context.Context, username string) (int64, error) {
	return m.remainingResult, m.remainingErr
}

func (m *mockTracker) ResetClear(ctx context.Context, username string) error {
	m.resetCalled = true
	return m.resetErr
}

// 确保mockTracker实现LoginFailureTracker接口（编译期检查）。
var _ LoginFailureTracker = (*mockTracker)(nil)

// TestNewBruteForceProtector 测试创建暴力破解防护领域服务实例。
func TestNewBruteForceProtector(t *testing.T) {
	tracker := &mockTracker{}
	p := NewBruteForceProtector(tracker, 5, 900000)
	require.NotNil(t, p)
}

// TestBruteForceProtector_OnLoginFailure 测试登录失败记录逻辑。
func TestBruteForceProtector_OnLoginFailure(t *testing.T) {
	t.Run("正常记录失败", func(t *testing.T) {
		tracker := &mockTracker{recordFailCount: 3}
		p := NewBruteForceProtector(tracker, 5, 900000)
		count, err := p.OnLoginFailure(context.Background(), "testuser")
		require.NoError(t, err)
		assert.Equal(t, 3, count)
		assert.True(t, tracker.recordFailCalled)
	})

	t.Run("计数器故障降级返回错误", func(t *testing.T) {
		tracker := &mockTracker{recordFailErr: gatewayerr.ErrFailureTrackerUnavailable}
		p := NewBruteForceProtector(tracker, 5, 900000)
		_, err := p.OnLoginFailure(context.Background(), "testuser")
		require.Error(t, err)
		// 错误链应包裹底层错误
		assert.True(t, errors.Is(err, gatewayerr.ErrFailureTrackerUnavailable))
	})
}

// TestBruteForceProtector_CheckLocked 测试锁定状态查询逻辑，覆盖降级语义。
func TestBruteForceProtector_CheckLocked(t *testing.T) {
	t.Run("未锁定返回false", func(t *testing.T) {
		tracker := &mockTracker{isLockedResult: false}
		p := NewBruteForceProtector(tracker, 5, 900000)
		locked, remaining, err := p.CheckLocked(context.Background(), "testuser")
		require.NoError(t, err)
		assert.False(t, locked)
		assert.Equal(t, int64(0), remaining)
	})

	t.Run("已锁定返回剩余秒数", func(t *testing.T) {
		tracker := &mockTracker{isLockedResult: true, remainingResult: 600}
		p := NewBruteForceProtector(tracker, 5, 900000)
		locked, remaining, err := p.CheckLocked(context.Background(), "testuser")
		require.NoError(t, err)
		assert.True(t, locked)
		assert.Equal(t, int64(600), remaining)
	})

	t.Run("IsLocked故障降级为不锁定", func(t *testing.T) {
		tracker := &mockTracker{isLockedErr: gatewayerr.ErrFailureTrackerUnavailable}
		p := NewBruteForceProtector(tracker, 5, 900000)
		locked, remaining, err := p.CheckLocked(context.Background(), "testuser")
		// 降级语义：故障时返回(false,0,nil)避免全部请求被拒
		require.NoError(t, err)
		assert.False(t, locked)
		assert.Equal(t, int64(0), remaining)
	})

	t.Run("已锁定但RemainingLockSeconds故障返回0", func(t *testing.T) {
		tracker := &mockTracker{
			isLockedResult:  true,
			remainingErr:    gatewayerr.ErrFailureTrackerUnavailable,
			remainingResult: 0,
		}
		p := NewBruteForceProtector(tracker, 5, 900000)
		locked, remaining, err := p.CheckLocked(context.Background(), "testuser")
		require.NoError(t, err)
		assert.True(t, locked)
		assert.Equal(t, int64(0), remaining)
	})
}

// TestBruteForceProtector_OnLoginSuccess 测试登录成功清零逻辑。
func TestBruteForceProtector_OnLoginSuccess(t *testing.T) {
	t.Run("正常清零", func(t *testing.T) {
		tracker := &mockTracker{}
		p := NewBruteForceProtector(tracker, 5, 900000)
		err := p.OnLoginSuccess(context.Background(), "testuser")
		require.NoError(t, err)
		assert.True(t, tracker.resetCalled)
	})

	t.Run("清零故障返回错误", func(t *testing.T) {
		tracker := &mockTracker{resetErr: gatewayerr.ErrFailureTrackerUnavailable}
		p := NewBruteForceProtector(tracker, 5, 900000)
		err := p.OnLoginSuccess(context.Background(), "testuser")
		require.Error(t, err)
		assert.True(t, errors.Is(err, gatewayerr.ErrFailureTrackerUnavailable))
	})
}
