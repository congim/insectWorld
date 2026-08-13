// Package player 定义玩家档案聚合及其持久化契约。
package player

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gameerr "insectworld/server/game/domain/errors"
)

// TestNewProfile 验证玩家档案初始化不变量和输入标准化。
func TestNewProfile(t *testing.T) {
	t.Parallel()
	profile, err := NewProfile(7, "faction", "  玩家  ", 1000, "0.1.0", "create-1")
	require.NoError(t, err)
	assert.Equal(t, int64(7), profile.PlayerID())
	assert.Equal(t, "faction", profile.FactionID())
	assert.Equal(t, "玩家", profile.Nickname())
	assert.Equal(t, int32(1), profile.Level())
	assert.Zero(t, profile.Experience())
	assert.Equal(t, int64(1000), profile.CreatedAt())
	assert.Equal(t, "0.1.0", profile.ConfigVersion())
	assert.Equal(t, "create-1", profile.CommandID())
	assert.Equal(t, profile.PlayerID(), profile.Clone().PlayerID())
}

// TestNewProfileRejectsInvalidInput 验证无效玩家档案输入被领域拒绝。
func TestNewProfileRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string // 用例名称
		playerID  int64  // 玩家ID输入
		factionID string // 阵营ID输入
		nickname  string // 昵称输入
		createdAt int64  // 创建时间输入
		commandID string // 幂等键输入
	}{
		{name: "玩家ID非法", playerID: 0, factionID: "faction", nickname: "玩家", createdAt: 1000, commandID: "cmd"},
		{name: "阵营为空", playerID: 1, nickname: "玩家", createdAt: 1000, commandID: "cmd"},
		{name: "昵称为空", playerID: 1, factionID: "faction", nickname: " ", createdAt: 1000, commandID: "cmd"},
		{name: "时间非法", playerID: 1, factionID: "faction", nickname: "玩家", commandID: "cmd"},
		{name: "幂等键为空", playerID: 1, factionID: "faction", nickname: "玩家", createdAt: 1000},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewProfile(testCase.playerID, testCase.factionID, testCase.nickname, testCase.createdAt, "0.1.0", testCase.commandID)
			require.Error(t, err)
			assert.True(t, errors.Is(err, gameerr.ErrInvalidCommand))
		})
	}
}

// TestRestoreProfile 验证持久化恢复接受合法成长值并拒绝损坏数据。
func TestRestoreProfile(t *testing.T) {
	t.Parallel()
	profile, err := RestoreProfile(1, "faction", "玩家", 3, 20, 1000, "0.1.0", "cmd")
	require.NoError(t, err)
	assert.Equal(t, int32(3), profile.Level())
	assert.Equal(t, int64(20), profile.Experience())
	_, err = RestoreProfile(1, "faction", "玩家", 0, 20, 1000, "0.1.0", "cmd")
	require.Error(t, err)
	assert.True(t, errors.Is(err, gameerr.ErrStateConflict))
}
