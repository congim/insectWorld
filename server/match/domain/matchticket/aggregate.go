// Package matchticket 匹配票聚合根，维护匹配队列、匹配条件与等待状态。
// MatchTicket聚合根是跨服匹配的核心，按匹配池分片。
// 对应spec.md 5.1.7.4节Match上下文功能1"跨服匹配"。
package matchticket

import (
	"context"
	"fmt"

	matcherr "insectworld/server/match/domain/errors"
)

// 匹配票状态常量（规范1）。
const (
	StatusWaiting   = 1 // 等待匹配中
	StatusMatched   = 2 // 匹配成功
	StatusTimeout   = 3 // 匹配超时
	StatusCancelled = 4 // 已取消
)

// 匹配主体类型常量（规范1）。
const (
	SubjectPlayer   = 1 // 玩家匹配
	SubjectAlliance = 2 // 联盟匹配
)

// MatchTicket 匹配票聚合根，维护匹配队列与等待状态。
type MatchTicket struct {
	ticketID      int64 // 匹配票ID，全局唯一
	poolID        int64 // 匹配池ID，对应match.json的匹配池定义
	subjectType   int   // 匹配主体类型：1=玩家 2=联盟
	subjectID     int64 // 匹配主体ID（玩家ID或联盟ID）
	tier          int32 // 段位，用于ELO/段位匹配
	score         int64 // 匹配评分（战力/ELO值），用于匹配范围计算
	status        int   // 匹配状态：1=等待中 2=匹配成功 3=超时 4=已取消
	createTime    int64 // 创建时间戳（毫秒）
	matchTime     int64 // 匹配成功时间戳（毫秒），0表示未匹配
	battlefieldID int64 // 匹配成功后分配的战场ID，0表示未分配
}

// NewMatchTicket 创建匹配票聚合根实例。
func NewMatchTicket(ticketID int64, poolID int64, subjectType int, subjectID int64, tier int32, score int64, createTime int64) *MatchTicket {
	return &MatchTicket{
		ticketID:    ticketID,
		poolID:      poolID,
		subjectType: subjectType,
		subjectID:   subjectID,
		tier:        tier,
		score:       score,
		status:      StatusWaiting,
		createTime:  createTime,
	}
}

// TicketID 返回匹配票ID。
func (t *MatchTicket) TicketID() int64 { return t.ticketID }

// Status 返回匹配状态。
func (t *MatchTicket) Status() int { return t.status }

// Match 匹配成功，分配战场。
func (t *MatchTicket) Match(battlefieldID int64, matchTime int64) (*MatchSucceededEvent, error) {
	if t.status != StatusWaiting {
		return nil, fmt.Errorf("匹配票状态非等待中，ticketID=%d，当前状态=%d: %w",
			t.ticketID, t.status, matcherr.ErrInvalidParams)
	}

	t.status = StatusMatched
	t.matchTime = matchTime
	t.battlefieldID = battlefieldID

	return &MatchSucceededEvent{
		TicketID:      t.ticketID,
		PoolID:        t.poolID,
		SubjectType:   t.subjectType,
		SubjectID:     t.subjectID,
		BattlefieldID: battlefieldID,
		MatchTime:     matchTime,
	}, nil
}

// Timeout 匹配超时。
func (t *MatchTicket) Timeout(timeoutTime int64) *MatchTimeoutEvent {
	t.status = StatusTimeout
	return &MatchTimeoutEvent{
		TicketID:     t.ticketID,
		SubjectType:  t.subjectType,
		SubjectID:    t.subjectID,
		WaitDuration: timeoutTime - t.createTime,
	}
}

// Cancel 取消匹配。
func (t *MatchTicket) Cancel() error {
	if t.status != StatusWaiting {
		return fmt.Errorf("匹配票状态非等待中，ticketID=%d: %w", t.ticketID, matcherr.ErrInvalidParams)
	}
	t.status = StatusCancelled
	return nil
}

// MatchSucceededEvent 匹配成功事件。
type MatchSucceededEvent struct {
	TicketID      int64 // 匹配票ID
	PoolID        int64 // 匹配池ID
	SubjectType   int   // 匹配主体类型
	SubjectID     int64 // 匹配主体ID
	BattlefieldID int64 // 分配的战场ID
	MatchTime     int64 // 匹配成功时间戳（毫秒）
}

// MatchTimeoutEvent 匹配超时事件。
type MatchTimeoutEvent struct {
	TicketID     int64 // 匹配票ID
	SubjectType  int   // 匹配主体类型
	SubjectID    int64 // 匹配主体ID
	WaitDuration int64 // 等待时长（毫秒）
}

// MatchTicketRepository 匹配票聚合根仓储接口，在domain层声明（规范3）。
type MatchTicketRepository interface {
	// LoadMatchTicket 加载匹配票聚合根
	LoadMatchTicket(ctx context.Context, ticketID int64) (*MatchTicket, error)
	// SaveMatchTicket 保存匹配票聚合根
	SaveMatchTicket(ctx context.Context, t *MatchTicket) error
}
