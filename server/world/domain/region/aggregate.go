// Package region 区域聚合根，维护区域定义与生命周期状态。
// Region聚合根提供区域创建与销毁能力，销毁前需检查无活跃实体。
package region

import (
	"context"
	"fmt"

	worlderr "insectworld/server/world/domain/errors"
	"insectworld/server/world/domain/vo"
)

// 区域状态常量（规范1）。
const (
	StatusActive    = 1 // 区域活跃
	StatusDestroyed = 2 // 区域已销毁
)

// Region 区域聚合根，维护区域定义与生命周期状态。
type Region struct {
	regionID int64         // 区域ID，全局唯一，由雪花算法生成
	center   vo.Position   // 区域中心坐标
	radius   int32         // 区域半径（格子数）
	cells    []vo.Position // 区域包含的格子坐标列表
	status   int           // 区域状态：1=活跃 2=已销毁（规范8用int枚举）
}

// NewRegion 创建区域聚合根实例。
func NewRegion(regionID int64, center vo.Position, radius int32) *Region {
	return &Region{
		regionID: regionID,
		center:   center,
		radius:   radius,
		cells:    []vo.Position{},
		status:   StatusActive,
	}
}

// RegionID 返回区域ID。
func (r *Region) RegionID() int64 {
	return r.regionID
}

// Center 返回区域中心坐标。
func (r *Region) Center() vo.Position {
	return r.center
}

// Radius 返回区域半径。
func (r *Region) Radius() int32 {
	return r.radius
}

// Status 返回区域状态。
func (r *Region) Status() int {
	return r.status
}

// IsActive 判断区域是否活跃。
func (r *Region) IsActive() bool {
	return r.status == StatusActive
}

// Contains 判断坐标是否在区域内。
func (r *Region) Contains(pos vo.Position) bool {
	for _, cell := range r.cells {
		if cell.Equal(pos) {
			return true
		}
	}
	return false
}

// AddCell 添加格子到区域，用于区域创建时构建格子集合。
func (r *Region) AddCell(pos vo.Position) {
	r.cells = append(r.cells, pos)
}

// CellCount 返回区域格子数量。
func (r *Region) CellCount() int {
	return len(r.cells)
}

// Create 创建区域，校验格子集合非空。
func (r *Region) Create() error {
	if len(r.cells) == 0 {
		return fmt.Errorf("区域创建失败，格子集合为空，regionID=%d: %w", r.regionID, worlderr.ErrInvalidParams)
	}
	if r.status != StatusActive {
		return fmt.Errorf("区域创建失败，区域状态非活跃，regionID=%d，status=%d: %w", r.regionID, r.status, worlderr.ErrRuleViolation)
	}
	return nil
}

// Destroy 销毁区域，需确保区域内无活跃实体。
// hasActiveEntities由application层查询后传入，domain层不直接查询。
func (r *Region) Destroy(hasActiveEntities bool) (*RegionDestroyedEvent, error) {
	if r.status != StatusActive {
		return nil, fmt.Errorf("区域销毁失败，区域状态非活跃，regionID=%d: %w", r.regionID, worlderr.ErrRuleViolation)
	}
	if hasActiveEntities {
		return nil, fmt.Errorf("区域销毁失败，区域内有活跃实体，regionID=%d: %w", r.regionID, worlderr.ErrRegionNotEmpty)
	}
	r.status = StatusDestroyed
	return &RegionDestroyedEvent{RegionID: r.regionID}, nil
}

// RegionCreatedEvent 区域创建领域事件。
type RegionCreatedEvent struct {
	RegionID int64       // 区域ID
	Center   vo.Position // 区域中心坐标
	Radius   int32       // 区域半径
}

// RegionDestroyedEvent 区域销毁领域事件。
type RegionDestroyedEvent struct {
	RegionID int64 // 区域ID
}

// RegionRepository Region聚合根仓储接口，在domain层声明（规范3），infrastructure层实现。
type RegionRepository interface {
	// LoadRegion 加载区域聚合根
	LoadRegion(ctx context.Context, regionID int64) (*Region, error)
	// SaveRegion 保存区域聚合根
	SaveRegion(ctx context.Context, r *Region) error
	// RegionExists 检查区域ID是否已存在
	RegionExists(ctx context.Context, regionID int64) (bool, error)
}
