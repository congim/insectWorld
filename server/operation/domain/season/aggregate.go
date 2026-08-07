// Package season 赛季聚合根，维护赛季阶段与状态。
// Season聚合根提供阶段切换与重置能力，重置时跨服务协调。
package season

import (
	"context"
	"fmt"

	operationerr "insectworld/server/operation/domain/errors"
)

// 赛季阶段常量（规范1）。
const (
	PhasePreparation = 1 // 准备阶段
	PhaseWar         = 2 // 战争阶段
	PhaseSettlement  = 3 // 结算阶段
	PhaseEnded       = 4 // 已结束
)

// Season 赛季聚合根，维护赛季阶段与状态。
type Season struct {
	seasonID   int64       // 赛季ID，全局唯一
	phase      int         // 当前阶段：1=准备 2=战争 3=结算 4=已结束
	startTime  int64       // 赛季开始时间戳（毫秒）
	endTime    int64       // 赛季结束时间戳（毫秒），0表示未结束
	scoreBoard *ScoreBoard // 排行榜
}

// NewSeason 创建赛季聚合根实例。
func NewSeason(seasonID int64, startTime int64) *Season {
	return &Season{
		seasonID:   seasonID,
		phase:      PhasePreparation,
		startTime:  startTime,
		scoreBoard: NewScoreBoard(seasonID),
	}
}

// SeasonID 返回赛季ID。
func (s *Season) SeasonID() int64 { return s.seasonID }

// Phase 返回当前阶段。
func (s *Season) Phase() int { return s.phase }

// TransitionPhase 切换赛季阶段。
func (s *Season) TransitionPhase(targetPhase int) (*PhaseChangedEvent, error) {
	if s.phase == PhaseEnded {
		return nil, fmt.Errorf("阶段切换失败，赛季已结束，seasonID=%d: %w", s.seasonID, operationerr.ErrRuleViolation)
	}

	if targetPhase <= s.phase {
		return nil, fmt.Errorf("阶段切换失败，目标阶段非递进，current=%d，target=%d: %w",
			s.phase, targetPhase, operationerr.ErrPhaseTransitionInvalid)
	}

	oldPhase := s.phase
	s.phase = targetPhase

	return &PhaseChangedEvent{
		SeasonID: s.seasonID,
		OldPhase: oldPhase,
		NewPhase: targetPhase,
	}, nil
}

// Reset 重置赛季，按重置范围清理数据并保留指定数据。
func (s *Season) Reset(ctx context.Context, resetScope, preserveList []string) (*SeasonEndedEvent, error) {
	if s.phase == PhaseEnded {
		return nil, fmt.Errorf("赛季重置失败，赛季已结束，seasonID=%d: %w", s.seasonID, operationerr.ErrRuleViolation)
	}

	s.phase = PhaseEnded
	return &SeasonEndedEvent{
		SeasonID:     s.seasonID,
		ResetScope:   resetScope,
		PreserveList: preserveList,
	}, nil
}

// ScoreBoard 排行榜，维护玩家/联盟积分排名。
type ScoreBoard struct {
	seasonID int64           // 赛季ID
	scores   map[int64]int64 // 积分映射，key=目标ID（玩家或联盟），value=积分
}

// NewScoreBoard 创建排行榜实例。
func NewScoreBoard(seasonID int64) *ScoreBoard {
	return &ScoreBoard{
		seasonID: seasonID,
		scores:   make(map[int64]int64),
	}
}

// AddScore 增加积分。
func (sb *ScoreBoard) AddScore(targetID, score int64) {
	sb.scores[targetID] += score
}

// GetScore 查询积分。
func (sb *ScoreBoard) GetScore(targetID int64) int64 {
	return sb.scores[targetID]
}

// PhaseChangedEvent 阶段切换领域事件。
type PhaseChangedEvent struct {
	SeasonID int64 // 赛季ID
	OldPhase int   // 切换前阶段
	NewPhase int   // 切换后阶段
}

// SeasonEndedEvent 赛季结束领域事件。
type SeasonEndedEvent struct {
	SeasonID     int64    // 赛季ID
	ResetScope   []string // 重置范围
	PreserveList []string // 保留数据列表
}

// SeasonRepository Season聚合根仓储接口（规范3）。
type SeasonRepository interface {
	LoadSeason(ctx context.Context, seasonID int64) (*Season, error)
	SaveSeason(ctx context.Context, s *Season) error
}
