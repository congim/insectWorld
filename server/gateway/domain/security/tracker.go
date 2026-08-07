// Package security 登录安全领域服务，提供暴力破解防护能力。
//
// domain层零外部依赖（规范3），LoginFailureTracker接口在本包声明，
// infrastructure层实现Redis适配。状态机：正常→失败计数中→锁定。
package security

import "context"

// LoginFailureTracker 登录失败计数与锁定能力接口，infrastructure层实现Redis适配。
//
// 接口在domain层声明（规范3 DDD），保证domain层不依赖第三方Redis包。
// 失败计数与锁定状态通过Redis TTL自动过期，无需后台清理任务。
type LoginFailureTracker interface {
	// RecordFailure 记录一次登录失败，递增失败计数。
	// 返回当前失败次数，达上限时由调用方决定是否进入锁定。
	// Redis故障返回ErrFailureTrackerUnavailable，降级为"不锁定"。
	RecordFailure(ctx context.Context, username string) (currentCount int, err error)

	// IsLocked 查询账号是否处于锁定状态。
	// 返回true表示已锁定，Redis故障返回(false, ErrFailureTrackerUnavailable)降级为"不锁定"。
	IsLocked(ctx context.Context, username string) (bool, error)

	// RemainingLockSeconds 查询账号锁定剩余秒数。
	// 未锁定返回0，Redis故障返回(0, ErrFailureTrackerUnavailable)。
	RemainingLockSeconds(ctx context.Context, username string) (int64, error)

	// ResetClear 清零失败计数与锁定状态，登录成功时调用。
	ResetClear(ctx context.Context, username string) error
}
