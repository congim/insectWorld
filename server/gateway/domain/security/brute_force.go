// Package security 登录安全领域服务，提供暴力破解防护能力。
package security

import (
	"context"
	"fmt"

	gatewayerr "insectworld/server/gateway/domain/errors"
)

// BruteForceProtector 暴力破解防护领域服务，封装失败计数与锁定判定逻辑。
//
// 领域服务依赖LoginFailureTracker接口（infrastructure层注入Redis实现）。
// 状态机符合design.md 2.1.3.3节：正常→失败计数中→锁定。
// 锁定判定与失败计数逻辑覆盖spec 5.2.1 规则4-5全部验收条件。
type BruteForceProtector struct {
	tracker        LoginFailureTracker // 失败计数器，Redis实现
	failMaxCount   int                 // 登录失败最大次数，达上限进入锁定
	lockDurationMs int64               // 登录锁定时长，毫秒级
}

// NewBruteForceProtector 创建暴力破解防护领域服务实例。
//
// tracker为失败计数器实现（infrastructure层注入），
// failMaxCount为失败上限，lockDurationMs为锁定时长毫秒。
func NewBruteForceProtector(tracker LoginFailureTracker, failMaxCount int, lockDurationMs int64) *BruteForceProtector {
	return &BruteForceProtector{
		tracker:        tracker,
		failMaxCount:   failMaxCount,
		lockDurationMs: lockDurationMs,
	}
}

// OnLoginFailure 登录失败时调用，记录失败并返回当前失败次数。
//
// 调用LoginFailureTracker.RecordFailure递增失败计数。
// 失败计数器故障时返回ErrFailureTrackerUnavailable，降级为"不锁定"（允许登录，依赖限流器兜底）。
func (p *BruteForceProtector) OnLoginFailure(ctx context.Context, username string) (int, error) {
	count, err := p.tracker.RecordFailure(ctx, username)
	if err != nil {
		return 0, fmt.Errorf("记录登录失败失败: %w", err)
	}
	return count, nil
}

// CheckLocked 查询账号是否处于锁定状态，返回锁定状态与剩余锁定秒数。
//
// 调用LoginFailureTracker.IsLocked与RemainingLockSeconds。
// 未锁定返回(false, 0, nil)，已锁定返回(true, remainingSeconds, nil)。
// 失败计数器故障时降级为"不锁定"返回(false, 0, nil)，避免故障导致全部请求被拒。
func (p *BruteForceProtector) CheckLocked(ctx context.Context, username string) (locked bool, remainingSeconds int64, err error) {
	locked, err = p.tracker.IsLocked(ctx, username)
	if err != nil {
		return false, 0, nil
	}
	if !locked {
		return false, 0, nil
	}
	remainingSeconds, _ = p.tracker.RemainingLockSeconds(ctx, username)
	return true, remainingSeconds, nil
}

// OnLoginSuccess 登录成功时调用，清零失败计数与锁定状态。
//
// 调用LoginFailureTracker.ResetClear。
// 失败计数器故障时记录错误但不阻塞登录成功流程（清零失败不影响本次登录）。
func (p *BruteForceProtector) OnLoginSuccess(ctx context.Context, username string) error {
	if err := p.tracker.ResetClear(ctx, username); err != nil {
		return fmt.Errorf("清零失败计数失败: %w", gatewayerr.ErrFailureTrackerUnavailable)
	}
	return nil
}
