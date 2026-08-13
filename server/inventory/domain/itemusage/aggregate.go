// Package itemusage 道具使用订单聚合根，维护道具使用的上下文、效果执行与消耗记录。
// ItemUsage聚合根有独立生命周期（创建→执行→完成），用于追踪道具使用的完整流程。
// 对应spec.md 5.1.7.2节Inventory上下文功能2"道具使用"。
package itemusage

import (
	"context"
	"fmt"

	inventoryerr "insectworld/server/inventory/domain/errors"
	"insectworld/server/inventory/domain/vo"
)

// 使用订单状态常量（规范1）。
const (
	StatusPending    = 1 // 待执行
	StatusExecuting  = 2 // 执行中
	StatusCompleted  = 3 // 已完成
	StatusFailed     = 4 // 执行失败
	StatusRolledBack = 5 // 已回滚（效果执行失败时回滚道具消耗）
)

// ItemUsage 道具使用订单聚合根，维护道具使用的完整流程。
type ItemUsage struct {
	usageID     int64        // 使用订单ID，全局唯一，由雪花算法生成
	playerID    int64        // 玩家ID
	itemID      vo.ItemID    // 道具实例ID
	defID       vo.ItemDefID // 道具定义ID
	count       int64        // 使用数量
	contextType int          // 使用上下文类型：1=手动使用 2=任务自动使用 3=活动自动使用
	status      int          // 使用状态：1=待执行 2=执行中 3=已完成 4=执行失败 5=已回滚
	createTime  int64        // 创建时间戳（毫秒）
	finishTime  int64        // 完成时间戳（毫秒），0表示未完成
	errorMsg    string       // 失败原因，status=4时有效
}

// NewItemUsage 创建道具使用订单聚合根实例。
func NewItemUsage(usageID int64, playerID int64, itemID vo.ItemID, defID vo.ItemDefID, count int64, contextType int, createTime int64) *ItemUsage {
	return &ItemUsage{
		usageID:     usageID,
		playerID:    playerID,
		itemID:      itemID,
		defID:       defID,
		count:       count,
		contextType: contextType,
		status:      StatusPending,
		createTime:  createTime,
	}
}

// UsageID 返回使用订单ID。
func (u *ItemUsage) UsageID() int64 { return u.usageID }

// Status 返回当前状态。
func (u *ItemUsage) Status() int { return u.status }

// StartExecution 开始执行道具效果，状态从待执行转为执行中。
func (u *ItemUsage) StartExecution() error {
	if u.status != StatusPending {
		return fmt.Errorf("道具使用订单状态非待执行，usageID=%d，当前状态=%d: %w",
			u.usageID, u.status, inventoryerr.ErrInvalidParams)
	}
	u.status = StatusExecuting
	return nil
}

// Complete 完成道具使用，状态从执行中转为已完成。
func (u *ItemUsage) Complete(finishTime int64) *UsageCompletedEvent {
	u.status = StatusCompleted
	u.finishTime = finishTime
	return &UsageCompletedEvent{
		UsageID:    u.usageID,
		PlayerID:   u.playerID,
		DefID:      u.defID,
		Count:      u.count,
		FinishTime: finishTime,
	}
}

// Fail 标记道具使用失败，状态从执行中转为执行失败。
func (u *ItemUsage) Fail(errMsg string, finishTime int64) *UsageFailedEvent {
	u.status = StatusFailed
	u.errorMsg = errMsg
	u.finishTime = finishTime
	return &UsageFailedEvent{
		UsageID:    u.usageID,
		PlayerID:   u.playerID,
		DefID:      u.defID,
		ErrorMsg:   errMsg,
		FinishTime: finishTime,
	}
}

// Rollback 回滚道具消耗，状态从执行失败转为已回滚。
// application层收到Rollback事件后，将道具数量加回背包。
func (u *ItemUsage) Rollback() *UsageRolledBackEvent {
	u.status = StatusRolledBack
	return &UsageRolledBackEvent{
		UsageID:  u.usageID,
		PlayerID: u.playerID,
		ItemID:   u.itemID,
		DefID:    u.defID,
		Count:    u.count,
	}
}

// UsageCompletedEvent 道具使用完成事件。
type UsageCompletedEvent struct {
	UsageID    int64        // 使用订单ID
	PlayerID   int64        // 玩家ID
	DefID      vo.ItemDefID // 道具定义ID
	Count      int64        // 使用数量
	FinishTime int64        // 完成时间戳（毫秒）
}

// UsageFailedEvent 道具使用失败事件。
type UsageFailedEvent struct {
	UsageID    int64        // 使用订单ID
	PlayerID   int64        // 玩家ID
	DefID      vo.ItemDefID // 道具定义ID
	ErrorMsg   string       // 失败原因
	FinishTime int64        // 失败时间戳（毫秒）
}

// UsageRolledBackEvent 道具使用回滚事件，application层据此将道具加回背包。
type UsageRolledBackEvent struct {
	UsageID  int64        // 使用订单ID
	PlayerID int64        // 玩家ID
	ItemID   vo.ItemID    // 道具实例ID
	DefID    vo.ItemDefID // 道具定义ID
	Count    int64        // 回滚数量
}

// ItemUsageRepository 道具使用订单仓储接口，在domain层声明（规范3）。
type ItemUsageRepository interface {
	// LoadItemUsage 按使用订单ID加载聚合根
	LoadItemUsage(ctx context.Context, usageID int64) (*ItemUsage, error)
	// SaveItemUsage 保存道具使用订单聚合根
	SaveItemUsage(ctx context.Context, u *ItemUsage) error
}
