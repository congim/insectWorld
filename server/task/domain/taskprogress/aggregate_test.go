// Package taskprogress 任务进度聚合根单元测试。
package taskprogress

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTaskProgress 测试任务进度创建。
func TestNewTaskProgress(t *testing.T) {
	tp := NewTaskProgress(1, 100, 1001, 10, CycleOnce)
	assert.Equal(t, int64(1), tp.TaskID())
	assert.Equal(t, int64(100), tp.PlayerID())
	assert.Equal(t, int64(0), tp.Current())
	assert.Equal(t, StatusInProgress, tp.Status())
}

// TestTaskProgress_Advance 测试任务进度推进。
func TestTaskProgress_Advance(t *testing.T) {
	tp := NewTaskProgress(1, 100, 1001, 10, CycleOnce)
	event, err := tp.Advance(context.Background(), 3, 1000)
	require.NoError(t, err)
	assert.Equal(t, int64(3), tp.Current())
	assert.Equal(t, int64(3), event.Current)
	assert.False(t, event.Completed)
}

// TestTaskProgress_Advance_Completed 测试进度达到目标自动完成。
func TestTaskProgress_Advance_Completed(t *testing.T) {
	tp := NewTaskProgress(1, 100, 1001, 10, CycleOnce)
	event, err := tp.Advance(context.Background(), 10, 1000)
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, tp.Status())
	assert.True(t, event.Completed)
}

// TestTaskProgress_Advance_AlreadyClaimed 测试已领取奖励后推进失败。
func TestTaskProgress_Advance_AlreadyClaimed(t *testing.T) {
	tp := NewTaskProgress(1, 100, 1001, 10, CycleOnce)
	_, _ = tp.Advance(context.Background(), 10, 1000)
	_, err := tp.ClaimReward(2000)
	require.NoError(t, err)

	_, err = tp.Advance(context.Background(), 1, 3000)
	assert.Error(t, err)
}

// TestTaskProgress_ClaimReward 测试领取奖励。
func TestTaskProgress_ClaimReward(t *testing.T) {
	tp := NewTaskProgress(1, 100, 1001, 10, CycleOnce)
	_, _ = tp.Advance(context.Background(), 10, 1000)

	event, err := tp.ClaimReward(2000)
	require.NoError(t, err)
	assert.Equal(t, int64(1), event.TaskID)
	assert.Equal(t, StatusClaimed, tp.Status())
}

// TestTaskProgress_ClaimReward_NotCompleted 测试未完成时领取失败。
func TestTaskProgress_ClaimReward_NotCompleted(t *testing.T) {
	tp := NewTaskProgress(1, 100, 1001, 10, CycleOnce)
	_, _ = tp.Advance(context.Background(), 5, 1000)

	_, err := tp.ClaimReward(2000)
	assert.Error(t, err)
}

// TestTaskProgress_Reset 测试周期任务重置。
func TestTaskProgress_Reset(t *testing.T) {
	tp := NewTaskProgress(1, 100, 1001, 10, CycleDaily)
	_, _ = tp.Advance(context.Background(), 5, 1000)

	event := tp.Reset(2000)
	assert.Equal(t, int64(0), tp.Current())
	assert.Equal(t, StatusInProgress, tp.Status())
	assert.Equal(t, int64(0), event.Current)
}
