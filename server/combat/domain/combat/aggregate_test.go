// Package combat 战斗聚合根，维护战斗状态与轮次执行。
// 本文件定义Combat聚合根的单元测试。
package combat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCombat 测试战斗聚合根创建。
func TestNewCombat(t *testing.T) {
	attackerIDs := []int64{101, 102}
	defenderIDs := []int64{201, 202}

	c := NewCombat(1, 1, 10, attackerIDs, defenderIDs, 1000)

	assert.Equal(t, int64(1), c.CombatID())
	assert.Equal(t, 1, c.CombatType())
	assert.Equal(t, StatusInProgress, c.Status())
	assert.Equal(t, 0, c.CurrentRound())
	assert.Equal(t, 10, c.MaxRounds())
	assert.True(t, c.IsInProgress())
}

// TestCombat_ExecuteRound 测试轮次执行。
func TestCombat_ExecuteRound(t *testing.T) {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 1000)

	event, err := c.ExecuteRound()
	require.NoError(t, err)
	assert.Equal(t, int64(1), event.CombatID)
	assert.Equal(t, 1, event.Round)
	assert.Equal(t, 1, c.CurrentRound())

	// 第二轮
	event2, err := c.ExecuteRound()
	require.NoError(t, err)
	assert.Equal(t, 2, event2.Round)
	assert.Equal(t, 2, c.CurrentRound())
}

// TestCombat_ExecuteRound_NotInProgress 测试非进行中状态执行轮次失败。
func TestCombat_ExecuteRound_NotInProgress(t *testing.T) {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 1000)

	// 结束战斗
	_, err := c.End(ResultAttackerWin)
	require.NoError(t, err)

	// 再执行轮次应失败
	_, err = c.ExecuteRound()
	assert.Error(t, err)
}

// TestCombat_CheckMaxRounds 测试轮数超限校验。
func TestCombat_CheckMaxRounds(t *testing.T) {
	c := NewCombat(1, 1, 3, []int64{101}, []int64{201}, 1000)

	// 执行3轮
	for i := 0; i < 3; i++ {
		_, err := c.ExecuteRound()
		require.NoError(t, err)
	}

	assert.True(t, c.CheckMaxRounds())
}

// TestCombat_CheckMaxRounds_NotExceeded 测试轮数未超限。
func TestCombat_CheckMaxRounds_NotExceeded(t *testing.T) {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 1000)

	_, err := c.ExecuteRound()
	require.NoError(t, err)

	assert.False(t, c.CheckMaxRounds())
}

// TestCombat_End 测试战斗结束。
func TestCombat_End(t *testing.T) {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 1000)

	_, err := c.ExecuteRound()
	require.NoError(t, err)

	event, err := c.End(ResultAttackerWin)
	require.NoError(t, err)
	assert.Equal(t, int64(1), event.CombatID)
	assert.Equal(t, ResultAttackerWin, event.Result)
	assert.Equal(t, 1, event.TotalRounds)
	assert.Equal(t, StatusEnded, c.Status())
	assert.False(t, c.IsInProgress())
}

// TestCombat_End_NotInProgress 测试非进行中状态结束战斗失败。
func TestCombat_End_NotInProgress(t *testing.T) {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 1000)

	// 先结束
	_, err := c.End(ResultAttackerWin)
	require.NoError(t, err)

	// 再结束应失败
	_, err = c.End(ResultDefenderWin)
	assert.Error(t, err)
}

// TestCombat_Retreat 测试撤退。
func TestCombat_Retreat(t *testing.T) {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 1000)

	_, err := c.ExecuteRound()
	require.NoError(t, err)

	event, err := c.Retreat()
	require.NoError(t, err)
	assert.Equal(t, int64(1), event.CombatID)
	assert.Equal(t, ResultDefenderWin, event.Result)
	assert.Equal(t, StatusRetreated, c.Status())
	assert.False(t, c.IsInProgress())
}

// TestCombat_Retreat_NotInProgress 测试非进行中状态撤退失败。
func TestCombat_Retreat_NotInProgress(t *testing.T) {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 1000)

	// 先撤退
	_, err := c.Retreat()
	require.NoError(t, err)

	// 再撤退应失败
	_, err = c.Retreat()
	assert.Error(t, err)
}

// TestCombat_IsLongCombat 测试长时战斗判定。
func TestCombat_IsLongCombat(t *testing.T) {
	c1 := NewCombat(1, 1, 5, []int64{101}, []int64{201}, 1000)
	assert.False(t, c1.IsLongCombat())

	c2 := NewCombat(2, 1, 6, []int64{101}, []int64{201}, 1000)
	assert.True(t, c2.IsLongCombat())
}

// TestCombat_SetFormation 测试设置阵型ID。
func TestCombat_SetFormation(t *testing.T) {
	c := NewCombat(1, 1, 10, []int64{101}, []int64{201}, 1000)
	c.SetFormation(5)
	// 阵型ID设置后无getter暴露，验证不panic即可
}
