// Package building 定义玩家建筑聚合及建造状态机。
package building

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gameerr "insectworld/server/game/domain/errors"
)

// TestBuildingComplete 验证建筑只能在配置时间到达后完成且重复完成幂等。
func TestBuildingComplete(t *testing.T) {
	t.Parallel()
	aggregate, err := NewConstruction(1, 2, "farm", 1000, 2000, "build-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), aggregate.ID())
	assert.Equal(t, int64(2), aggregate.PlayerID())
	assert.Equal(t, "farm", aggregate.TypeID())
	assert.Equal(t, int64(1000), aggregate.StartedAt())
	assert.Equal(t, int64(2000), aggregate.CompleteAt())
	assert.Equal(t, "build-1", aggregate.CommandID())
	assert.Equal(t, aggregate.ID(), aggregate.Clone().ID())

	err = aggregate.Complete(1999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrTaskNotReady))
	assert.Equal(t, StatusConstructing, aggregate.Status())

	require.NoError(t, aggregate.Complete(2000))
	require.NoError(t, aggregate.Complete(2001))
	assert.Equal(t, StatusOperational, aggregate.Status())
}

// TestNewConstructionRejectsInvalidInput 验证建筑领域不接受无效时间边界。
func TestNewConstructionRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	_, err := NewConstruction(1, 2, "farm", 1000, 1000, "build-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrInvalidCommand))
}
