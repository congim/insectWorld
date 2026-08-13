// Package battlefield 限时战场聚合根，维护战场状态、参与方与结算结果。
// Battlefield聚合根独立于主世界，是限时玩法（国战/皇城争夺/资源竞速）的载体。
// 对应spec.md 5.1.7.4节Match上下文功能2-3"战场创建与结算"。
package battlefield

import (
	"context"
	"fmt"

	matcherr "insectworld/server/match/domain/errors"
)

// 战场状态常量（规范1）。
const (
	StatusPreparing = 1 // 准备中（等待玩家加入）
	StatusRunning   = 2 // 进行中
	StatusSettled   = 3 // 已结算
	StatusClosed    = 4 // 已关闭
)

// Battlefield 限时战场聚合根。
type Battlefield struct {
	battlefieldID   int64       // 战场ID，全局唯一
	templateID      int64       // 战场模板ID，对应battlefields.json配置
	zoneID          int64       // 跨服区ID，跨服战场所属的跨服区
	participants    []int64     // 参与方ID列表（玩家ID或联盟ID）
	maxParticipants int         // 最大参与方数，从配置查询
	status          int         // 战场状态：1=准备中 2=进行中 3=已结算 4=已关闭
	startTime       int64       // 开始时间戳（毫秒）
	endTime         int64       // 结束时间戳（毫秒），0表示未结束
	settlement      *Settlement // 结算结果，status=3时有效
}

// Settlement 战场结算结果值对象。
type Settlement struct {
	WinnerIDs []int64         // 胜方ID列表
	LoserIDs  []int64         // 败方ID列表
	Scores    map[int64]int64 // 各参与方得分，key=参与方ID，value=得分
	Rewards   map[int64]int64 // 各参与方奖励配置ID，key=参与方ID，value=奖励配置ID
}

// NewBattlefield 创建战场聚合根实例。
func NewBattlefield(battlefieldID int64, templateID int64, zoneID int64, maxParticipants int, startTime int64) *Battlefield {
	return &Battlefield{
		battlefieldID:   battlefieldID,
		templateID:      templateID,
		zoneID:          zoneID,
		maxParticipants: maxParticipants,
		status:          StatusPreparing,
		startTime:       startTime,
	}
}

// BattlefieldID 返回战场ID。
func (b *Battlefield) BattlefieldID() int64 { return b.battlefieldID }

// Status 返回战场状态。
func (b *Battlefield) Status() int { return b.status }

// AddParticipant 添加参与方。
func (b *Battlefield) AddParticipant(participantID int64) error {
	if b.status != StatusPreparing {
		return fmt.Errorf("战场非准备中状态，battlefieldID=%d: %w", b.battlefieldID, matcherr.ErrBattlefieldEnded)
	}
	if len(b.participants) >= b.maxParticipants {
		return fmt.Errorf("战场已满，battlefieldID=%d: %w", b.battlefieldID, matcherr.ErrBattlefieldFull)
	}
	b.participants = append(b.participants, participantID)
	return nil
}

// Start 开始战场，状态从准备中转为进行中。
func (b *Battlefield) Start() error {
	if b.status != StatusPreparing {
		return fmt.Errorf("战场非准备中状态，battlefieldID=%d: %w", b.battlefieldID, matcherr.ErrBattlefieldEnded)
	}
	b.status = StatusRunning
	return nil
}

// Settle 战场结算，状态从进行中转为已结算。
func (b *Battlefield) Settle(settlement *Settlement, endTime int64) (*BattlefieldEndedEvent, error) {
	if b.status != StatusRunning {
		return nil, fmt.Errorf("战场非进行中状态，battlefieldID=%d: %w", b.battlefieldID, matcherr.ErrBattlefieldEnded)
	}

	b.status = StatusSettled
	b.endTime = endTime
	b.settlement = settlement

	return &BattlefieldEndedEvent{
		BattlefieldID: b.battlefieldID,
		TemplateID:    b.templateID,
		Settlement:    settlement,
		EndTime:       endTime,
	}, nil
}

// Close 关闭战场，状态从已结算转为已关闭。
func (b *Battlefield) Close() {
	b.status = StatusClosed
}

// BattlefieldEndedEvent 战场结束事件。
type BattlefieldEndedEvent struct {
	BattlefieldID int64       // 战场ID
	TemplateID    int64       // 战场模板ID
	Settlement    *Settlement // 结算结果
	EndTime       int64       // 结束时间戳（毫秒）
}

// BattlefieldRepository 战场聚合根仓储接口，在domain层声明（规范3）。
type BattlefieldRepository interface {
	// LoadBattlefield 加载战场聚合根
	LoadBattlefield(ctx context.Context, battlefieldID int64) (*Battlefield, error)
	// SaveBattlefield 保存战场聚合根
	SaveBattlefield(ctx context.Context, b *Battlefield) error
}
