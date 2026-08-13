// Package achievement 成就聚合根，维护玩家成就达成状态与奖励领取。
// Achievement聚合根订阅业务事件判定成就条件，达$成后解锁称号/头像框等奖励。
// 对应spec.md 5.1.7.3节Task上下文功能4"成就达成"。
package achievement

import (
	"context"
	"fmt"

	taskerr "insectworld/server/task/domain/errors"
)

// 成就状态常量（规范1）。
const (
	StatusLocked   = 1 // 未解锁
	StatusUnlocked = 2 // 已解锁
	StatusClaimed  = 3 // 已领取奖励
)

// Achievement 成就聚合根，维护玩家单个成就的达成与领取状态。
type Achievement struct {
	achievementID int64 // 成就ID，全局唯一
	playerID      int64 // 玩家ID
	defID         int64 // 成就定义ID，对应achievements.json配置
	status        int   // 成就状态：1=未解锁 2=已解锁 3=已领取
	unlockTime    int64 // 解锁时间戳（毫秒），0表示未解锁
	claimTime     int64 // 领取时间戳（毫秒），0表示未领取
}

// NewAchievement 创建成就聚合根实例。
func NewAchievement(achievementID int64, playerID int64, defID int64) *Achievement {
	return &Achievement{
		achievementID: achievementID,
		playerID:      playerID,
		defID:         defID,
		status:        StatusLocked,
	}
}

// AchievementID 返回成就ID。
func (a *Achievement) AchievementID() int64 { return a.achievementID }

// Status 返回成就状态。
func (a *Achievement) Status() int { return a.status }

// Unlock 解锁成就，状态从未解锁转为已解锁。
func (a *Achievement) Unlock(now int64) (*AchievementUnlockedEvent, error) {
	if a.status != StatusLocked {
		return nil, fmt.Errorf("成就已解锁，achievementID=%d: %w",
			a.achievementID, taskerr.ErrAchievementAlreadyUnlocked)
	}

	a.status = StatusUnlocked
	a.unlockTime = now

	return &AchievementUnlockedEvent{
		AchievementID: a.achievementID,
		PlayerID:      a.playerID,
		DefID:         a.defID,
		UnlockTime:    now,
	}, nil
}

// ClaimReward 领取成就奖励，状态从已解锁转为已领取。
func (a *Achievement) ClaimReward(now int64) (*AchievementClaimedEvent, error) {
	if a.status != StatusUnlocked {
		return nil, fmt.Errorf("成就未解锁或已领取，achievementID=%d，当前状态=%d: %w",
			a.achievementID, a.status, taskerr.ErrAchievementAlreadyUnlocked)
	}

	a.status = StatusClaimed
	a.claimTime = now

	return &AchievementClaimedEvent{
		AchievementID: a.achievementID,
		PlayerID:      a.playerID,
		DefID:         a.defID,
		ClaimTime:     now,
	}, nil
}

// AchievementUnlockedEvent 成就解锁事件。
type AchievementUnlockedEvent struct {
	AchievementID int64 // 成就ID
	PlayerID      int64 // 玩家ID
	DefID         int64 // 成就定义ID，用于查询称号/头像框奖励
	UnlockTime    int64 // 解锁时间戳（毫秒）
}

// AchievementClaimedEvent 成就奖励领取事件。
type AchievementClaimedEvent struct {
	AchievementID int64 // 成就ID
	PlayerID      int64 // 玩家ID
	DefID         int64 // 成就定义ID
	ClaimTime     int64 // 领取时间戳（毫秒）
}

// AchievementRepository 成就聚合根仓储接口，在domain层声明（规范3）。
type AchievementRepository interface {
	// LoadAchievement 加载成就聚合根
	LoadAchievement(ctx context.Context, achievementID int64) (*Achievement, error)
	// SaveAchievement 保存成就聚合根
	SaveAchievement(ctx context.Context, a *Achievement) error
}
