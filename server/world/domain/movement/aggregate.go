// Package movement 移动订单聚合根，维护移动路径与状态。
// MovementOrder聚合根提供移动启动、位置更新、跨区域迁移能力。
package movement

import (
	"context"
	"fmt"

	worlderr "insectworld/server/world/domain/errors"
	"insectworld/server/world/domain/vo"
)

// 移动状态常量（规范1），对应移动订单status字段。
const (
	StatusPending   = 1 // 待开始
	StatusMoving    = 2 // 移动中
	StatusArrived   = 3 // 已到达
	StatusBlocked   = 4 // 已阻挡
	StatusMigrating = 5 // 迁移中（跨区域）
)

// MovementOrder 移动订单聚合根，维护移动路径与状态。
type MovementOrder struct {
	orderID     int64         // 移动订单ID，全局唯一，由雪花算法生成
	entityID    int64         // 移动实体ID，对应World中的实体
	path        []vo.Position // 移动路径，坐标序列
	status      int           // 移动状态：1=待开始 2=移动中 3=已到达 4=已阻挡 5=迁移中
	startTime   int64         // 移动开始时间戳（毫秒）
	currentStep int           // 当前路径步序号，从0开始
	formationID string        // 编队队形ID，空表示非编队移动
}

// NewMovementOrder 创建移动订单聚合根实例。
func NewMovementOrder(orderID, entityID int64, path []vo.Position) *MovementOrder {
	return &MovementOrder{
		orderID:     orderID,
		entityID:    entityID,
		path:        path,
		status:      StatusPending,
		currentStep: 0,
	}
}

// OrderID 返回订单ID。
func (m *MovementOrder) OrderID() int64 { return m.orderID }

// EntityID 返回实体ID。
func (m *MovementOrder) EntityID() int64 { return m.entityID }

// Status 返回移动状态。
func (m *MovementOrder) Status() int { return m.status }

// CurrentPosition 返回当前坐标。
func (m *MovementOrder) CurrentPosition() vo.Position {
	if m.currentStep < len(m.path) {
		return m.path[m.currentStep]
	}
	if len(m.path) > 0 {
		return m.path[len(m.path)-1]
	}
	return vo.Position{}
}

// IsMoving 判断是否移动中。
func (m *MovementOrder) IsMoving() bool {
	return m.status == StatusMoving || m.status == StatusMigrating
}

// IsCompleted 判断移动是否已完成。
func (m *MovementOrder) IsCompleted() bool {
	return m.status == StatusArrived || m.status == StatusBlocked
}

// StartMove 启动移动，校验路径非空且状态为待开始。
func (m *MovementOrder) StartMove(startTime int64) error {
	if m.status != StatusPending {
		return fmt.Errorf("移动启动失败，状态非待开始，orderID=%d，status=%d: %w", m.orderID, m.status, worlderr.ErrRuleViolation)
	}
	if len(m.path) == 0 {
		return fmt.Errorf("移动启动失败，路径为空，orderID=%d: %w", m.orderID, worlderr.ErrInvalidParams)
	}
	m.status = StatusMoving
	m.startTime = startTime
	return nil
}

// UpdatePosition 更新位置，移动到路径的下一步。
// 返回移动结果事件（到达/继续移动/阻挡）。
func (m *MovementOrder) UpdatePosition() (*PositionUpdatedEvent, error) {
	if m.status != StatusMoving {
		return nil, fmt.Errorf("位置更新失败，状态非移动中，orderID=%d，status=%d: %w", m.orderID, m.status, worlderr.ErrRuleViolation)
	}

	m.currentStep++
	if m.currentStep >= len(m.path) {
		m.status = StatusArrived
		return &PositionUpdatedEvent{
			OrderID:  m.orderID,
			Position: m.CurrentPosition(),
			Arrived:  true,
		}, nil
	}

	return &PositionUpdatedEvent{
		OrderID:  m.orderID,
		Position: m.CurrentPosition(),
		Arrived:  false,
	}, nil
}

// Block 阻挡移动，移动过程中遇到阻挡时调用。
func (m *MovementOrder) Block() error {
	if m.status != StatusMoving {
		return fmt.Errorf("阻挡失败，状态非移动中，orderID=%d: %w", m.orderID, worlderr.ErrRuleViolation)
	}
	m.status = StatusBlocked
	return nil
}

// Migrate 开始跨区域迁移。
func (m *MovementOrder) Migrate() error {
	if m.status != StatusMoving {
		return fmt.Errorf("迁移失败，状态非移动中，orderID=%d: %w", m.orderID, worlderr.ErrRuleViolation)
	}
	m.status = StatusMigrating
	return nil
}

// FinishMigration 完成跨区域迁移，恢复移动状态。
func (m *MovementOrder) FinishMigration() error {
	if m.status != StatusMigrating {
		return fmt.Errorf("完成迁移失败，状态非迁移中，orderID=%d: %w", m.orderID, worlderr.ErrRuleViolation)
	}
	m.status = StatusMoving
	return nil
}

// SetFormation 设置编队队形ID，用于编队移动。
func (m *MovementOrder) SetFormation(formationID string) {
	m.formationID = formationID
}

// PositionUpdatedEvent 位置更新领域事件。
type PositionUpdatedEvent struct {
	OrderID  int64       // 移动订单ID
	Position vo.Position // 当前坐标
	Arrived  bool        // 是否已到达终点
}

// MovementRepository MovementOrder聚合根仓储接口，在domain层声明（规范3）。
type MovementRepository interface {
	// LoadMovement 加载移动订单
	LoadMovement(ctx context.Context, orderID int64) (*MovementOrder, error)
	// SaveMovement 保存移动订单
	SaveMovement(ctx context.Context, m *MovementOrder) error
	// FindActiveMovement 查询实体的活跃移动订单
	FindActiveMovement(ctx context.Context, entityID int64) (*MovementOrder, error)
}
