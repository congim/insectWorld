// Package gamemap 地图聚合根，维护地图格子状态与地形定义。
// 本文件定义Map聚合根的单元测试。
package gamemap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"insectworld/server/world/domain/vo"
)

// TestMap_InBounds 测试坐标边界校验。
func TestMap_InBounds(t *testing.T) {
	m := NewMap(1, 100, 100, GridTypeQuad)

	assert.True(t, m.InBounds(vo.Position{X: 0, Y: 0}))
	assert.True(t, m.InBounds(vo.Position{X: 99, Y: 99}))
	assert.False(t, m.InBounds(vo.Position{X: -1, Y: 0}))
	assert.False(t, m.InBounds(vo.Position{X: 100, Y: 0}))
	assert.False(t, m.InBounds(vo.Position{X: 0, Y: 100}))
}

// TestMap_ChangeTerrain 测试地形变更。
func TestMap_ChangeTerrain(t *testing.T) {
	m := NewMap(1, 100, 100, GridTypeQuad)
	pos := vo.Position{X: 10, Y: 10}

	// 设置初始地形
	require.NoError(t, m.SetCell(pos, 1))

	// 变更地形
	event, err := m.ChangeTerrain(pos, 2)
	require.NoError(t, err)
	assert.Equal(t, int32(1), event.OldTerrainID)
	assert.Equal(t, int32(2), event.NewTerrainID)

	// 验证地形已变更
	terrainID, err := m.GetTerrainID(pos)
	require.NoError(t, err)
	assert.Equal(t, int32(2), terrainID)
}

// TestMap_ChangeTerrain_OutOfBounds 测试坐标越界时地形变更失败。
func TestMap_ChangeTerrain_OutOfBounds(t *testing.T) {
	m := NewMap(1, 100, 100, GridTypeQuad)
	pos := vo.Position{X: 200, Y: 200}

	_, err := m.ChangeTerrain(pos, 2)
	assert.Error(t, err)
}

// TestMap_GetCell_OutOfBounds 测试坐标越界时查询格子失败。
func TestMap_GetCell_OutOfBounds(t *testing.T) {
	m := NewMap(1, 100, 100, GridTypeQuad)

	_, err := m.GetCell(vo.Position{X: -1, Y: 0})
	assert.Error(t, err)
}
