// Package training 定义单位训练任务聚合及训练状态机。
package training

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gameerr "insectworld/server/game/domain/errors"
)

// TestTaskComplete 验证训练任务只能在配置时间到达后完成且重复完成幂等。
func TestTaskComplete(t *testing.T) {
	t.Parallel()
	aggregate, err := NewTask(1, 2, 3, "worker", 2, 1000, 3000, "0.1.0", "train-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), aggregate.ID())
	assert.Equal(t, int64(2), aggregate.PlayerID())
	assert.Equal(t, int64(3), aggregate.BuildingID())
	assert.Equal(t, "worker", aggregate.UnitTypeID())
	assert.Equal(t, int64(2), aggregate.Count())
	assert.Equal(t, int64(1000), aggregate.StartedAt())
	assert.Equal(t, int64(3000), aggregate.CompleteAt())
	assert.Equal(t, "0.1.0", aggregate.ConfigVersion())
	assert.Equal(t, "train-1", aggregate.CommandID())
	assert.Equal(t, aggregate.ID(), aggregate.Clone().ID())

	err = aggregate.Complete(2999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrTaskNotReady))
	assert.Equal(t, StatusTraining, aggregate.Status())

	require.NoError(t, aggregate.Complete(3000))
	require.NoError(t, aggregate.Complete(3001))
	assert.Equal(t, StatusComplete, aggregate.Status())
}

// TestNewTaskRejectsInvalidCount 验证训练数量必须大于0。
func TestNewTaskRejectsInvalidCount(t *testing.T) {
	t.Parallel()
	_, err := NewTask(1, 2, 3, "worker", 0, 1000, 3000, "0.1.0", "train-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrInvalidCommand))
}

// TestRestoreTask 验证训练任务恢复保留合法状态并拒绝未知状态。
func TestRestoreTask(t *testing.T) {
	t.Parallel()
	aggregate, err := RestoreTask(1, 2, 3, "worker", 2, StatusComplete, 1000, 3000, "0.1.0", "cmd")
	require.NoError(t, err)
	assert.Equal(t, StatusComplete, aggregate.Status())
	_, err = RestoreTask(1, 2, 3, "worker", 2, Status(99), 1000, 3000, "0.1.0", "cmd")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrStateConflict))
}
