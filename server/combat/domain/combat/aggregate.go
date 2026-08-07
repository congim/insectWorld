// Package combat 战斗聚合根，维护战斗状态与轮次执行。
// Combat聚合根是Combat Service的核心聚合根，提供战斗启动、轮次执行、撤退与结束能力。
package combat

import (
	"context"
	"fmt"

	combaterr "insectworld/server/combat/domain/errors"
)

// 战斗状态常量（规范1）。
const (
	StatusInProgress = 1 // 进行中
	StatusEnded      = 2 // 已结束
	StatusRetreated  = 3 // 已撤退
)

// 战斗结果常量（规范1）。
const (
	ResultAttackerWin = 1 // 攻击方胜
	ResultDefenderWin = 2 // 防守方胜
	ResultDraw        = 3 // 平局
)

// Combat 战斗聚合根，维护战斗状态与轮次执行。
type Combat struct {
	combatID     int64   // 战斗ID，全局唯一，由雪花算法生成
	combatType   int     // 战斗类型：1=野战 2=攻城 3=城战，由combat.json配置驱动
	status       int     // 战斗状态：1=进行中 2=已结束 3=已撤退
	currentRound int     // 当前轮次，从1开始，超过maxRounds强制结束
	maxRounds    int     // 最大轮数，从ConfigQueryAPI.GetMaxRounds查询，不硬编码
	attackerIDs  []int64 // 攻击方实体ID列表
	defenderIDs  []int64 // 防守方实体ID列表
	formationID  int32   // 阵型ID，0表示无阵型
	startTime    int64   // 战斗开始时间戳（毫秒）
}

// NewCombat 创建战斗聚合根实例。
func NewCombat(combatID int64, combatType, maxRounds int, attackerIDs, defenderIDs []int64, startTime int64) *Combat {
	return &Combat{
		combatID:     combatID,
		combatType:   combatType,
		status:       StatusInProgress,
		currentRound: 0,
		maxRounds:    maxRounds,
		attackerIDs:  attackerIDs,
		defenderIDs:  defenderIDs,
		startTime:    startTime,
	}
}

// CombatID 返回战斗ID。
func (c *Combat) CombatID() int64 { return c.combatID }

// CombatType 返回战斗类型。
func (c *Combat) CombatType() int { return c.combatType }

// Status 返回战斗状态。
func (c *Combat) Status() int { return c.status }

// CurrentRound 返回当前轮次。
func (c *Combat) CurrentRound() int { return c.currentRound }

// MaxRounds 返回最大轮数。
func (c *Combat) MaxRounds() int { return c.maxRounds }

// IsLongCombat 判断是否长时战斗（需要每轮刷新属性快照）。
// TODO 后续从配置查询长时战斗判定规则，当前以轮数>5为判定标准
func (c *Combat) IsLongCombat() bool {
	return c.maxRounds > 5
}

// IsInProgress 判断战斗是否进行中。
func (c *Combat) IsInProgress() bool {
	return c.status == StatusInProgress
}

// ExecuteRound 执行一轮战斗，返回轮次完成事件。
// 阵型效果与技能释放由application层编排调用，聚合根只负责轮次计数与状态管理。
func (c *Combat) ExecuteRound() (*RoundCompletedEvent, error) {
	if c.status != StatusInProgress {
		return nil, fmt.Errorf("轮次执行失败，战斗状态非进行中，combatID=%d，status=%d: %w", c.combatID, c.status, combaterr.ErrRuleViolation)
	}

	c.currentRound++

	event := &RoundCompletedEvent{
		CombatID:    c.combatID,
		Round:       c.currentRound,
		AttackerIDs: c.attackerIDs,
		DefenderIDs: c.defenderIDs,
	}
	return event, nil
}

// CheckMaxRounds 校验是否轮数超限，超限则强制平局结束。
// 返回true表示轮数超限需强制结束。
func (c *Combat) CheckMaxRounds() bool {
	return c.currentRound >= c.maxRounds
}

// End 结束战斗，设置战斗结果。
func (c *Combat) End(result int) (*CombatEndedEvent, error) {
	if c.status != StatusInProgress {
		return nil, fmt.Errorf("战斗结束失败，状态非进行中，combatID=%d: %w", c.combatID, combaterr.ErrRuleViolation)
	}
	c.status = StatusEnded
	return &CombatEndedEvent{
		CombatID:    c.combatID,
		Result:      result,
		TotalRounds: c.currentRound,
	}, nil
}

// Retreat 撤退，设置战斗状态为撤退。
func (c *Combat) Retreat() (*CombatEndedEvent, error) {
	if c.status != StatusInProgress {
		return nil, fmt.Errorf("撤退失败，状态非进行中，combatID=%d: %w", c.combatID, combaterr.ErrRuleViolation)
	}
	c.status = StatusRetreated
	return &CombatEndedEvent{
		CombatID:    c.combatID,
		Result:      ResultDefenderWin,
		TotalRounds: c.currentRound,
	}, nil
}

// SetFormation 设置阵型ID。
func (c *Combat) SetFormation(formationID int32) {
	c.formationID = formationID
}

// RoundCompletedEvent 轮次完成领域事件。
type RoundCompletedEvent struct {
	CombatID    int64   // 战斗ID
	Round       int     // 轮次序号
	AttackerIDs []int64 // 攻击方实体ID列表
	DefenderIDs []int64 // 防守方实体ID列表
}

// CombatEndedEvent 战斗结束领域事件。
type CombatEndedEvent struct {
	CombatID    int64 // 战斗ID
	Result      int   // 战斗结果：1=攻击方胜 2=防守方胜 3=平局
	TotalRounds int   // 总轮次
}

// CombatRepository Combat聚合根仓储接口，在domain层声明（规范3）。
type CombatRepository interface {
	// LoadCombat 加载战斗聚合根
	LoadCombat(ctx context.Context, combatID int64) (*Combat, error)
	// SaveCombat 保存战斗聚合根
	SaveCombat(ctx context.Context, c *Combat) error
}
