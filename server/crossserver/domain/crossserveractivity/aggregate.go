// Package crossserveractivity 跨服活动聚合根，维护跨服活动的生命周期与参与区。
// CrossServerActivity聚合根管理跨服活动的创建、开启、结算与关闭。
// 对应spec.md 5.1.7.5节CrossServer上下文功能2"跨服活动"。
package crossserveractivity

import (
	"context"
	"fmt"

	cserr "insectworld/server/crossserver/domain/errors"
)

// 活动状态常量（规范1）。
const (
	StatusCreated = 1 // 已创建（未开始）
	StatusRunning = 2 // 进行中
	StatusSettled = 3 // 已结算
	StatusClosed  = 4 // 已关闭
)

// 活动类型常量（规范1）。
const (
	TypeCrossWar     = 1 // 跨服国战
	TypeEmperorRace  = 2 // 皇城争夺
	TypeResourceRace = 3 // 资源竞速
	TypeSeasonMatch  = 4 // 赛季跨服匹配
)

// CrossServerActivity 跨服活动聚合根。
type CrossServerActivity struct {
	activityID   int64   // 活动ID，全局唯一
	activityType int     // 活动类型：1=跨服国战 2=皇城争夺 3=资源竞速 4=赛季跨服匹配
	zoneIDs      []int64 // 参与区ID列表
	season       int64   // 赛季编号
	status       int     // 活动状态：1=已创建 2=进行中 3=已结算 4=已关闭
	startTime    int64   // 开始时间戳（毫秒）
	endTime      int64   // 结束时间戳（毫秒）
	settleTime   int64   // 结算时间戳（毫秒），0表示未结算
}

// NewCrossServerActivity 创建跨服活动聚合根实例。
func NewCrossServerActivity(activityID int64, activityType int, zoneIDs []int64, season int64, startTime int64, endTime int64) *CrossServerActivity {
	return &CrossServerActivity{
		activityID:   activityID,
		activityType: activityType,
		zoneIDs:      zoneIDs,
		season:       season,
		status:       StatusCreated,
		startTime:    startTime,
		endTime:      endTime,
	}
}

// ActivityID 返回活动ID。
func (a *CrossServerActivity) ActivityID() int64 { return a.activityID }

// Status 返回活动状态。
func (a *CrossServerActivity) Status() int { return a.status }

// Start 开始活动，状态从已创建转为进行中。
func (a *CrossServerActivity) Start(now int64) error {
	if a.status != StatusCreated {
		return fmt.Errorf("活动非已创建状态，activityID=%d，当前状态=%d: %w",
			a.activityID, a.status, cserr.ErrActivityEnded)
	}
	if now < a.startTime {
		return fmt.Errorf("活动未到开始时间，activityID=%d: %w", a.activityID, cserr.ErrInvalidParams)
	}
	a.status = StatusRunning
	return nil
}

// Settle 结算活动，状态从进行中转为已结算。
func (a *CrossServerActivity) Settle(settleTime int64) (*ActivitySettledEvent, error) {
	if a.status != StatusRunning {
		return nil, fmt.Errorf("活动非进行中状态，activityID=%d: %w", a.activityID, cserr.ErrActivityEnded)
	}
	a.status = StatusSettled
	a.settleTime = settleTime
	return &ActivitySettledEvent{
		ActivityID:   a.activityID,
		ActivityType: a.activityType,
		Season:       a.season,
		SettleTime:   settleTime,
	}, nil
}

// Close 关闭活动，状态从已结算转为已关闭。
func (a *CrossServerActivity) Close() {
	a.status = StatusClosed
}

// ActivitySettledEvent 活动结算事件。
type ActivitySettledEvent struct {
	ActivityID   int64 // 活动ID
	ActivityType int   // 活动类型
	Season       int64 // 赛季编号
	SettleTime   int64 // 结算时间戳（毫秒）
}

// CrossServerActivityRepository 跨服活动仓储接口，在domain层声明（规范3）。
type CrossServerActivityRepository interface {
	// LoadActivity 加载跨服活动聚合根
	LoadActivity(ctx context.Context, activityID int64) (*CrossServerActivity, error)
	// SaveActivity 保存跨服活动聚合根
	SaveActivity(ctx context.Context, a *CrossServerActivity) error
}
