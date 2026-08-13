// Package achievement 成就聚合根单元测试。
package achievement

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAchievement 测试成就创建。
func TestNewAchievement(t *testing.T) {
	a := NewAchievement(1, 100, 1001)
	assert.Equal(t, int64(1), a.AchievementID())
	assert.Equal(t, StatusLocked, a.Status())
}

// TestAchievement_Unlock 测试成就解锁。
func TestAchievement_Unlock(t *testing.T) {
	a := NewAchievement(1, 100, 1001)
	event, err := a.Unlock(1000)
	require.NoError(t, err)
	assert.Equal(t, StatusUnlocked, a.Status())
	assert.Equal(t, int64(1000), event.UnlockTime)
}

// TestAchievement_Unlock_AlreadyUnlocked 测试重复解锁失败。
func TestAchievement_Unlock_AlreadyUnlocked(t *testing.T) {
	a := NewAchievement(1, 100, 1001)
	_, _ = a.Unlock(1000)
	_, err := a.Unlock(2000)
	assert.Error(t, err)
}

// TestAchievement_ClaimReward 测试领取成就奖励。
func TestAchievement_ClaimReward(t *testing.T) {
	a := NewAchievement(1, 100, 1001)
	_, _ = a.Unlock(1000)

	event, err := a.ClaimReward(2000)
	require.NoError(t, err)
	assert.Equal(t, int64(1), event.AchievementID)
	assert.Equal(t, StatusClaimed, a.Status())
}

// TestAchievement_ClaimReward_NotUnlocked 测试未解锁时领取失败。
func TestAchievement_ClaimReward_NotUnlocked(t *testing.T) {
	a := NewAchievement(1, 100, 1001)
	_, err := a.ClaimReward(2000)
	assert.Error(t, err)
}
