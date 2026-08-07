// Package gamemap 地图聚合根，维护地图格子状态与地形定义。
// Map聚合根是World Service的核心聚合根，提供坐标边界校验与地形变更能力。
package gamemap

import (
	"fmt"

	worlderr "insectworld/server/world/domain/errors"
	"insectworld/server/world/domain/vo"
)

// 格子类型常量（规范1），对应game.json的grid_type配置。
const (
	GridTypeHex  = 1 // 六边形格子
	GridTypeQuad = 2 // 四边形格子
	GridTypeFree = 3 // 自由格子（无固定形状）
)

// Map 地图聚合根，维护地图格子状态与地形定义。
// 地图尺寸与格子类型在创建时确定，后续不可变更（值对象特性）。
type Map struct {
	mapID    int64                 // 地图ID，全局唯一
	width    int32                 // 地图宽度（格子数）
	height   int32                 // 地图高度（格子数）
	gridType int                   // 格子类型：1=hex 2=quad 3=free，对应game.json配置
	cells    map[vo.Position]*Cell // 格子数据，key=坐标
}

// Cell 单个格子数据，属于Map聚合根的内部状态。
type Cell struct {
	Position  vo.Position // 格子坐标
	TerrainID int32       // 地形类型ID，对应terrains.json配置
	EntityID  int64       // 格子上实体ID，0表示无实体
}

// NewMap 创建地图聚合根实例。
func NewMap(mapID int64, width, height int32, gridType int) *Map {
	return &Map{
		mapID:    mapID,
		width:    width,
		height:   height,
		gridType: gridType,
		cells:    make(map[vo.Position]*Cell),
	}
}

// MapID 返回地图ID。
func (m *Map) MapID() int64 {
	return m.mapID
}

// Width 返回地图宽度。
func (m *Map) Width() int32 {
	return m.width
}

// Height 返回地图高度。
func (m *Map) Height() int32 {
	return m.height
}

// GridType 返回格子类型。
func (m *Map) GridType() int {
	return m.gridType
}

// InBounds 校验坐标是否在地图范围内。
func (m *Map) InBounds(pos vo.Position) bool {
	return pos.X >= 0 && pos.X < m.width && pos.Y >= 0 && pos.Y < m.height
}

// GetCell 查询指定坐标的格子数据。
func (m *Map) GetCell(pos vo.Position) (*Cell, error) {
	if !m.InBounds(pos) {
		return nil, fmt.Errorf("坐标越界，pos=(%d,%d)，地图范围=(%d,%d): %w", pos.X, pos.Y, m.width, m.height, worlderr.ErrOutOfBounds)
	}
	return m.cells[pos], nil
}

// GetTerrainID 查询指定坐标的地形类型ID。
func (m *Map) GetTerrainID(pos vo.Position) (int32, error) {
	cell, err := m.GetCell(pos)
	if err != nil {
		return 0, err
	}
	if cell == nil {
		return 0, nil
	}
	return cell.TerrainID, nil
}

// SetCell 设置格子数据，用于MapInitializer初始化地图。
func (m *Map) SetCell(pos vo.Position, terrainID int32) error {
	if !m.InBounds(pos) {
		return fmt.Errorf("坐标越界，pos=(%d,%d): %w", pos.X, pos.Y, worlderr.ErrOutOfBounds)
	}
	m.cells[pos] = &Cell{
		Position:  pos,
		TerrainID: terrainID,
	}
	return nil
}

// ChangeTerrain 变更指定坐标的地形类型。
// 返回TerrainChangedEvent领域事件，由application层写入Outbox投递。
func (m *Map) ChangeTerrain(pos vo.Position, newTerrainID int32) (*TerrainChangedEvent, error) {
	if !m.InBounds(pos) {
		return nil, fmt.Errorf("地形变更失败，坐标越界，pos=(%d,%d): %w", pos.X, pos.Y, worlderr.ErrOutOfBounds)
	}

	cell, ok := m.cells[pos]
	if !ok || cell == nil {
		return nil, fmt.Errorf("地形变更失败，格子不存在，pos=(%d,%d): %w", pos.X, pos.Y, worlderr.ErrInvalidParams)
	}

	oldTerrainID := cell.TerrainID
	cell.TerrainID = newTerrainID

	return &TerrainChangedEvent{
		MapID:        m.mapID,
		Position:     pos,
		OldTerrainID: oldTerrainID,
		NewTerrainID: newTerrainID,
	}, nil
}

// TerrainChangedEvent 地形变更领域事件。
type TerrainChangedEvent struct {
	MapID        int64       // 地图ID
	Position     vo.Position // 变更坐标
	OldTerrainID int32       // 变更前地形ID
	NewTerrainID int32       // 变更后地形ID
}
