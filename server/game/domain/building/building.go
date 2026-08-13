// Package building 定义玩家建筑聚合及建造状态机。
package building

import (
	"context"
	"fmt"

	gameerr "insectworld/server/game/domain/errors"
)

// Status 是建筑状态枚举。
type Status int32

// 建筑状态常量。
const (
	StatusConstructing Status = 1 // 建造中
	StatusOperational  Status = 2 // 可用
)

// Building 是玩家建筑聚合根。
type Building struct {
	id            int64  // 建筑实例ID，全局唯一
	playerID      int64  // 所属玩家ID
	typeID        string // 建筑类型稳定ID，来源于游戏包
	status        Status // 当前状态：1=建造中，2=可用
	startedAt     int64  // 建造开始时间戳，Unix毫秒
	completeAt    int64  // 可完成时间戳，Unix毫秒
	configVersion string // 建造时绑定的游戏包语义版本
	commandID     string // 建造命令幂等键
}

// NewConstruction 创建建造中的建筑聚合。
func NewConstruction(id int64, playerID int64, typeID string, startedAt int64, completeAt int64, configVersion string, commandID string) (*Building, error) {
	if id <= 0 || playerID <= 0 || typeID == "" || startedAt <= 0 || completeAt <= startedAt || configVersion == "" || commandID == "" {
		return nil, fmt.Errorf("建筑参数非法，buildingID=%d，playerID=%d: %w", id, playerID, gameerr.ErrInvalidCommand)
	}
	return &Building{id: id, playerID: playerID, typeID: typeID, status: StatusConstructing, startedAt: startedAt, completeAt: completeAt, configVersion: configVersion, commandID: commandID}, nil
}

// RestoreBuilding 从可信持久化数据恢复建筑，并重新校验状态与时间不变量。
func RestoreBuilding(id int64, playerID int64, typeID string, status Status, startedAt int64, completeAt int64, configVersion string, commandID string) (*Building, error) {
	aggregate, err := NewConstruction(id, playerID, typeID, startedAt, completeAt, configVersion, commandID)
	if err != nil {
		return nil, err
	}
	if status != StatusConstructing && status != StatusOperational {
		return nil, fmt.Errorf("持久化建筑状态非法，buildingID=%d，status=%d: %w", id, status, gameerr.ErrStateConflict)
	}
	aggregate.status = status
	return aggregate, nil
}

// Complete 在达到完成时间后将建筑切换为可用状态；重复完成保持幂等。
func (b *Building) Complete(nowMs int64) error {
	if b.status == StatusOperational {
		return nil
	}
	if nowMs < b.completeAt {
		return fmt.Errorf("建筑尚未达到完成时间，buildingID=%d，completeAt=%d: %w", b.id, b.completeAt, gameerr.ErrTaskNotReady)
	}
	b.status = StatusOperational
	return nil
}

// ID 返回建筑实例ID。
func (b *Building) ID() int64 { return b.id }

// PlayerID 返回所属玩家ID。
func (b *Building) PlayerID() int64 { return b.playerID }

// TypeID 返回建筑类型稳定ID。
func (b *Building) TypeID() string { return b.typeID }

// Status 返回建筑当前状态。
func (b *Building) Status() Status { return b.status }

// StartedAt 返回建造开始时间戳，单位毫秒。
func (b *Building) StartedAt() int64 { return b.startedAt }

// CompleteAt 返回最早完成时间戳，单位毫秒。
func (b *Building) CompleteAt() int64 { return b.completeAt }

// ConfigVersion 返回建筑绑定的游戏包语义版本。
func (b *Building) ConfigVersion() string { return b.configVersion }

// CommandID 返回建造命令幂等键。
func (b *Building) CommandID() string { return b.commandID }

// Clone 返回与仓储内部状态隔离的建筑副本。
func (b *Building) Clone() *Building {
	copyValue := *b
	return &copyValue
}

// Repository 是玩家建筑仓储，由Growth上下文拥有写权限。
type Repository interface {
	FindByID(ctx context.Context, buildingID int64) (*Building, error)
	FindByCommandID(ctx context.Context, commandID string) (*Building, error)
	SaveIfAbsent(ctx context.Context, building *Building) (*Building, bool, error)
	Save(ctx context.Context, building *Building) error
}
