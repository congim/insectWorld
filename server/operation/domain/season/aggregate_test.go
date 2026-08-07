// Package season 赛季聚合根，维护赛季阶段与状态。
// 本文件定义Season聚合根与ScoreBoard的单元测试。
package season

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSeason 测试赛季创建。
func TestNewSeason(t *testing.T) {
	s := NewSeason(1, 1000)
	assert.Equal(t, int64(1), s.SeasonID())
	assert.Equal(t, PhasePreparation, s.Phase())
}

// TestSeason_TransitionPhase 测试阶段切换。
func TestSeason_TransitionPhase(t *testing.T) {
	s := NewSeason(1, 1000)

	event, err := s.TransitionPhase(PhaseWar)
	require.NoError(t, err)
	assert.Equal(t, int64(1), event.SeasonID)
	assert.Equal(t, PhasePreparation, event.OldPhase)
	assert.Equal(t, PhaseWar, event.NewPhase)
	assert.Equal(t, PhaseWar, s.Phase())
}

// TestSeason_TransitionPhase_Sequence 测试阶段顺序切换。
func TestSeason_TransitionPhase_Sequence(t *testing.T) {
	s := NewSeason(1, 1000)

	_, err := s.TransitionPhase(PhaseWar)
	require.NoError(t, err)

	_, err = s.TransitionPhase(PhaseSettlement)
	require.NoError(t, err)

	assert.Equal(t, PhaseSettlement, s.Phase())
}

// TestSeason_TransitionPhase_NonProgressive 测试非递进阶段切换失败。
func TestSeason_TransitionPhase_NonProgressive(t *testing.T) {
	s := NewSeason(1, 1000)

	// 准备→准备（同阶段）应失败
	_, err := s.TransitionPhase(PhasePreparation)
	assert.Error(t, err)

	// 准备→结算（跳过战争）可以，因为只校验target>current
	_, err = s.TransitionPhase(PhaseSettlement)
	require.NoError(t, err)
}

// TestSeason_TransitionPhase_AlreadyEnded 测试已结束赛季阶段切换失败。
func TestSeason_TransitionPhase_AlreadyEnded(t *testing.T) {
	s := NewSeason(1, 1000)

	_, err := s.Reset(context.Background(), []string{"player"}, []string{"profile"})
	require.NoError(t, err)

	_, err = s.TransitionPhase(PhaseWar)
	assert.Error(t, err)
}

// TestSeason_Reset 测试赛季重置。
func TestSeason_Reset(t *testing.T) {
	s := NewSeason(1, 1000)

	resetScope := []string{"player", "alliance"}
	preserveList := []string{"player_profile"}

	event, err := s.Reset(context.Background(), resetScope, preserveList)
	require.NoError(t, err)
	assert.Equal(t, int64(1), event.SeasonID)
	assert.Equal(t, resetScope, event.ResetScope)
	assert.Equal(t, preserveList, event.PreserveList)
	assert.Equal(t, PhaseEnded, s.Phase())
}

// TestSeason_Reset_AlreadyEnded 测试已结束赛季重置失败。
func TestSeason_Reset_AlreadyEnded(t *testing.T) {
	s := NewSeason(1, 1000)

	_, err := s.Reset(context.Background(), []string{"player"}, nil)
	require.NoError(t, err)

	_, err = s.Reset(context.Background(), []string{"player"}, nil)
	assert.Error(t, err)
}

// TestScoreBoard_AddScore 测试增加积分。
func TestScoreBoard_AddScore(t *testing.T) {
	sb := NewScoreBoard(1)

	sb.AddScore(101, 500)
	assert.Equal(t, int64(500), sb.GetScore(101))

	sb.AddScore(101, 300)
	assert.Equal(t, int64(800), sb.GetScore(101))
}

// TestScoreBoard_GetScore_NotExist 测试查询不存在目标的积分。
func TestScoreBoard_GetScore_NotExist(t *testing.T) {
	sb := NewScoreBoard(1)
	assert.Equal(t, int64(0), sb.GetScore(999))
}
