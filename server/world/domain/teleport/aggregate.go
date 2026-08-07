// Package teleport 传送聚合根，维护传送条件校验、冷却记录与资源消耗。
// 传送类型（增援/迁城/传送门）由movement.json配置驱动，聚合根方法从ConfigQueryAPI查询传送规则。
package teleport

import (
	"context"
	"fmt"

	worlderr "insectworld/server/world/domain/errors"
	"insectworld/server/world/domain/vo"
	"insectworld/server/shared/pkg/config"
)

// 传送类型常量（规范1），对应movement.json的传送类型配置。
const (
	TeleportTypeReinforce = 1 // 增援传送
	TeleportTypeRelocate  = 2 // 迁城传送
	TeleportTypePortal    = 3 // 传送门传送
)

// 传送状态常量（规范1）。
const (
	StatusPending    = 1 // 待校验
	StatusTeleported = 2 // 已传送
	StatusCancelled  = 3 // 已取消
)

// TeleportAggregate 传送聚合根，维护传送条件校验、冷却记录与资源消耗。
type TeleportAggregate struct {
	teleportID    int64           // 传送订单ID，全局唯一，由雪花算法生成
	entityID      int64           // 传送实体ID
	teleportType  int             // 传送类型：1=增援 2=迁城 3=传送门，由movement.json配置驱动
	targetCoord   vo.Position     // 目标坐标
	status        int             // 传送状态：1=待校验 2=已传送 3=已取消
	cooldownEndAt int64           // 冷却结束时间戳（毫秒），0表示无冷却
	resourceCost  map[int64]int64 // 资源消耗，key=资源类型ID，value=消耗数量
	createdAt     int64           // 创建时间戳（毫秒）
}

// NewTeleportAggregate 创建传送聚合根实例。
func NewTeleportAggregate(teleportID, entityID int64, teleportType int, targetCoord vo.Position, createdAt int64) *TeleportAggregate {
	return &TeleportAggregate{
		teleportID:   teleportID,
		entityID:     entityID,
		teleportType: teleportType,
		targetCoord:  targetCoord,
		status:       StatusPending,
		createdAt:    createdAt,
		resourceCost: make(map[int64]int64),
	}
}

// TeleportID 返回传送订单ID。
func (t *TeleportAggregate) TeleportID() int64 { return t.teleportID }

// EntityID 返回实体ID。
func (t *TeleportAggregate) EntityID() int64 { return t.entityID }

// TeleportType 返回传送类型。
func (t *TeleportAggregate) TeleportType() int { return t.teleportType }

// Status 返回传送状态。
func (t *TeleportAggregate) Status() int { return t.status }

// TargetCoord 返回目标坐标。
func (t *TeleportAggregate) TargetCoord() vo.Position { return t.targetCoord }

// CheckCooldown 校验传送冷却是否已过期。
// now为当前时间戳（毫秒），返回true表示冷却已过可以传送。
func (t *TeleportAggregate) CheckCooldown(now int64) bool {
	return now >= t.cooldownEndAt
}

// SetCooldown 设置冷却结束时间。
func (t *TeleportAggregate) SetCooldown(cooldownEndAt int64) {
	t.cooldownEndAt = cooldownEndAt
}

// SetResourceCost 设置资源消耗。
func (t *TeleportAggregate) SetResourceCost(cost map[int64]int64) {
	t.resourceCost = cost
}

// Teleport 执行传送，校验冷却与资源消耗后变更状态。
// now为当前时间戳（毫秒），hasSufficientResource为资源是否充足（由application层查询后传入）。
// isAllyTerritory为目标是否同盟领土（由application层gRPC查询Social后传入）。
func (t *TeleportAggregate) Teleport(ctx context.Context, now int64, hasSufficientResource, isAllyTerritory bool) (*TeleportCompletedEvent, error) {
	if t.status != StatusPending {
		return nil, fmt.Errorf("传送失败，状态非待校验，teleportID=%d，status=%d: %w", t.teleportID, t.status, worlderr.ErrRuleViolation)
	}

	if !t.CheckCooldown(now) {
		remaining := t.cooldownEndAt - now
		return nil, fmt.Errorf("传送冷却未过，剩余冷却时间%dms: %w", remaining, worlderr.ErrCooldownActive)
	}

	if !hasSufficientResource {
		return nil, fmt.Errorf("传送资源不足，teleportID=%d: %w", t.teleportID, worlderr.ErrResourceInsufficient)
	}

	if !isAllyTerritory {
		return nil, fmt.Errorf("传送目标非同盟领土，teleportID=%d: %w", t.teleportID, worlderr.ErrRuleViolation)
	}

	t.status = StatusTeleported

	return &TeleportCompletedEvent{
		TeleportID:   t.teleportID,
		EntityID:     t.entityID,
		TeleportType: t.teleportType,
		TargetCoord:  t.targetCoord,
	}, nil
}

// Cancel 取消传送。
func (t *TeleportAggregate) Cancel() error {
	if t.status != StatusPending {
		return fmt.Errorf("取消传送失败，状态非待校验，teleportID=%d: %w", t.teleportID, worlderr.ErrRuleViolation)
	}
	t.status = StatusCancelled
	return nil
}

// TeleportCompletedEvent 传送完成领域事件。
type TeleportCompletedEvent struct {
	TeleportID   int64       // 传送订单ID
	EntityID     int64       // 传送实体ID
	TeleportType int         // 传送类型
	TargetCoord  vo.Position // 目标坐标
}

// TeleportRepository TeleportAggregate仓储接口，在domain层声明（规范3）。
type TeleportRepository interface {
	// SaveTeleport 保存传送记录
	SaveTeleport(ctx context.Context, t *TeleportAggregate) error
	// LoadTeleport 加载传送记录
	LoadTeleport(ctx context.Context, teleportID int64) (*TeleportAggregate, error)
	// FindLatestTeleport 查询实体最近的传送记录（用于冷却校验）
	FindLatestTeleport(ctx context.Context, entityID int64, teleportType int) (*TeleportAggregate, error)
}

// TeleportConfigQuery 传送配置查询接口，封装ConfigQueryAPI的传送相关查询。
// domain层通过此接口查询传送规则，不直接依赖infrastructure（规范3）。
type TeleportConfigQuery interface {
	// GetMovementType 查询移动类型配置（含传送规则）
	GetMovementType(ctx context.Context, typeID string) *config.MovementTypeConfig
}
